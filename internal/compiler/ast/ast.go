package ast

// Node is the base interface for all AST nodes
type Node interface {
	TokenLiteral() string
}

// GMXFile is the root AST node representing a complete .gmx file
type GMXFile struct {
	Imports  []*ImportDecl
	Models   []*ModelDecl
	Services []*ServiceDecl
	Vars     []*VarDecl
	Script   *ScriptBlock
	Template *TemplateBlock
	Style    *StyleBlock
}

func (f *GMXFile) TokenLiteral() string { return "gmx" }

// ============ MODEL SECTION ============

// ModelDecl represents a model definition: model Task { ... }
type ModelDecl struct {
	Name   string
	Fields []*FieldDecl
}

func (m *ModelDecl) TokenLiteral() string { return "model" }

// FieldDecl represents a field: title: string @min(3) @max(255)
type FieldDecl struct {
	Name        string
	Type        string // "uuid", "string", "bool", "int", "float", "datetime", "User", "Post[]"
	Annotations []*Annotation
}

func (f *FieldDecl) TokenLiteral() string { return f.Name }

// ============ SERVICE SECTION ============

// ServiceDecl represents a service declaration
type ServiceDecl struct {
	Name     string
	Provider string
	Fields   []*ServiceField
	Methods  []*ServiceMethod
}

func (s *ServiceDecl) TokenLiteral() string { return "service" }

// ServiceField is a config field with env binding
type ServiceField struct {
	Name        string
	Type        string
	EnvVar      string
	Annotations []*Annotation
}

func (s *ServiceField) TokenLiteral() string { return s.Name }

// ServiceMethod is a method signature declared on a service
type ServiceMethod struct {
	Name       string
	Params     []*Param
	ReturnType string
}

func (s *ServiceMethod) TokenLiteral() string { return s.Name }

// ============ ANNOTATIONS ============

// Annotation represents a tag like @pk, @default(uuid_v4), @relation(references: [id])
type Annotation struct {
	Name string            // "pk", "default", "min", "max", "unique", "email", "scoped", "relation"
	Args map[string]string // For simple: @default(false) -> {"_": "false"}, For named: @relation(references: [id]) -> {"references": "id"}
}

func (a *Annotation) TokenLiteral() string { return "@" + a.Name }

// SimpleArg returns the unnamed argument value (for @default(x), @min(3), etc.)
func (a *Annotation) SimpleArg() string {
	if v, ok := a.Args["_"]; ok {
		return v
	}
	return ""
}

// ============ SCRIPT SECTION ============

// ImportDecl represents an import declaration with three syntaxes:
// 1. Default import: import TaskItem from './components/TaskItem.gmx'
// 2. Destructured import: import { sendEmail, MailerConfig } from './services/mailer.gmx'
// 3. Native Go import: import "github.com/stripe/stripe-go" as Stripe
type ImportDecl struct {
	Default  string   // "TaskItem" (import X from '...')
	Members  []string // ["sendEmail", "MailerConfig"] (import { x, y } from '...')
	Path     string   // "./components/TaskItem.gmx" or "github.com/stripe/stripe-go"
	Alias    string   // "Stripe" (import "pkg" as X)
	IsNative bool     // true for Go package imports (no 'from', has 'as')
}

func (i *ImportDecl) TokenLiteral() string { return "import" }

// VarDecl represents a top-level let or const declaration
type VarDecl struct {
	Name    string
	Type    string     // optional, empty = inferred
	Value   Expression // initial value (required)
	IsConst bool       // true for const, false for let
}

func (v *VarDecl) TokenLiteral() string {
	if v.IsConst {
		return "const"
	}
	return "let"
}

// ScriptBlock contains GMX Script code (TypeScript-inspired syntax)
type ScriptBlock struct {
	Source    string         // Raw source (preserved for fallback)
	Imports   []*ImportDecl  // Parsed import declarations
	Models    []*ModelDecl   // Parsed model declarations
	Services  []*ServiceDecl // Parsed service declarations
	Vars      []*VarDecl     // Parsed top-level variable declarations
	Funcs     []*FuncDecl    // Parsed functions
	StartLine int            // Line offset in the .gmx file for source maps
}

func (s *ScriptBlock) TokenLiteral() string { return "script" }

// FuncDecl represents a function declaration
type FuncDecl struct {
	Name       string
	Params     []*Param
	ReturnType string // "error", "string", "bool", etc. Empty if void
	Body       []Statement
	Line       int // Source line for source maps
}

func (f *FuncDecl) TokenLiteral() string { return "func" }

// Param represents a function parameter
type Param struct {
	Name string
	Type string
}

// Statement is the interface for all statements
type Statement interface {
	Node
	statementNode()
}

// Expression is the interface for all expressions
type Expression interface {
	Node
	expressionNode()
}

// ============ STATEMENTS ============

// LetStmt: let x = expr or const x = expr
type LetStmt struct {
	Name  string
	Value Expression
	Const bool // true if declared with 'const'
	Line  int
}

func (l *LetStmt) TokenLiteral() string { return "let" }
func (l *LetStmt) statementNode()       {}

// AssignStmt: x = expr, x.field = expr
type AssignStmt struct {
	Target Expression // Could be Ident or MemberExpr
	Value  Expression
	Line   int
}

func (a *AssignStmt) TokenLiteral() string { return "=" }
func (a *AssignStmt) statementNode()       {}

// ReturnStmt: return expr
type ReturnStmt struct {
	Value Expression // nil for bare return
	Line  int
}

func (r *ReturnStmt) TokenLiteral() string { return "return" }
func (r *ReturnStmt) statementNode()       {}

// IfStmt: if condition { ... } else { ... }
type IfStmt struct {
	Condition   Expression
	Consequence []Statement
	Alternative []Statement // nil if no else
	Line        int
}

func (i *IfStmt) TokenLiteral() string { return "if" }
func (i *IfStmt) statementNode()       {}

// ExprStmt: expression used as statement (e.g. function calls)
type ExprStmt struct {
	Expr Expression
	Line int
}

func (e *ExprStmt) TokenLiteral() string { return e.Expr.TokenLiteral() }
func (e *ExprStmt) statementNode()       {}

// ============ EXPRESSIONS ============

// Ident: variable name
type Ident struct {
	Name string
	Line int
}

func (i *Ident) TokenLiteral() string { return i.Name }
func (i *Ident) expressionNode()      {}

// IntLit: 42
type IntLit struct {
	Value string
	Line  int
}

func (i *IntLit) TokenLiteral() string { return i.Value }
func (i *IntLit) expressionNode()      {}

// FloatLit: 3.14
type FloatLit struct {
	Value string
	Line  int
}

func (f *FloatLit) TokenLiteral() string { return f.Value }
func (f *FloatLit) expressionNode()      {}

// StringLit: "hello" (including interpolation segments)
type StringLit struct {
	Value string       // Raw string value
	Parts []StringPart // For interpolated strings, nil for simple
	Line  int
}

func (s *StringLit) TokenLiteral() string { return s.Value }
func (s *StringLit) expressionNode()      {}

// StringPart represents a segment of an interpolated string
type StringPart struct {
	IsExpr bool
	Text   string     // Literal text (if !IsExpr)
	Expr   Expression // Expression (if IsExpr)
}

// BoolLit: true, false
type BoolLit struct {
	Value bool
	Line  int
}

func (b *BoolLit) TokenLiteral() string {
	if b.Value {
		return "true"
	}
	return "false"
}
func (b *BoolLit) expressionNode() {}

// UnaryExpr: !expr, -expr
type UnaryExpr struct {
	Op      string
	Operand Expression
	Line    int
}

func (u *UnaryExpr) TokenLiteral() string { return u.Op }
func (u *UnaryExpr) expressionNode()      {}

// BinaryExpr: a + b, a == b, a && b
type BinaryExpr struct {
	Left  Expression
	Op    string
	Right Expression
	Line  int
}

func (b *BinaryExpr) TokenLiteral() string { return b.Op }
func (b *BinaryExpr) expressionNode()      {}

// CallExpr: func(args...) — regular function call
type CallExpr struct {
	Function Expression // Could be Ident or MemberExpr
	Args     []Expression
	Line     int
}

func (c *CallExpr) TokenLiteral() string { return "call" }
func (c *CallExpr) expressionNode()      {}

// MemberExpr: obj.field (property access)
type MemberExpr struct {
	Object   Expression
	Property string
	Line     int
}

func (m *MemberExpr) TokenLiteral() string { return "." }
func (m *MemberExpr) expressionNode()      {}

// TryExpr: try expr (unwrap or return error)
type TryExpr struct {
	Expr Expression
	Line int
}

func (t *TryExpr) TokenLiteral() string { return "try" }
func (t *TryExpr) expressionNode()      {}

// RenderExpr: render(model) or render(model1, model2)
type RenderExpr struct {
	Args []Expression
	Line int
}

func (r *RenderExpr) TokenLiteral() string { return "render" }
func (r *RenderExpr) expressionNode()      {}

// ErrorExpr: error("message")
type ErrorExpr struct {
	Message Expression
	Line    int
}

func (e *ErrorExpr) TokenLiteral() string { return "error" }
func (e *ErrorExpr) expressionNode()      {}

// CtxExpr: ctx.field
type CtxExpr struct {
	Field string
	Line  int
}

func (c *CtxExpr) TokenLiteral() string { return "ctx" }
func (c *CtxExpr) expressionNode()      {}

// StructLit: Post{title: title, userId: userId}
type StructLit struct {
	TypeName string
	Fields   map[string]Expression
	Line     int
}

func (s *StructLit) TokenLiteral() string { return s.TypeName }
func (s *StructLit) expressionNode()      {}

// ============ TEMPLATE SECTION ============

// TemplateBlock contains the raw HTML/template content
type TemplateBlock struct {
	Source string // Raw HTML with Go template syntax
}

func (t *TemplateBlock) TokenLiteral() string { return "template" }

// ============ STYLE SECTION ============

// StyleBlock contains the raw CSS
type StyleBlock struct {
	Source string // Raw CSS content
	Scoped bool   // Whether <style scoped> was used
}

func (s *StyleBlock) TokenLiteral() string { return "style" }
