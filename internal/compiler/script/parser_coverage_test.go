package script

import (
	"gmx/internal/compiler/ast"
	"testing"
)

// Tests pour combler les gaps de coverage (fonctions à 0%)

func TestParseFloatLiteral(t *testing.T) {
	input := `func test() float {
		return 3.14
	}`

	result, errors := Parse(input, 0)

	if len(errors) > 0 {
		t.Fatalf("parse errors: %v", errors)
	}

	fn := result.Funcs[0]
	returnStmt, ok := fn.Body[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected ReturnStmt, got %T", fn.Body[0])
	}

	floatLit, ok := returnStmt.Value.(*ast.FloatLit)
	if !ok {
		t.Fatalf("expected FloatLit, got %T", returnStmt.Value)
	}

	if floatLit.Value != "3.14" {
		t.Errorf("expected float value '3.14', got %q", floatLit.Value)
	}
}

func TestParseGroupedExpression(t *testing.T) {
	input := `func test() int {
		return (5 + 3) * 2
	}`

	result, errors := Parse(input, 0)

	if len(errors) > 0 {
		t.Fatalf("parse errors: %v", errors)
	}

	fn := result.Funcs[0]
	returnStmt, ok := fn.Body[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected ReturnStmt, got %T", fn.Body[0])
	}

	// Should parse as: (5 + 3) * 2
	mulExpr, ok := returnStmt.Value.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr for *, got %T", returnStmt.Value)
	}

	if mulExpr.Op != "*" {
		t.Errorf("expected outer op '*', got %q", mulExpr.Op)
	}

	// Left should be the grouped expression (5 + 3)
	addExpr, ok := mulExpr.Left.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr for +, got %T", mulExpr.Left)
	}

	if addExpr.Op != "+" {
		t.Errorf("expected inner op '+', got %q", addExpr.Op)
	}
}

func TestParseBooleanLiterals(t *testing.T) {
	input := `func test() error {
		let x = true
		let y = false
		return nil
	}`

	result, errors := Parse(input, 0)

	if len(errors) > 0 {
		t.Fatalf("parse errors: %v", errors)
	}

	fn := result.Funcs[0]

	// First let: x = true
	let1, ok := fn.Body[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected LetStmt, got %T", fn.Body[0])
	}

	bool1, ok := let1.Value.(*ast.BoolLit)
	if !ok {
		t.Fatalf("expected BoolLit, got %T", let1.Value)
	}

	if !bool1.Value {
		t.Error("expected true, got false")
	}

	// Second let: y = false
	let2, ok := fn.Body[1].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected LetStmt, got %T", fn.Body[1])
	}

	bool2, ok := let2.Value.(*ast.BoolLit)
	if !ok {
		t.Fatalf("expected BoolLit, got %T", let2.Value)
	}

	if bool2.Value {
		t.Error("expected false, got true")
	}
}

func TestParseRenderExpressionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantArgs int
	}{
		{
			"render with no args",
			`func test() error { return render() }`,
			0,
		},
		{
			"render with multiple args",
			`func test() error { return render(task, user, post) }`,
			3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, errors := Parse(tt.input, 0)

			if len(errors) > 0 {
				t.Fatalf("parse errors: %v", errors)
			}

			fn := result.Funcs[0]
			returnStmt, ok := fn.Body[0].(*ast.ReturnStmt)
			if !ok {
				t.Fatalf("expected ReturnStmt, got %T", fn.Body[0])
			}

			renderExpr, ok := returnStmt.Value.(*ast.RenderExpr)
			if !ok {
				t.Fatalf("expected RenderExpr, got %T", returnStmt.Value)
			}

			if len(renderExpr.Args) != tt.wantArgs {
				t.Errorf("expected %d args, got %d", tt.wantArgs, len(renderExpr.Args))
			}
		})
	}
}

func TestParseErrorExpression(t *testing.T) {
	input := `func test() error {
		if x == "" {
			return error("x is required")
		}
		return nil
	}`

	result, errors := Parse(input, 0)

	if len(errors) > 0 {
		t.Fatalf("parse errors: %v", errors)
	}

	fn := result.Funcs[0]
	ifStmt, ok := fn.Body[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("expected IfStmt, got %T", fn.Body[0])
	}

	returnStmt, ok := ifStmt.Consequence[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected ReturnStmt, got %T", ifStmt.Consequence[0])
	}

	errorExpr, ok := returnStmt.Value.(*ast.ErrorExpr)
	if !ok {
		t.Fatalf("expected ErrorExpr, got %T", returnStmt.Value)
	}

	stringLit, ok := errorExpr.Message.(*ast.StringLit)
	if !ok {
		t.Fatalf("expected StringLit, got %T", errorExpr.Message)
	}

	if stringLit.Value != "x is required" {
		t.Errorf("expected message 'x is required', got %q", stringLit.Value)
	}
}

func TestParseCtxExpression(t *testing.T) {
	input := `func test() error {
		let tenantID = ctx.tenant
		let userID = ctx.user
		return nil
	}`

	result, errors := Parse(input, 0)

	if len(errors) > 0 {
		t.Fatalf("parse errors: %v", errors)
	}

	fn := result.Funcs[0]

	// First let: tenantID = ctx.tenant
	let1, ok := fn.Body[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected LetStmt, got %T", fn.Body[0])
	}

	ctxExpr1, ok := let1.Value.(*ast.CtxExpr)
	if !ok {
		t.Fatalf("expected CtxExpr, got %T", let1.Value)
	}

	if ctxExpr1.Field != "tenant" {
		t.Errorf("expected field 'tenant', got %q", ctxExpr1.Field)
	}

	// Second let: userID = ctx.user
	let2, ok := fn.Body[1].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected LetStmt, got %T", fn.Body[1])
	}

	ctxExpr2, ok := let2.Value.(*ast.CtxExpr)
	if !ok {
		t.Fatalf("expected CtxExpr, got %T", let2.Value)
	}

	if ctxExpr2.Field != "user" {
		t.Errorf("expected field 'user', got %q", ctxExpr2.Field)
	}
}

// Complex expressions edge cases

func TestParseComplexBinaryExpressions(t *testing.T) {
	input := `func test() bool {
		return a == b && c != d || e < f
	}`

	result, errors := Parse(input, 0)

	if len(errors) > 0 {
		t.Fatalf("parse errors: %v", errors)
	}

	fn := result.Funcs[0]
	returnStmt, ok := fn.Body[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected ReturnStmt, got %T", fn.Body[0])
	}

	// Verify it's a BinaryExpr (precedence testing)
	_, ok = returnStmt.Value.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", returnStmt.Value)
	}
}

func TestParseChainedMemberAccess(t *testing.T) {
	input := `func test() string {
		return user.profile.name
	}`

	result, errors := Parse(input, 0)

	if len(errors) > 0 {
		t.Fatalf("parse errors: %v", errors)
	}

	fn := result.Funcs[0]
	returnStmt, ok := fn.Body[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected ReturnStmt, got %T", fn.Body[0])
	}

	// Outermost: .name
	member1, ok := returnStmt.Value.(*ast.MemberExpr)
	if !ok {
		t.Fatalf("expected MemberExpr, got %T", returnStmt.Value)
	}

	if member1.Property != "name" {
		t.Errorf("expected property 'name', got %q", member1.Property)
	}

	// Inner: user.profile
	member2, ok := member1.Object.(*ast.MemberExpr)
	if !ok {
		t.Fatalf("expected nested MemberExpr, got %T", member1.Object)
	}

	if member2.Property != "profile" {
		t.Errorf("expected property 'profile', got %q", member2.Property)
	}
}

func TestParseFuncNoParams(t *testing.T) {
	input := `func test() error {
		return nil
	}`

	result, errors := Parse(input, 0)

	if len(errors) > 0 {
		t.Fatalf("parse errors: %v", errors)
	}

	fn := result.Funcs[0]
	if len(fn.Params) != 0 {
		t.Errorf("expected 0 params, got %d", len(fn.Params))
	}

	if fn.ReturnType != "error" {
		t.Errorf("expected return type 'error', got %q", fn.ReturnType)
	}
}

func TestParseIfWithoutElse(t *testing.T) {
	input := `func test() error {
		if x > 5 {
			return error("too big")
		}
		return nil
	}`

	result, errors := Parse(input, 0)

	if len(errors) > 0 {
		t.Fatalf("parse errors: %v", errors)
	}

	fn := result.Funcs[0]
	ifStmt, ok := fn.Body[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("expected IfStmt, got %T", fn.Body[0])
	}

	if len(ifStmt.Alternative) != 0 {
		t.Errorf("expected no else clause, got %d statements", len(ifStmt.Alternative))
	}

	if len(ifStmt.Consequence) == 0 {
		t.Error("expected consequence statements")
	}
}
