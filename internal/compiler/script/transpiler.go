package script

import (
	"fmt"
	"strings"

	"gmx/internal/compiler/ast"
	"gmx/internal/compiler/utils"
)

// SourceMap tracks line mappings from generated Go code to original GMX Script
type SourceMap struct {
	Entries []SourceMapEntry
}

type SourceMapEntry struct {
	GoLine  int
	GmxLine int
	GmxFile string
}

// TranspileResult holds the output of transpilation
type TranspileResult struct {
	GoCode    string
	SourceMap *SourceMap
	Errors    []string
}

type Transpiler struct {
	buf          strings.Builder
	sourceMap    *SourceMap
	goLine       int      // current line in generated Go
	indent       int      // indentation level
	models       []string // known model names for ORM method detection
	errDeclared  bool     // tracks if err variable has been declared in current scope
	varTypes     map[string]string // tracks variable types for instance method detection
	currentFunc  string   // current function name for context
}

func NewTranspiler(modelNames []string) *Transpiler {
	return &Transpiler{
		sourceMap: &SourceMap{
			Entries: []SourceMapEntry{},
		},
		models:   modelNames,
		varTypes: make(map[string]string),
	}
}

// Transpile converts all functions in a ScriptBlock to Go code
func Transpile(script *ast.ScriptBlock, modelNames []string) *TranspileResult {
	t := NewTranspiler(modelNames)
	result := &TranspileResult{
		SourceMap: t.sourceMap,
		Errors:    []string{},
	}

	// Generate ORM helpers first
	t.genORMHelpers()

	// Generate GMXContext struct
	t.genGMXContext()

	// Generate renderFragment helper
	t.genRenderFragment()

	// Transpile each function
	for _, fn := range script.Funcs {
		t.TranspileFunc(fn)
		t.emit("\n")
	}

	result.GoCode = t.buf.String()
	return result
}

// TranspileFunc converts a single FuncDecl to Go code
func (t *Transpiler) TranspileFunc(fn *ast.FuncDecl) string {
	t.currentFunc = fn.Name
	t.errDeclared = false
	t.varTypes = make(map[string]string) // reset for new function

	// Generate function signature
	// GMX: func toggleTask(id: uuid) error
	// Go:  func toggleTask(ctx *GMXContext, id string) error
	t.emitLineComment(fn.Line)
	t.emit("func %s(ctx *GMXContext", fn.Name)

	// Add parameters
	for _, param := range fn.Params {
		t.emit(", %s %s", param.Name, t.transpileType(param.Type))
		// Track parameter types
		if t.isModelType(param.Type) {
			t.varTypes[param.Name] = param.Type
		}
	}

	// Emit return type
	returnType := fn.ReturnType
	if returnType == "" {
		returnType = "error"
	}
	goReturnType := t.transpileType(returnType)
	t.emit(") %s {\n", goReturnType)
	t.indent++

	// Transpile function body
	for _, stmt := range fn.Body {
		t.transpileStmt(stmt)
	}

	// Ensure function returns (in case no explicit return)
	if !t.endsWithReturn(fn.Body) {
		t.emitIndent()
		if goReturnType == "error" {
			t.emit("return nil\n")
		} else {
			t.emit("var zero %s\n", goReturnType)
			t.emitIndent()
			t.emit("return zero\n")
		}
	}

	t.indent--
	t.emit("}\n")

	return t.buf.String()
}

func (t *Transpiler) transpileStmt(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		if s.Const {
			t.transpileConstStmt(s)
		} else {
			t.transpileLetStmt(s)
		}
	case *ast.ReturnStmt:
		t.transpileReturnStmt(s)
	case *ast.IfStmt:
		t.transpileIfStmt(s)
	case *ast.ExprStmt:
		t.transpileExprStmt(s)
	case *ast.AssignStmt:
		t.transpileAssignStmt(s)
	default:
		t.emitIndent()
		t.emit("// unknown statement type: %T\n", stmt)
	}
}

func (t *Transpiler) transpileLetStmt(stmt *ast.LetStmt) {
	t.emitIndent()
	t.emitLineComment(stmt.Line)

	// Check if value is a try expression
	if tryExpr, ok := stmt.Value.(*ast.TryExpr); ok {
		// let x = try expr -> x, err := expr; if err != nil { return err }
		if t.errDeclared {
			t.emit("%s, err = %s\n", stmt.Name, t.transpileExpr(tryExpr.Expr))
		} else {
			t.emit("%s, err := %s\n", stmt.Name, t.transpileExpr(tryExpr.Expr))
			t.errDeclared = true
		}
		t.emitIndent()
		t.emit("if err != nil {\n")
		t.indent++
		t.emitIndent()
		t.emit("return err\n")
		t.indent--
		t.emitIndent()
		t.emit("}\n")

		// Track type if it's a model
		t.trackVarType(stmt.Name, tryExpr.Expr)
	} else {
		// let x = expr -> x := expr
		t.emit("%s := %s\n", stmt.Name, t.transpileExpr(stmt.Value))
		t.trackVarType(stmt.Name, stmt.Value)
	}
}

func (t *Transpiler) transpileConstStmt(stmt *ast.LetStmt) {
	t.emitIndent()
	t.emitLineComment(stmt.Line)
	t.emit("%s := %s\n", stmt.Name, t.transpileExpr(stmt.Value))
	t.trackVarType(stmt.Name, stmt.Value)
}

func (t *Transpiler) transpileReturnStmt(stmt *ast.ReturnStmt) {
	t.emitIndent()
	t.emitLineComment(stmt.Line)

	if stmt.Value == nil {
		t.emit("return nil\n")
		return
	}

	// Check for render() expression
	if renderExpr, ok := stmt.Value.(*ast.RenderExpr); ok {
		t.transpileRenderExpr(renderExpr)
		t.emit("return nil\n")
		return
	}

	// Check for render() call (legacy - for backward compatibility)
	if call, ok := stmt.Value.(*ast.CallExpr); ok {
		if ident, ok := call.Function.(*ast.Ident); ok && ident.Name == "render" {
			t.transpileRenderCall(call)
			t.emit("return nil\n")
			return
		}
	}

	t.emit("return %s\n", t.transpileExpr(stmt.Value))
}

func (t *Transpiler) transpileIfStmt(stmt *ast.IfStmt) {
	t.emitIndent()
	t.emitLineComment(stmt.Line)
	t.emit("if %s {\n", t.transpileExpr(stmt.Condition))
	t.indent++
	for _, s := range stmt.Consequence {
		t.transpileStmt(s)
	}
	t.indent--

	if stmt.Alternative != nil && len(stmt.Alternative) > 0 {
		t.emitIndent()
		t.emit("} else {\n")
		t.indent++
		for _, s := range stmt.Alternative {
			t.transpileStmt(s)
		}
		t.indent--
	}

	t.emitIndent()
	t.emit("}\n")
}

func (t *Transpiler) transpileExprStmt(stmt *ast.ExprStmt) {
	t.emitIndent()
	t.emitLineComment(stmt.Line)

	// Check if it's a try expression used as a statement
	if tryExpr, ok := stmt.Expr.(*ast.TryExpr); ok {
		// try expr -> if err := expr; err != nil { return err }
		t.emit("if err := %s; err != nil {\n", t.transpileExpr(tryExpr.Expr))
		t.indent++
		t.emitIndent()
		t.emit("return err\n")
		t.indent--
		t.emitIndent()
		t.emit("}\n")
		t.errDeclared = true
	} else {
		t.emit("%s\n", t.transpileExpr(stmt.Expr))
	}
}

func (t *Transpiler) transpileAssignStmt(stmt *ast.AssignStmt) {
	t.emitIndent()
	t.emitLineComment(stmt.Line)
	t.emit("%s = %s\n", t.transpileExpr(stmt.Target), t.transpileExpr(stmt.Value))
}

func (t *Transpiler) transpileExpr(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.IntLit:
		return e.Value
	case *ast.FloatLit:
		return e.Value
	case *ast.StringLit:
		// Check for string interpolation
		if len(e.Parts) > 0 {
			return t.transpileStringInterpolationParts(e.Parts)
		}
		return fmt.Sprintf("%q", e.Value)
	case *ast.BoolLit:
		return fmt.Sprintf("%t", e.Value)
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", t.transpileExpr(e.Left), e.Op, t.transpileExpr(e.Right))
	case *ast.UnaryExpr:
		return fmt.Sprintf("%s%s", e.Op, t.transpileExpr(e.Operand))
	case *ast.CallExpr:
		return t.transpileCallExpr(e)
	case *ast.MemberExpr:
		return t.transpileMemberExpr(e)
	case *ast.StructLit:
		return t.transpileStructLiteral(e)
	case *ast.TryExpr:
		// Bare try expression (not in let/const)
		return t.transpileExpr(e.Expr)
	case *ast.ErrorExpr:
		return t.transpileErrorExpr(e)
	case *ast.CtxExpr:
		return fmt.Sprintf("ctx.%s", utils.Capitalize(e.Field))
	case *ast.RenderExpr:
		// render() as expression (shouldn't happen, but handle it)
		return "nil /* render() should be used in return statement */"
	default:
		return fmt.Sprintf("/* unknown expr: %T */", expr)
	}
}

func (t *Transpiler) transpileCallExpr(expr *ast.CallExpr) string {
	// Check for error() builtin
	if ident, ok := expr.Function.(*ast.Ident); ok && ident.Name == "error" {
		if len(expr.Args) == 1 {
			if strLit, ok := expr.Args[0].(*ast.StringLit); ok {
				return fmt.Sprintf("fmt.Errorf(%q)", strLit.Value)
			}
			return fmt.Sprintf("fmt.Errorf(\"%%v\", %s)", t.transpileExpr(expr.Args[0]))
		}
	}

	// Check for Model.find(), Model.all() static methods
	if member, ok := expr.Function.(*ast.MemberExpr); ok {
		if ident, ok := member.Object.(*ast.Ident); ok {
			modelName := ident.Name
			methodName := member.Property

			if t.isModelType(modelName) {
				// Static model method
				switch methodName {
				case "find":
					if len(expr.Args) == 1 {
						return fmt.Sprintf("%sFind(ctx.DB, %s)", modelName, t.transpileExpr(expr.Args[0]))
					}
				case "all":
					return fmt.Sprintf("%sAll(ctx.DB)", modelName)
				}
			} else {
				// Instance method - check if variable is a model instance
				if varType, ok := t.varTypes[modelName]; ok && t.isModelType(varType) {
					switch methodName {
					case "save":
						return fmt.Sprintf("%sSave(ctx.DB, %s)", varType, modelName)
					case "delete":
						return fmt.Sprintf("%sDelete(ctx.DB, %s)", varType, modelName)
					}
				}
			}
		}
	}

	// Regular function call
	var args []string
	for _, arg := range expr.Args {
		args = append(args, t.transpileExpr(arg))
	}
	return fmt.Sprintf("%s(%s)", t.transpileExpr(expr.Function), strings.Join(args, ", "))
}

func (t *Transpiler) transpileMemberExpr(expr *ast.MemberExpr) string {
	// Check for ctx.tenant, ctx.user
	if ident, ok := expr.Object.(*ast.Ident); ok && ident.Name == "ctx" {
		switch expr.Property {
		case "tenant":
			return "ctx.Tenant"
		case "user":
			return "ctx.User"
		}
	}

	// Property access on model instances - capitalize property name
	objStr := t.transpileExpr(expr.Object)
	propStr := utils.ToPascalCase(expr.Property)
	return fmt.Sprintf("%s.%s", objStr, propStr)
}

func (t *Transpiler) transpileStructLiteral(expr *ast.StructLit) string {
	var fields []string
	for key, value := range expr.Fields {
		fieldName := utils.ToPascalCase(key)
		fieldValue := t.transpileExpr(value)
		fields = append(fields, fmt.Sprintf("%s: %s", fieldName, fieldValue))
	}
	// If it's a model type, create as pointer for consistency with ORM helpers
	if t.isModelType(expr.TypeName) {
		return fmt.Sprintf("&%s{%s}", expr.TypeName, strings.Join(fields, ", "))
	}
	return fmt.Sprintf("%s{%s}", expr.TypeName, strings.Join(fields, ", "))
}

func (t *Transpiler) transpileStringInterpolationParts(parts []ast.StringPart) string {
	// Convert StringParts to fmt.Sprintf
	var formatParts []string
	var args []string

	for _, part := range parts {
		if part.IsExpr {
			formatParts = append(formatParts, "%v")
			args = append(args, t.transpileExpr(part.Expr))
		} else {
			formatParts = append(formatParts, part.Text)
		}
	}

	formatStr := strings.Join(formatParts, "")
	if len(args) == 0 {
		return fmt.Sprintf("%q", formatStr)
	}

	return fmt.Sprintf("fmt.Sprintf(%q, %s)", formatStr, strings.Join(args, ", "))
}

func (t *Transpiler) transpileRenderCall(call *ast.CallExpr) {
	// render(task) or render(task, sidebar)
	t.emitIndent()

	if len(call.Args) == 1 {
		// Single render
		arg := t.transpileExpr(call.Args[0])
		// Determine type name for template lookup
		typeName := t.inferTypeName(call.Args[0])
		t.emit("if err := renderFragment(ctx.Writer, %q, %s); err != nil {\n", typeName, arg)
		t.indent++
		t.emitIndent()
		t.emit("return err\n")
		t.indent--
		t.emitIndent()
		t.emit("}\n")
	} else {
		// Multiple renders (OOB)
		for _, arg := range call.Args {
			argStr := t.transpileExpr(arg)
			typeName := t.inferTypeName(arg)
			t.emit("if err := renderFragment(ctx.Writer, %q, %s); err != nil {\n", typeName, argStr)
			t.indent++
			t.emitIndent()
			t.emit("return err\n")
			t.indent--
			t.emitIndent()
			t.emit("}\n")
			t.emitIndent()
		}
	}
}

func (t *Transpiler) transpileType(typ string) string {
	switch typ {
	case "uuid":
		return "string"
	case "int":
		return "int"
	case "bool":
		return "bool"
	case "string":
		return "string"
	default:
		// Might be a model type
		if t.isModelType(typ) {
			return "*" + typ
		}
		return typ
	}
}

func (t *Transpiler) isModelType(name string) bool {
	for _, model := range t.models {
		if model == name {
			return true
		}
	}
	return false
}

func (t *Transpiler) trackVarType(varName string, expr ast.Expression) {
	// Track variable type for instance method detection
	switch e := expr.(type) {
	case *ast.CallExpr:
		// Check if it's Model.find() or Model.all()
		if member, ok := e.Function.(*ast.MemberExpr); ok {
			if ident, ok := member.Object.(*ast.Ident); ok {
				if t.isModelType(ident.Name) {
					t.varTypes[varName] = ident.Name
				}
			}
		}
	case *ast.StructLit:
		if t.isModelType(e.TypeName) {
			t.varTypes[varName] = e.TypeName
		}
	}
}

func (t *Transpiler) inferTypeName(expr ast.Expression) string {
	// Infer type name for render template lookup
	switch e := expr.(type) {
	case *ast.Ident:
		// Check if we know the type
		if typ, ok := t.varTypes[e.Name]; ok {
			return typ
		}
		return e.Name
	case *ast.StructLit:
		return e.TypeName
	default:
		return "Unknown"
	}
}

func (t *Transpiler) endsWithReturn(body []ast.Statement) bool {
	if len(body) == 0 {
		return false
	}
	_, ok := body[len(body)-1].(*ast.ReturnStmt)
	return ok
}

func (t *Transpiler) emit(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	t.buf.WriteString(str)
	t.goLine += strings.Count(str, "\n")
}

func (t *Transpiler) emitIndent() {
	t.buf.WriteString(strings.Repeat("\t", t.indent))
}

func (t *Transpiler) emitLineComment(gmxLine int) {
	t.sourceMap.Entries = append(t.sourceMap.Entries, SourceMapEntry{
		GoLine:  t.goLine + 1,
		GmxLine: gmxLine,
	})
	t.emit(" // gmx:%d\n", gmxLine)
}

func (t *Transpiler) genORMHelpers() {
	t.emit("// ORM helper functions\n\n")

	for _, model := range t.models {
		// Find helper
		t.emit("func %sFind(db *gorm.DB, id string) (*%s, error) {\n", model, model)
		t.emit("\tvar obj %s\n", model)
		t.emit("\tif err := db.First(&obj, \"id = ?\", id).Error; err != nil {\n")
		t.emit("\t\treturn nil, err\n")
		t.emit("\t}\n")
		t.emit("\treturn &obj, nil\n")
		t.emit("}\n\n")

		// All helper
		t.emit("func %sAll(db *gorm.DB) ([]%s, error) {\n", model, model)
		t.emit("\tvar objs []%s\n", model)
		t.emit("\tif err := db.Find(&objs).Error; err != nil {\n")
		t.emit("\t\treturn nil, err\n")
		t.emit("\t}\n")
		t.emit("\treturn objs, nil\n")
		t.emit("}\n\n")

		// Save helper
		t.emit("func %sSave(db *gorm.DB, obj *%s) error {\n", model, model)
		t.emit("\treturn db.Save(obj).Error\n")
		t.emit("}\n\n")

		// Delete helper
		t.emit("func %sDelete(db *gorm.DB, obj *%s) error {\n", model, model)
		t.emit("\treturn db.Delete(obj).Error\n")
		t.emit("}\n\n")
	}
}

func (t *Transpiler) genGMXContext() {
	t.emit("// GMXContext holds request context and dependencies\n")
	t.emit("type GMXContext struct {\n")
	t.emit("\tDB      *gorm.DB\n")
	t.emit("\tTenant  string\n")
	t.emit("\tUser    string\n")
	t.emit("\tWriter  http.ResponseWriter\n")
	t.emit("\tRequest *http.Request\n")
	t.emit("}\n\n")
}

func (t *Transpiler) genRenderFragment() {
	t.emit("// renderFragment executes a template fragment\n")
	t.emit("func renderFragment(w http.ResponseWriter, name string, data interface{}) error {\n")
	t.emit("\tw.Header().Set(\"Content-Type\", \"text/html; charset=utf-8\")\n")
	t.emit("\treturn tmpl.ExecuteTemplate(w, name, data)\n")
	t.emit("}\n\n")
}

func (t *Transpiler) transpileErrorExpr(expr *ast.ErrorExpr) string {
	// error("message") -> fmt.Errorf("message")
	msgStr := t.transpileExpr(expr.Message)
	return fmt.Sprintf("fmt.Errorf(%s)", msgStr)
}

func (t *Transpiler) transpileRenderExpr(expr *ast.RenderExpr) {
	// render(task) or render(task, sidebar)
	t.emitIndent()

	if len(expr.Args) == 1 {
		// Single render
		arg := t.transpileExpr(expr.Args[0])
		// Determine type name for template lookup
		typeName := t.inferTypeName(expr.Args[0])
		t.emit("if err := renderFragment(ctx.Writer, %q, %s); err != nil {\n", typeName, arg)
		t.indent++
		t.emitIndent()
		t.emit("return err\n")
		t.indent--
		t.emitIndent()
		t.emit("}\n")
	} else {
		// Multiple renders (OOB)
		for _, arg := range expr.Args {
			argStr := t.transpileExpr(arg)
			typeName := t.inferTypeName(arg)
			t.emit("if err := renderFragment(ctx.Writer, %q, %s); err != nil {\n", typeName, argStr)
			t.indent++
			t.emitIndent()
			t.emit("return err\n")
			t.indent--
			t.emitIndent()
			t.emit("}\n")
			t.emitIndent()
		}
	}
}
