package ast

import "testing"

func TestTokenLiterals(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		expected string
	}{
		{"GMXFile", &GMXFile{}, "gmx"},
		{"ModelDecl", &ModelDecl{Name: "Task"}, "model"},
		{"FieldDecl", &FieldDecl{Name: "title"}, "title"},
		{"ServiceDecl", &ServiceDecl{Name: "Database"}, "service"},
		{"ServiceField", &ServiceField{Name: "url"}, "url"},
		{"ServiceMethod", &ServiceMethod{Name: "send"}, "send"},
		{"Annotation @pk", &Annotation{Name: "pk"}, "@pk"},
		{"Annotation @default", &Annotation{Name: "default"}, "@default"},
		{"ScriptBlock", &ScriptBlock{}, "script"},
		{"FuncDecl", &FuncDecl{Name: "test"}, "func"},
		{"LetStmt", &LetStmt{Name: "x"}, "let"},
		{"AssignStmt", &AssignStmt{}, "="},
		{"ReturnStmt", &ReturnStmt{}, "return"},
		{"IfStmt", &IfStmt{}, "if"},
		{"ExprStmt", &ExprStmt{Expr: &Ident{Name: "x"}}, "x"},
		{"Ident", &Ident{Name: "task"}, "task"},
		{"IntLit", &IntLit{Value: "42"}, "42"},
		{"FloatLit", &FloatLit{Value: "3.14"}, "3.14"},
		{"StringLit", &StringLit{Value: "hello"}, "hello"},
		{"BoolLit true", &BoolLit{Value: true}, "true"},
		{"BoolLit false", &BoolLit{Value: false}, "false"},
		{"UnaryExpr", &UnaryExpr{Op: "!"}, "!"},
		{"BinaryExpr", &BinaryExpr{Op: "+"}, "+"},
		{"CallExpr", &CallExpr{}, "call"},
		{"MemberExpr", &MemberExpr{Property: "name"}, "."},
		{"TryExpr", &TryExpr{}, "try"},
		{"RenderExpr", &RenderExpr{}, "render"},
		{"ErrorExpr", &ErrorExpr{}, "error"},
		{"CtxExpr", &CtxExpr{Field: "tenant"}, "ctx"},
		{"StructLit", &StructLit{TypeName: "Task"}, "Task"},
		{"TemplateBlock", &TemplateBlock{}, "template"},
		{"StyleBlock", &StyleBlock{}, "style"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.TokenLiteral()
			if result != tt.expected {
				t.Errorf("TokenLiteral() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAnnotationSimpleArg(t *testing.T) {
	tests := []struct {
		name     string
		ann      *Annotation
		expected string
	}{
		{
			"default with simple arg",
			&Annotation{Name: "default", Args: map[string]string{"_": "uuid_v4"}},
			"uuid_v4",
		},
		{
			"min with simple arg",
			&Annotation{Name: "min", Args: map[string]string{"_": "3"}},
			"3",
		},
		{
			"annotation without simple arg",
			&Annotation{Name: "relation", Args: map[string]string{"references": "id"}},
			"",
		},
		{
			"annotation with empty args",
			&Annotation{Name: "pk", Args: map[string]string{}},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ann.SimpleArg()
			if result != tt.expected {
				t.Errorf("SimpleArg() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStatementNodes(t *testing.T) {
	// Just verify that statement interface is implemented
	var _ Statement = (*LetStmt)(nil)
	var _ Statement = (*AssignStmt)(nil)
	var _ Statement = (*ReturnStmt)(nil)
	var _ Statement = (*IfStmt)(nil)
	var _ Statement = (*ExprStmt)(nil)
}

func TestExpressionNodes(t *testing.T) {
	// Just verify that expression interface is implemented
	var _ Expression = (*Ident)(nil)
	var _ Expression = (*IntLit)(nil)
	var _ Expression = (*FloatLit)(nil)
	var _ Expression = (*StringLit)(nil)
	var _ Expression = (*BoolLit)(nil)
	var _ Expression = (*UnaryExpr)(nil)
	var _ Expression = (*BinaryExpr)(nil)
	var _ Expression = (*CallExpr)(nil)
	var _ Expression = (*MemberExpr)(nil)
	var _ Expression = (*TryExpr)(nil)
	var _ Expression = (*RenderExpr)(nil)
	var _ Expression = (*ErrorExpr)(nil)
	var _ Expression = (*CtxExpr)(nil)
	var _ Expression = (*StructLit)(nil)
}
