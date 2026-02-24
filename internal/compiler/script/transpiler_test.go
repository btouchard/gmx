package script

import (
	"strings"
	"testing"

	"gmx/internal/compiler/ast"
)

func TestTranspileLetStatement(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "x",
						Value: &ast.IntLit{Value: "42"},
						Const: false,
						Line:  1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, "x := 42") {
		t.Errorf("Expected 'x := 42', got: %s", result.GoCode)
	}
}

func TestTranspileTryLet(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "test",
				Params: []*ast.Param{
					{Name: "id", Type: "uuid"},
				},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.TryExpr{
							Expr: &ast.CallExpr{
								Function: &ast.MemberExpr{
									Object:   &ast.Ident{Name: "Task"},
									Property: "find",
								},
								Args: []ast.Expression{
									&ast.Ident{Name: "id"},
								},
							},
						},
						Const: false,
						Line:  2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	code := result.GoCode

	// Should have error handling
	if !strings.Contains(code, "task, err := TaskFind(ctx.DB, id)") {
		t.Errorf("Expected 'task, err := TaskFind(ctx.DB, id)', got: %s", code)
	}
	if !strings.Contains(code, "if err != nil {") {
		t.Errorf("Expected error checking, got: %s", code)
	}
	if !strings.Contains(code, "return err") {
		t.Errorf("Expected 'return err', got: %s", code)
	}
}

func TestTranspileTryStatement(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields: map[string]ast.Expression{
								"title": &ast.StringLit{Value: "Test"},
							},
						},
						Const: true,
						Line:  1,
					},
					&ast.ExprStmt{
						Expr: &ast.TryExpr{
							Expr: &ast.CallExpr{
								Function: &ast.MemberExpr{
									Object:   &ast.Ident{Name: "task"},
									Property: "save",
								},
								Args: []ast.Expression{},
							},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	code := result.GoCode

	if !strings.Contains(code, "if err := TaskSave(ctx.DB, task); err != nil {") {
		t.Errorf("Expected 'if err := TaskSave(ctx.DB, task); err != nil {', got: %s", code)
	}
}

func TestTranspileModelFind(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "test",
				Params: []*ast.Param{
					{Name: "id", Type: "uuid"},
				},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.CallExpr{
							Function: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "Task"},
								Property: "find",
							},
							Args: []ast.Expression{
								&ast.Ident{Name: "id"},
							},
						},
						Const: false,
						Line:  1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	if !strings.Contains(result.GoCode, "TaskFind(ctx.DB, id)") {
		t.Errorf("Expected 'TaskFind(ctx.DB, id)', got: %s", result.GoCode)
	}
}

func TestTranspileModelAll(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "tasks",
						Value: &ast.CallExpr{
							Function: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "Task"},
								Property: "all",
							},
							Args: []ast.Expression{},
						},
						Const: false,
						Line:  1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	if !strings.Contains(result.GoCode, "TaskAll(ctx.DB)") {
		t.Errorf("Expected 'TaskAll(ctx.DB)', got: %s", result.GoCode)
	}
}

func TestTranspileModelSave(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields: map[string]ast.Expression{
								"title": &ast.StringLit{Value: "Test"},
							},
						},
						Const: true,
						Line:  1,
					},
					&ast.ExprStmt{
						Expr: &ast.CallExpr{
							Function: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "task"},
								Property: "save",
							},
							Args: []ast.Expression{},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	if !strings.Contains(result.GoCode, "TaskSave(ctx.DB, task)") {
		t.Errorf("Expected 'TaskSave(ctx.DB, task)', got: %s", result.GoCode)
	}
}

func TestTranspileModelDelete(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields: map[string]ast.Expression{
								"title": &ast.StringLit{Value: "Test"},
							},
						},
						Const: true,
						Line:  1,
					},
					&ast.ExprStmt{
						Expr: &ast.CallExpr{
							Function: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "task"},
								Property: "delete",
							},
							Args: []ast.Expression{},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	if !strings.Contains(result.GoCode, "TaskDelete(ctx.DB, task)") {
		t.Errorf("Expected 'TaskDelete(ctx.DB, task)', got: %s", result.GoCode)
	}
}

func TestTranspileRender(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields: map[string]ast.Expression{
								"title": &ast.StringLit{Value: "Test"},
							},
						},
						Const: true,
						Line:  1,
					},
					&ast.ReturnStmt{
						Value: &ast.CallExpr{
							Function: &ast.Ident{Name: "render"},
							Args: []ast.Expression{
								&ast.Ident{Name: "task"},
							},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	code := result.GoCode

	if !strings.Contains(code, `renderFragment(ctx.Writer, "Task", task)`) {
		t.Errorf("Expected renderFragment call, got: %s", code)
	}
}

func TestTranspileError(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.ReturnStmt{
						Value: &ast.CallExpr{
							Function: &ast.Ident{Name: "error"},
							Args: []ast.Expression{
								&ast.StringLit{Value: "not found"},
							},
						},
						Line: 1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, `fmt.Errorf("not found")`) {
		t.Errorf("Expected 'fmt.Errorf(\"not found\")', got: %s", result.GoCode)
	}
}

func TestTranspileCtx(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "tenant",
						Value: &ast.MemberExpr{
							Object:   &ast.Ident{Name: "ctx"},
							Property: "tenant",
						},
						Const: false,
						Line:  1,
					},
					&ast.LetStmt{
						Name: "user",
						Value: &ast.MemberExpr{
							Object:   &ast.Ident{Name: "ctx"},
							Property: "user",
						},
						Const: false,
						Line:  2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, "ctx.Tenant") {
		t.Errorf("Expected 'ctx.Tenant', got: %s", result.GoCode)
	}
	if !strings.Contains(result.GoCode, "ctx.User") {
		t.Errorf("Expected 'ctx.User', got: %s", result.GoCode)
	}
}

func TestTranspileStringInterpolation(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "name",
						Value: &ast.StringLit{Value: "World"},
						Const: true,
						Line:  1,
					},
					&ast.LetStmt{
						Name: "msg",
						Value: &ast.StringLit{
							Value: "Hello {name}",
							Parts: []ast.StringPart{
								{IsExpr: false, Text: "Hello "},
								{IsExpr: true, Expr: &ast.Ident{Name: "name"}},
							},
						},
						Const: false,
						Line:  2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, `fmt.Sprintf("Hello %v", name)`) {
		t.Errorf("Expected string interpolation, got: %s", result.GoCode)
	}
}

func TestTranspileIfElse(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "x",
						Value: &ast.IntLit{Value: "5"},
						Const: true,
						Line:  1,
					},
					&ast.IfStmt{
						Condition: &ast.BinaryExpr{
							Left:  &ast.Ident{Name: "x"},
							Op:    ">",
							Right: &ast.IntLit{Value: "3"},
						},
						Consequence: []ast.Statement{
							&ast.LetStmt{
								Name:  "a",
								Value: &ast.IntLit{Value: "1"},
								Const: false,
								Line:  3,
							},
						},
						Alternative: []ast.Statement{
							&ast.LetStmt{
								Name:  "b",
								Value: &ast.IntLit{Value: "2"},
								Const: false,
								Line:  5,
							},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	code := result.GoCode

	if !strings.Contains(code, "if x > 3 {") {
		t.Errorf("Expected 'if x > 3 {', got: %s", code)
	}
	if !strings.Contains(code, "} else {") {
		t.Errorf("Expected '} else {', got: %s", code)
	}
}

func TestTranspileAssignment(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields: map[string]ast.Expression{
								"done": &ast.BoolLit{Value: false},
							},
						},
						Const: true,
						Line:  1,
					},
					&ast.AssignStmt{
						Target: &ast.MemberExpr{
							Object:   &ast.Ident{Name: "task"},
							Property: "done",
						},
						Value: &ast.UnaryExpr{
							Op: "!",
							Operand: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "task"},
								Property: "done",
							},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	if !strings.Contains(result.GoCode, "task.Done = !task.Done") {
		t.Errorf("Expected 'task.Done = !task.Done', got: %s", result.GoCode)
	}
}

func TestTranspileStructLiteral(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "test",
				Params: []*ast.Param{
					{Name: "title", Type: "string"},
				},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields: map[string]ast.Expression{
								"title": &ast.Ident{Name: "title"},
								"done":  &ast.BoolLit{Value: false},
							},
						},
						Const: true,
						Line:  1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	// Note: map iteration order is random, so we check for both fields separately
	if !strings.Contains(result.GoCode, "Title: title") {
		t.Errorf("Expected 'Title: title', got: %s", result.GoCode)
	}
	if !strings.Contains(result.GoCode, "Done: false") {
		t.Errorf("Expected 'Done: false', got: %s", result.GoCode)
	}
}

func TestTranspileCompleteFunction(t *testing.T) {
	// Full toggleTask function
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "toggleTask",
				Params: []*ast.Param{
					{Name: "id", Type: "uuid"},
				},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.TryExpr{
							Expr: &ast.CallExpr{
								Function: &ast.MemberExpr{
									Object:   &ast.Ident{Name: "Task"},
									Property: "find",
								},
								Args: []ast.Expression{
									&ast.Ident{Name: "id"},
								},
							},
						},
						Const: false,
						Line:  2,
					},
					&ast.AssignStmt{
						Target: &ast.MemberExpr{
							Object:   &ast.Ident{Name: "task"},
							Property: "done",
						},
						Value: &ast.UnaryExpr{
							Op: "!",
							Operand: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "task"},
								Property: "done",
							},
						},
						Line: 3,
					},
					&ast.ExprStmt{
						Expr: &ast.TryExpr{
							Expr: &ast.CallExpr{
								Function: &ast.MemberExpr{
									Object:   &ast.Ident{Name: "task"},
									Property: "save",
								},
								Args: []ast.Expression{},
							},
						},
						Line: 4,
					},
					&ast.ReturnStmt{
						Value: &ast.CallExpr{
							Function: &ast.Ident{Name: "render"},
							Args: []ast.Expression{
								&ast.Ident{Name: "task"},
							},
						},
						Line: 5,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	code := result.GoCode

	// Check function signature
	if !strings.Contains(code, "func toggleTask(ctx *GMXContext, id string) error {") {
		t.Errorf("Expected proper function signature, got: %s", code)
	}

	// Check Task.find with error handling
	if !strings.Contains(code, "task, err := TaskFind(ctx.DB, id)") {
		t.Errorf("Expected TaskFind call, got: %s", code)
	}

	// Check assignment
	if !strings.Contains(code, "task.Done = !task.Done") {
		t.Errorf("Expected assignment, got: %s", code)
	}

	// Check save with error handling
	if !strings.Contains(code, "if err := TaskSave(ctx.DB, task); err != nil {") {
		t.Errorf("Expected TaskSave call, got: %s", code)
	}

	// Check render
	if !strings.Contains(code, `renderFragment(ctx.Writer, "Task", task)`) {
		t.Errorf("Expected renderFragment call, got: %s", code)
	}
}

func TestTranspileGeneratesOrmHelpers(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body:   []ast.Statement{},
				Line:   1,
			},
		},
	}

	result := Transpile(script, []string{"Task", "Post"})
	code := result.GoCode

	// Check ORM helpers are generated
	if !strings.Contains(code, "func TaskFind(db *gorm.DB, id string) (*Task, error)") {
		t.Errorf("Expected TaskFind helper, got: %s", code)
	}
	if !strings.Contains(code, "func TaskAll(db *gorm.DB) ([]Task, error)") {
		t.Errorf("Expected TaskAll helper, got: %s", code)
	}
	if !strings.Contains(code, "func TaskSave(db *gorm.DB, obj *Task) error") {
		t.Errorf("Expected TaskSave helper, got: %s", code)
	}
	if !strings.Contains(code, "func TaskDelete(db *gorm.DB, obj *Task) error") {
		t.Errorf("Expected TaskDelete helper, got: %s", code)
	}

	if !strings.Contains(code, "func PostFind(db *gorm.DB, id string) (*Post, error)") {
		t.Errorf("Expected PostFind helper, got: %s", code)
	}
}

func TestSourceMap(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "x",
						Value: &ast.IntLit{Value: "42"},
						Const: false,
						Line:  5,
					},
					&ast.LetStmt{
						Name:  "y",
						Value: &ast.IntLit{Value: "10"},
						Const: false,
						Line:  6,
					},
				},
				Line: 3,
			},
		},
	}

	result := Transpile(script, []string{})

	// Check that source map has entries
	if len(result.SourceMap.Entries) == 0 {
		t.Errorf("Expected source map entries, got none")
	}

	// Check that source map has correct GMX line numbers
	foundLine5 := false
	foundLine6 := false
	for _, entry := range result.SourceMap.Entries {
		if entry.GmxLine == 5 {
			foundLine5 = true
		}
		if entry.GmxLine == 6 {
			foundLine6 = true
		}
	}

	if !foundLine5 || !foundLine6 {
		t.Errorf("Expected source map entries for lines 5 and 6, got: %+v", result.SourceMap.Entries)
	}
}

func TestTranspileGMXContextGeneration(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body:   []ast.Statement{},
				Line:   1,
			},
		},
	}

	result := Transpile(script, []string{})
	code := result.GoCode

	if !strings.Contains(code, "type GMXContext struct {") {
		t.Errorf("Expected GMXContext struct, got: %s", code)
	}
	if !strings.Contains(code, "DB      *gorm.DB") {
		t.Errorf("Expected DB field, got: %s", code)
	}
	if !strings.Contains(code, "Tenant  string") {
		t.Errorf("Expected Tenant field, got: %s", code)
	}
}

func TestTranspileRenderFragmentHelper(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body:   []ast.Statement{},
				Line:   1,
			},
		},
	}

	result := Transpile(script, []string{})
	code := result.GoCode

	if !strings.Contains(code, "func renderFragment(w http.ResponseWriter, name string, data interface{}) error {") {
		t.Errorf("Expected renderFragment helper, got: %s", code)
	}
}

// Test transpileRenderExpr (0% coverage)
func TestTranspileRenderExpr(t *testing.T) {
	// Test single render
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields: map[string]ast.Expression{
								"title": &ast.StringLit{Value: "Test"},
							},
						},
						Const: true,
						Line:  1,
					},
					&ast.ReturnStmt{
						Value: &ast.RenderExpr{
							Args: []ast.Expression{
								&ast.Ident{Name: "task"},
							},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	code := result.GoCode

	if !strings.Contains(code, `renderFragment(ctx.Writer, "Task", task)`) {
		t.Errorf("Expected renderFragment call, got: %s", code)
	}
}

// Test transpileRenderExpr with multiple args (OOB)
func TestTranspileRenderExprMultiple(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields:   map[string]ast.Expression{},
						},
						Const: true,
						Line:  1,
					},
					&ast.LetStmt{
						Name: "tasks",
						Value: &ast.StructLit{
							TypeName: "TaskList",
							Fields:   map[string]ast.Expression{},
						},
						Const: true,
						Line:  2,
					},
					&ast.ReturnStmt{
						Value: &ast.RenderExpr{
							Args: []ast.Expression{
								&ast.Ident{Name: "task"},
								&ast.Ident{Name: "tasks"},
							},
						},
						Line: 3,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task", "TaskList"})
	code := result.GoCode

	if !strings.Contains(code, `renderFragment(ctx.Writer, "Task", task)`) {
		t.Errorf("Expected first renderFragment call, got: %s", code)
	}
	if !strings.Contains(code, `renderFragment(ctx.Writer, "TaskList", tasks)`) {
		t.Errorf("Expected second renderFragment call, got: %s", code)
	}
}

// Test transpileErrorExpr (0% coverage)
func TestTranspileErrorExpr(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.ReturnStmt{
						Value: &ast.ErrorExpr{
							Message: &ast.StringLit{Value: "not found"},
						},
						Line: 1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, `fmt.Errorf("not found")`) {
		t.Errorf("Expected 'fmt.Errorf(\"not found\")', got: %s", result.GoCode)
	}
}

// Test transpileErrorExpr as expression in let statement
func TestTranspileErrorExprInLet(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "err",
						Value: &ast.ErrorExpr{
							Message: &ast.StringLit{Value: "validation failed"},
						},
						Const: false,
						Line:  1,
					},
					&ast.ReturnStmt{
						Value: &ast.Ident{Name: "err"},
						Line:  2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, `fmt.Errorf("validation failed")`) {
		t.Errorf("Expected 'fmt.Errorf(\"validation failed\")', got: %s", result.GoCode)
	}
}

// Test FloatLit transpilation (bugfix)
func TestTranspileFloatLiteral(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "pi",
						Value: &ast.FloatLit{Value: "3.14"},
						Const: true,
						Line:  1,
					},
					&ast.LetStmt{
						Name:  "half",
						Value: &ast.FloatLit{Value: "0.5"},
						Const: false,
						Line:  2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	code := result.GoCode

	if !strings.Contains(code, "pi := 3.14") {
		t.Errorf("Expected 'pi := 3.14', got: %s", code)
	}
	if !strings.Contains(code, "half := 0.5") {
		t.Errorf("Expected 'half := 0.5', got: %s", code)
	}
}

// Test transpilation with const keyword
func TestTranspileConstDeclaration(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "max",
						Value: &ast.IntLit{Value: "100"},
						Const: true,
						Line:  1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, "max := 100") {
		t.Errorf("Expected 'max := 100', got: %s", result.GoCode)
	}
}

// Test transpilation of complex expressions
func TestTranspileComplexBinaryExpression(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "result",
						Value: &ast.BinaryExpr{
							Left: &ast.BinaryExpr{
								Left:  &ast.Ident{Name: "a"},
								Op:    "==",
								Right: &ast.Ident{Name: "b"},
							},
							Op: "&&",
							Right: &ast.BinaryExpr{
								Left:  &ast.Ident{Name: "c"},
								Op:    "!=",
								Right: &ast.Ident{Name: "d"},
							},
						},
						Const: false,
						Line:  1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, "a == b && c != d") {
		t.Errorf("Expected 'a == b && c != d', got: %s", result.GoCode)
	}
}

// Test transpilation of unary not expression
func TestTranspileUnaryNotExpression(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields: map[string]ast.Expression{
								"done": &ast.BoolLit{Value: false},
							},
						},
						Const: true,
						Line:  1,
					},
					&ast.LetStmt{
						Name: "notDone",
						Value: &ast.UnaryExpr{
							Op: "!",
							Operand: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "task"},
								Property: "done",
							},
						},
						Const: false,
						Line:  2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	if !strings.Contains(result.GoCode, "!task.Done") {
		t.Errorf("Expected '!task.Done', got: %s", result.GoCode)
	}
}

// Test transpilation of return statement without value
func TestTranspileReturnNil(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "x",
						Value: &ast.IntLit{Value: "1"},
						Const: false,
						Line:  1,
					},
					&ast.ReturnStmt{
						Value: nil,
						Line:  2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	// Should have explicit "return nil"
	if !strings.Contains(result.GoCode, "return nil") {
		t.Errorf("Expected 'return nil', got: %s", result.GoCode)
	}
}

// Test function without explicit return statement
func TestTranspileFunctionWithoutReturn(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "x",
						Value: &ast.IntLit{Value: "42"},
						Const: false,
						Line:  1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	code := result.GoCode

	// Should add implicit "return nil" at the end
	if !strings.Contains(code, "return nil") {
		t.Errorf("Expected implicit 'return nil', got: %s", code)
	}
}

// Test transpilation with multiple float values
func TestTranspileMultipleFloats(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "a",
						Value: &ast.FloatLit{Value: "3.14"},
						Const: false,
						Line:  1,
					},
					&ast.LetStmt{
						Name:  "b",
						Value: &ast.FloatLit{Value: "0.5"},
						Const: false,
						Line:  2,
					},
					&ast.LetStmt{
						Name:  "c",
						Value: &ast.FloatLit{Value: ".25"},
						Const: false,
						Line:  3,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	code := result.GoCode

	if !strings.Contains(code, "a := 3.14") {
		t.Errorf("Expected 'a := 3.14', got: %s", code)
	}
	if !strings.Contains(code, "b := 0.5") {
		t.Errorf("Expected 'b := 0.5', got: %s", code)
	}
	if !strings.Contains(code, "c := .25") {
		t.Errorf("Expected 'c := .25', got: %s", code)
	}
}

// Test transpileType with all GMX types (37.5% coverage)
func TestTranspileTypeUUID(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "test",
				Params: []*ast.Param{
					{Name: "id", Type: "uuid"},
				},
				Body:   []ast.Statement{},
				Line:   1,
			},
		},
	}

	result := Transpile(script, []string{})
	// uuid should be transpiled to string
	if !strings.Contains(result.GoCode, "id string") {
		t.Errorf("Expected 'id string', got: %s", result.GoCode)
	}
}

func TestTranspileTypeInt(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "test",
				Params: []*ast.Param{
					{Name: "count", Type: "int"},
				},
				Body:   []ast.Statement{},
				Line:   1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, "count int") {
		t.Errorf("Expected 'count int', got: %s", result.GoCode)
	}
}

func TestTranspileTypeBool(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "test",
				Params: []*ast.Param{
					{Name: "active", Type: "bool"},
				},
				Body:   []ast.Statement{},
				Line:   1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, "active bool") {
		t.Errorf("Expected 'active bool', got: %s", result.GoCode)
	}
}

func TestTranspileTypeString(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "test",
				Params: []*ast.Param{
					{Name: "name", Type: "string"},
				},
				Body:   []ast.Statement{},
				Line:   1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, "name string") {
		t.Errorf("Expected 'name string', got: %s", result.GoCode)
	}
}

func TestTranspileTypeModelPointer(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "test",
				Params: []*ast.Param{
					{Name: "task", Type: "Task"},
				},
				Body:   []ast.Statement{},
				Line:   1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	// Model types should be transpiled to pointers
	if !strings.Contains(result.GoCode, "task *Task") {
		t.Errorf("Expected 'task *Task', got: %s", result.GoCode)
	}
}

func TestTranspileTypeUnknown(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name: "test",
				Params: []*ast.Param{
					{Name: "data", Type: "CustomType"},
				},
				Body:   []ast.Statement{},
				Line:   1,
			},
		},
	}

	result := Transpile(script, []string{})
	// Unknown types should be passed through as-is
	if !strings.Contains(result.GoCode, "data CustomType") {
		t.Errorf("Expected 'data CustomType', got: %s", result.GoCode)
	}
}

// Test inferTypeName (50% coverage)
func TestInferTypeNameFromIdent(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.CallExpr{
							Function: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "Task"},
								Property: "find",
							},
							Args: []ast.Expression{
								&ast.StringLit{Value: "123"},
							},
						},
						Const: false,
						Line:  1,
					},
					&ast.ReturnStmt{
						Value: &ast.CallExpr{
							Function: &ast.Ident{Name: "render"},
							Args: []ast.Expression{
								&ast.Ident{Name: "task"},
							},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	// Should infer "Task" from the variable type
	if !strings.Contains(result.GoCode, `renderFragment(ctx.Writer, "Task", task)`) {
		t.Errorf("Expected type inference to 'Task', got: %s", result.GoCode)
	}
}

func TestInferTypeNameFromStructLit(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.ReturnStmt{
						Value: &ast.CallExpr{
							Function: &ast.Ident{Name: "render"},
							Args: []ast.Expression{
								&ast.StructLit{
									TypeName: "Post",
									Fields:   map[string]ast.Expression{},
								},
							},
						},
						Line: 1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Post"})
	// Should infer "Post" from struct literal
	if !strings.Contains(result.GoCode, `renderFragment(ctx.Writer, "Post",`) {
		t.Errorf("Expected type inference to 'Post', got: %s", result.GoCode)
	}
}

func TestInferTypeNameUnknown(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.ReturnStmt{
						Value: &ast.CallExpr{
							Function: &ast.Ident{Name: "render"},
							Args: []ast.Expression{
								&ast.IntLit{Value: "42"},
							},
						},
						Line: 1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	// Should fall back to "Unknown" for non-inferrable types
	if !strings.Contains(result.GoCode, `renderFragment(ctx.Writer, "Unknown",`) {
		t.Errorf("Expected type inference to 'Unknown', got: %s", result.GoCode)
	}
}

// Test transpileRenderCall with multiple args (50% coverage)
func TestTranspileRenderCallMultiple(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields:   map[string]ast.Expression{},
						},
						Const: true,
						Line:  1,
					},
					&ast.LetStmt{
						Name: "sidebar",
						Value: &ast.StructLit{
							TypeName: "Sidebar",
							Fields:   map[string]ast.Expression{},
						},
						Const: true,
						Line:  2,
					},
					&ast.ReturnStmt{
						Value: &ast.CallExpr{
							Function: &ast.Ident{Name: "render"},
							Args: []ast.Expression{
								&ast.Ident{Name: "task"},
								&ast.Ident{Name: "sidebar"},
							},
						},
						Line: 3,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task", "Sidebar"})
	code := result.GoCode

	// Should render both fragments
	if !strings.Contains(code, `renderFragment(ctx.Writer, "Task", task)`) {
		t.Errorf("Expected first renderFragment call, got: %s", code)
	}
	if !strings.Contains(code, `renderFragment(ctx.Writer, "Sidebar", sidebar)`) {
		t.Errorf("Expected second renderFragment call, got: %s", code)
	}
}

// Test transpileCallExpr with error builtin non-string arg (77.3% coverage)
func TestTranspileCallExprErrorWithExpr(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name:  "count",
						Value: &ast.IntLit{Value: "42"},
						Const: true,
						Line:  1,
					},
					&ast.ReturnStmt{
						Value: &ast.CallExpr{
							Function: &ast.Ident{Name: "error"},
							Args: []ast.Expression{
								&ast.Ident{Name: "count"},
							},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	// Should use %v format for non-string expressions
	if !strings.Contains(result.GoCode, `fmt.Errorf("%v", count)`) {
		t.Errorf("Expected 'fmt.Errorf(\"%%v\", count)', got: %s", result.GoCode)
	}
}

func TestTranspileCallExprInstanceMethodSave(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields:   map[string]ast.Expression{},
						},
						Const: true,
						Line:  1,
					},
					&ast.ExprStmt{
						Expr: &ast.CallExpr{
							Function: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "task"},
								Property: "save",
							},
							Args: []ast.Expression{},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	if !strings.Contains(result.GoCode, "TaskSave(ctx.DB, task)") {
		t.Errorf("Expected 'TaskSave(ctx.DB, task)', got: %s", result.GoCode)
	}
}

func TestTranspileCallExprInstanceMethodDelete(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "task",
						Value: &ast.StructLit{
							TypeName: "Task",
							Fields:   map[string]ast.Expression{},
						},
						Const: true,
						Line:  1,
					},
					&ast.ExprStmt{
						Expr: &ast.CallExpr{
							Function: &ast.MemberExpr{
								Object:   &ast.Ident{Name: "task"},
								Property: "delete",
							},
							Args: []ast.Expression{},
						},
						Line: 2,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{"Task"})
	if !strings.Contains(result.GoCode, "TaskDelete(ctx.DB, task)") {
		t.Errorf("Expected 'TaskDelete(ctx.DB, task)', got: %s", result.GoCode)
	}
}

func TestTranspileCallExprRegularFunction(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "result",
						Value: &ast.CallExpr{
							Function: &ast.Ident{Name: "doSomething"},
							Args: []ast.Expression{
								&ast.StringLit{Value: "arg1"},
								&ast.IntLit{Value: "42"},
							},
						},
						Const: false,
						Line:  1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	if !strings.Contains(result.GoCode, `doSomething("arg1", 42)`) {
		t.Errorf("Expected regular function call, got: %s", result.GoCode)
	}
}

// Test transpile with string interpolation but no parts (edge case)
func TestTranspileStringNoInterpolation(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.LetStmt{
						Name: "msg",
						Value: &ast.StringLit{
							Value: "plain string",
							Parts: nil, // No interpolation parts
						},
						Const: false,
						Line:  1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	// Should use normal string literal
	if !strings.Contains(result.GoCode, `msg := "plain string"`) {
		t.Errorf("Expected 'msg := \"plain string\"', got: %s", result.GoCode)
	}
}

// Test if statement without else branch
func TestTranspileIfWithoutElse(t *testing.T) {
	script := &ast.ScriptBlock{
		Funcs: []*ast.FuncDecl{
			{
				Name:   "test",
				Params: []*ast.Param{},
				Body: []ast.Statement{
					&ast.IfStmt{
						Condition: &ast.BoolLit{Value: true},
						Consequence: []ast.Statement{
							&ast.LetStmt{
								Name:  "x",
								Value: &ast.IntLit{Value: "1"},
								Const: false,
								Line:  2,
							},
						},
						Alternative: nil,
						Line:        1,
					},
				},
				Line: 1,
			},
		},
	}

	result := Transpile(script, []string{})
	code := result.GoCode

	if !strings.Contains(code, "if true {") {
		t.Errorf("Expected 'if true {', got: %s", code)
	}
	// Should NOT contain else branch
	if strings.Contains(code, "} else {") {
		t.Errorf("Expected no else branch, got: %s", code)
	}
}
