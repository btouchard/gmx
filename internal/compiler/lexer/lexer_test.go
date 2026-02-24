package lexer

import (
	"gmx/internal/compiler/token"
	"strings"
	"testing"
)

func TestBasicTokens(t *testing.T) {
	input := `= + - ! * / % < > ( ) { } [ ] @ : , . ;`

	expected := []token.TokenType{
		token.ASSIGN, token.PLUS, token.MINUS, token.BANG, token.ASTERISK,
		token.SLASH, token.PERCENT, token.LT, token.GT, token.LPAREN, token.RPAREN,
		token.LBRACE, token.RBRACE, token.LBRACKET, token.RBRACKET,
		token.AT, token.COLON, token.COMMA, token.DOT, token.SEMICOLON,
		token.EOF,
	}

	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Fatalf("test[%d] - wrong type. expected=%s, got=%s (literal=%q)", i, exp, tok.Type, tok.Literal)
		}
	}
}

func TestMultiCharOperators(t *testing.T) {
	input := `== != <= >=`

	expected := []struct {
		typ token.TokenType
		lit string
	}{
		{token.EQ, "=="}, {token.NOT_EQ, "!="}, {token.LT_EQ, "<="},
		{token.GT_EQ, ">="},
	}

	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp.typ || tok.Literal != exp.lit {
			t.Fatalf("test[%d] - expected %s(%q), got %s(%q)", i, exp.typ, exp.lit, tok.Type, tok.Literal)
		}
	}
}

func TestKeywords(t *testing.T) {
	input := `func let const if else return true false model service task import as`

	expected := []token.TokenType{
		token.FUNC, token.LET, token.CONST, token.IF, token.ELSE,
		token.RETURN, token.TRUE, token.FALSE, token.MODEL, token.SERVICE,
		token.TASK, token.IMPORT, token.AS,
	}

	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Fatalf("test[%d] - expected %s, got %s(%q)", i, exp, tok.Type, tok.Literal)
		}
	}
}

func TestStrings(t *testing.T) {
	input := `"hello world" "escaped \"quote\"" ` + "`backtick string`"

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.STRING || tok.Literal != "hello world" {
		t.Fatalf("test 1 - got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.STRING || tok.Literal != `escaped \"quote\"` {
		t.Fatalf("test 2 - got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.STRING || tok.Literal != "backtick string" {
		t.Fatalf("test 3 - got %s(%q)", tok.Type, tok.Literal)
	}
}

func TestNumbers(t *testing.T) {
	input := `42 3.14 0 100.5`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.INT || tok.Literal != "42" {
		t.Fatalf("test 1 - got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.FLOAT || tok.Literal != "3.14" {
		t.Fatalf("test 2 - got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.INT || tok.Literal != "0" {
		t.Fatalf("test 3 - got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.FLOAT || tok.Literal != "100.5" {
		t.Fatalf("test 4 - got %s(%q)", tok.Type, tok.Literal)
	}
}

func TestLineComments(t *testing.T) {
	input := "let x // this is a comment\nlet y"

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.LET {
		t.Fatalf("expected LET, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "x" {
		t.Fatalf("expected x, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.LET {
		t.Fatalf("expected LET after comment, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "y" {
		t.Fatalf("expected y, got %s(%q)", tok.Type, tok.Literal)
	}
}

func TestBlockComments(t *testing.T) {
	input := "let /* this\nis\na comment */ x"

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.LET {
		t.Fatalf("expected LET, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "x" {
		t.Fatalf("expected x, got %s(%q)", tok.Type, tok.Literal)
	}
}

func TestScriptTag(t *testing.T) {
	input := `model Task { }

<script>
func test() {}
</script>`

	l := New(input)

	// model keyword
	tok := l.NextToken()
	if tok.Type != token.MODEL {
		t.Fatalf("expected MODEL, got %s", tok.Type)
	}

	// Task
	tok = l.NextToken()
	if tok.Type != token.IDENT {
		t.Fatalf("expected IDENT, got %s", tok.Type)
	}

	// {
	tok = l.NextToken()
	if tok.Type != token.LBRACE {
		t.Fatalf("expected LBRACE, got %s", tok.Type)
	}

	// }
	tok = l.NextToken()
	if tok.Type != token.RBRACE {
		t.Fatalf("expected RBRACE, got %s", tok.Type)
	}

	// <script>...</script>
	tok = l.NextToken()
	if tok.Type != token.RAW_GO {
		t.Fatalf("expected RAW_GO, got %s(%q)", tok.Type, tok.Literal)
	}
	if !strings.Contains(tok.Literal, "func test()") {
		t.Fatalf("RAW_GO missing expected content: %q", tok.Literal)
	}
}

func TestRawGoSection(t *testing.T) {
	input := `model Task { }

<script>
// Pure Go code block
func toggleTask(w http.ResponseWriter, r *http.Request) error {
    return nil
}
</script>

<template></template>`

	l := New(input)

	// Skip model section tokens
	for {
		tok := l.NextToken()
		if tok.Type == token.RAW_GO {
			// Should get RAW_GO
			if !strings.Contains(tok.Literal, "func toggleTask") {
				t.Fatalf("RAW_GO missing expected content: %q", tok.Literal)
			}
			if !strings.Contains(tok.Literal, "http.ResponseWriter") {
				t.Fatalf("RAW_GO missing http.ResponseWriter: %q", tok.Literal)
			}
			break
		}
		if tok.Type == token.EOF {
			t.Fatal("never found RAW_GO")
		}
	}

	// Should get RAW_TEMPLATE
	tok := l.NextToken()
	if tok.Type != token.RAW_TEMPLATE {
		t.Fatalf("expected RAW_TEMPLATE, got %s", tok.Type)
	}
}

func TestRawTemplateSection(t *testing.T) {
	input := `<template>
<div class="task-item">
  <span>{{.Title}}</span>
</div>
</template>

<style>
.task-item { padding: 1rem; }
</style>`

	l := New(input)

	// RAW_TEMPLATE
	tok := l.NextToken()
	if tok.Type != token.RAW_TEMPLATE {
		t.Fatalf("expected RAW_TEMPLATE, got %s(%q)", tok.Type, tok.Literal)
	}
	if !strings.Contains(tok.Literal, "<div") || !strings.Contains(tok.Literal, "{{.Title}}") {
		t.Fatalf("RAW_TEMPLATE missing expected content: %q", tok.Literal)
	}

	// RAW_STYLE
	tok = l.NextToken()
	if tok.Type != token.RAW_STYLE {
		t.Fatalf("expected RAW_STYLE, got %s(%q)", tok.Type, tok.Literal)
	}
	if !strings.Contains(tok.Literal, ".task-item") {
		t.Fatalf("RAW_STYLE missing expected content: %q", tok.Literal)
	}
}

func TestRawStyleSection(t *testing.T) {
	input := `<style>
.task-item {
  padding: 1rem;
  background: #fff;
}
</style>`

	l := New(input)

	// RAW_STYLE
	tok := l.NextToken()
	if tok.Type != token.RAW_STYLE {
		t.Fatalf("expected RAW_STYLE, got %s", tok.Type)
	}
	if !strings.Contains(tok.Literal, ".task-item") || !strings.Contains(tok.Literal, "padding") {
		t.Fatalf("RAW_STYLE missing expected content: %q", tok.Literal)
	}
}

func TestPositionTracking(t *testing.T) {
	input := "let x\nlet y"

	l := New(input)

	tok := l.NextToken() // let
	if tok.Pos.Line != 1 {
		t.Fatalf("expected line 1, got %d", tok.Pos.Line)
	}

	tok = l.NextToken() // x
	tok = l.NextToken() // let (line 2)
	if tok.Pos.Line != 2 {
		t.Fatalf("expected line 2, got %d", tok.Pos.Line)
	}
}

func TestModelBlockWithAnnotations(t *testing.T) {
	input := `model Task {
  id:        uuid   @pk @default(uuid_v4)
  title:     string @min(3) @max(255)
  done:      bool   @default(false)
  tenant_id: uuid   @scoped
  email:     string @email
  author_id: uuid   @relation(references: [id])
}`

	l := New(input)

	// model
	tok := l.NextToken()
	if tok.Type != token.MODEL {
		t.Fatalf("expected MODEL, got %s", tok.Type)
	}

	// Task
	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "Task" {
		t.Fatalf("expected Task, got %s(%q)", tok.Type, tok.Literal)
	}

	// {
	tok = l.NextToken()
	if tok.Type != token.LBRACE {
		t.Fatalf("expected LBRACE, got %s", tok.Type)
	}

	// id field: id : uuid @pk @default(uuid_v4)
	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "id" {
		t.Fatalf("expected id, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.COLON {
		t.Fatalf("expected COLON, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "uuid" {
		t.Fatalf("expected uuid, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.AT {
		t.Fatalf("expected AT, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "pk" {
		t.Fatalf("expected pk, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.AT {
		t.Fatalf("expected AT for @default, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "default" {
		t.Fatalf("expected default, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.LPAREN {
		t.Fatalf("expected LPAREN, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "uuid_v4" {
		t.Fatalf("expected uuid_v4, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.RPAREN {
		t.Fatalf("expected RPAREN, got %s", tok.Type)
	}

	// Continue through the rest...
	tokenCount := 0
	for tok.Type != token.RBRACE && tok.Type != token.EOF {
		tok = l.NextToken()
		tokenCount++
		if tokenCount > 100 {
			t.Fatal("too many tokens, possible infinite loop")
		}
	}

	if tok.Type != token.RBRACE {
		t.Fatal("never found closing brace")
	}
}

func TestCompleteGMXFile(t *testing.T) {
	input := `model Task {
  id:    uuid   @pk @default(uuid_v4)
  title: string @min(3) @max(255)
  done:  bool   @default(false)
}

<script>
// Pure Go code block
func toggleTask(w http.ResponseWriter, r *http.Request) error {
    // handler code
    return nil
}
</script>

<template>
  <div class="task-item">{{.Title}}</div>
</template>

<style scoped>
  .task-item { padding: 1rem; }
</style>`

	l := New(input)
	tokenCount := 0
	illegalCount := 0

	var tokens []token.Token

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		tokenCount++
		if tok.Type == token.ILLEGAL {
			illegalCount++
			t.Logf("ILLEGAL token at %d:%d: %q", tok.Pos.Line, tok.Pos.Column, tok.Literal)
		}
		if tok.Type == token.EOF {
			break
		}
		if tokenCount > 500 {
			t.Fatal("too many tokens, possible infinite loop")
		}
	}

	if illegalCount > 0 {
		t.Fatalf("found %d illegal tokens", illegalCount)
	}

	// Verify we got all sections
	foundModel := false
	foundGo := false
	foundTemplate := false
	foundStyle := false

	for _, tok := range tokens {
		switch tok.Type {
		case token.MODEL:
			foundModel = true
		case token.RAW_GO:
			foundGo = true
			if !strings.Contains(tok.Literal, "func toggleTask") {
				t.Fatalf("RAW_GO missing expected content")
			}
		case token.RAW_TEMPLATE:
			foundTemplate = true
			if !strings.Contains(tok.Literal, "<div") {
				t.Fatalf("RAW_TEMPLATE missing expected content")
			}
		case token.RAW_STYLE:
			foundStyle = true
			if !strings.Contains(tok.Literal, ".task-item") {
				t.Fatalf("RAW_STYLE missing expected content")
			}
			// Check for SCOPED: prefix
			if !strings.HasPrefix(tok.Literal, "SCOPED:") {
				t.Error("expected RAW_STYLE to have SCOPED: prefix")
			}
		}
	}

	if !foundModel {
		t.Error("did not find MODEL keyword")
	}
	if !foundGo {
		t.Error("did not find RAW_GO")
	}
	if !foundTemplate {
		t.Error("did not find RAW_TEMPLATE")
	}
	if !foundStyle {
		t.Error("did not find RAW_STYLE")
	}

	t.Logf("lexed %d tokens successfully", tokenCount)
}

func TestUnicodeIdentifiers(t *testing.T) {
	input := `let café = "french"
let 日本語 = "japanese"`

	l := New(input)

	// let
	tok := l.NextToken()
	if tok.Type != token.LET {
		t.Fatalf("expected LET, got %s", tok.Type)
	}

	// café
	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "café" {
		t.Fatalf("expected café, got %s(%q)", tok.Type, tok.Literal)
	}

	// =
	tok = l.NextToken()
	if tok.Type != token.ASSIGN {
		t.Fatalf("expected ASSIGN, got %s", tok.Type)
	}

	// "french"
	tok = l.NextToken()
	if tok.Type != token.STRING || tok.Literal != "french" {
		t.Fatalf("expected french, got %s(%q)", tok.Type, tok.Literal)
	}

	// let
	tok = l.NextToken()
	if tok.Type != token.LET {
		t.Fatalf("expected LET, got %s", tok.Type)
	}

	// 日本語
	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "日本語" {
		t.Fatalf("expected 日本語, got %s(%q)", tok.Type, tok.Literal)
	}
}

func TestHyphenatedIdentifiers(t *testing.T) {
	// Used in template attributes like hx-get, hx-post, etc.
	input := `hx-get hx-post some-kebab-case`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "hx-get" {
		t.Fatalf("expected hx-get, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "hx-post" {
		t.Fatalf("expected hx-post, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "some-kebab-case" {
		t.Fatalf("expected some-kebab-case, got %s(%q)", tok.Type, tok.Literal)
	}
}

func TestAnnotationWithColonInArgs(t *testing.T) {
	input := `@relation(references: [id])`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.AT {
		t.Fatalf("expected AT, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "relation" {
		t.Fatalf("expected relation, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.LPAREN {
		t.Fatalf("expected LPAREN, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "references" {
		t.Fatalf("expected references, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.COLON {
		t.Fatalf("expected COLON, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.LBRACKET {
		t.Fatalf("expected LBRACKET, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "id" {
		t.Fatalf("expected id, got %s(%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.RBRACKET {
		t.Fatalf("expected RBRACKET, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.RPAREN {
		t.Fatalf("expected RPAREN, got %s", tok.Type)
	}
}

func TestStyleScopedAttribute(t *testing.T) {
	input := `<style scoped>
.task-item { padding: 1rem; }
</style>`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.RAW_STYLE {
		t.Fatalf("expected RAW_STYLE, got %s", tok.Type)
	}

	// Should have SCOPED: prefix
	if !strings.HasPrefix(tok.Literal, "SCOPED:") {
		t.Fatalf("expected SCOPED: prefix, got %q", tok.Literal)
	}

	content := strings.TrimPrefix(tok.Literal, "SCOPED:")
	if !strings.Contains(content, ".task-item") {
		t.Fatalf("expected style content, got %q", content)
	}
}

func TestLessThanInModelNotConfusedWithTag(t *testing.T) {
	input := `model Task {
  count: int
}
let x = 5 < 3`

	l := New(input)

	// Skip to the comparison
	for {
		tok := l.NextToken()
		if tok.Type == token.LT {
			// Should be a LT token, not start of a tag
			break
		}
		if tok.Type == token.EOF {
			t.Fatal("did not find < operator")
		}
	}
}

func TestSectionsInDifferentOrder(t *testing.T) {
	input := `<template>
<div>Hello</div>
</template>

model Task {
  id: uuid @pk
}

<script>
func test() {}
</script>`

	l := New(input)

	foundTemplate := false
	foundModel := false
	foundScript := false

	for {
		tok := l.NextToken()
		if tok.Type == token.EOF {
			break
		}
		if tok.Type == token.RAW_TEMPLATE {
			foundTemplate = true
		}
		if tok.Type == token.MODEL {
			foundModel = true
		}
		if tok.Type == token.RAW_GO {
			foundScript = true
		}
	}

	if !foundTemplate || !foundModel || !foundScript {
		t.Errorf("sections not found: template=%v model=%v script=%v", foundTemplate, foundModel, foundScript)
	}
}

func TestNestedHTMLInTemplate(t *testing.T) {
	input := `<template>
<div class="outer">
  <div class="inner">
    <span>Content</span>
  </div>
</div>
</template>`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.RAW_TEMPLATE {
		t.Fatalf("expected RAW_TEMPLATE, got %s", tok.Type)
	}

	// Should contain all the nested divs
	if !strings.Contains(tok.Literal, "<div class=\"outer\">") {
		t.Error("missing outer div")
	}
	if !strings.Contains(tok.Literal, "<div class=\"inner\">") {
		t.Error("missing inner div")
	}
	if !strings.Contains(tok.Literal, "</div>") {
		t.Error("missing closing tags")
	}
}

func TestEmptySections(t *testing.T) {
	input := `model Task { id: uuid }

<script>
</script>

<template>
</template>`

	l := New(input)

	foundModel := false
	foundScript := false
	foundTemplate := false

	for {
		tok := l.NextToken()
		if tok.Type == token.EOF {
			break
		}
		if tok.Type == token.MODEL {
			foundModel = true
		}
		if tok.Type == token.RAW_GO {
			foundScript = true
		}
		if tok.Type == token.RAW_TEMPLATE {
			foundTemplate = true
		}
	}

	if !foundModel || !foundScript || !foundTemplate {
		t.Errorf("sections not found: model=%v script=%v template=%v", foundModel, foundScript, foundTemplate)
	}
}

// Additional tests for edge cases and uncovered branches

func TestLexFloatNumbers(t *testing.T) {
	input := "3.14 0.5 2.0"
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.FLOAT, "3.14"},
		{token.FLOAT, "0.5"},
		{token.FLOAT, "2.0"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexComments(t *testing.T) {
	input := `// This is a comment
let x = 5 // inline comment
// Another comment
`
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LET, "let"},
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.INT, "5"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexIllegalCharacters(t *testing.T) {
	input := "let x = $ 5"
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LET, "let"},
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.ILLEGAL, "$"},
		{token.INT, "5"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
	}
}

func TestLexAllOperators(t *testing.T) {
	input := "== != <= >= && || + - * / < > !"
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.EQ, "=="},
		{token.NOT_EQ, "!="},
		{token.LT_EQ, "<="},
		{token.GT_EQ, ">="},
		{token.AND, "&&"},
		{token.OR, "||"},
		{token.PLUS, "+"},
		{token.MINUS, "-"},
		{token.ASTERISK, "*"},
		{token.SLASH, "/"},
		{token.LT, "<"},
		{token.GT, ">"},
		{token.BANG, "!"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexStringWithEscapes(t *testing.T) {
	input := `"Hello\nWorld" "Tab\there"`
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.STRING, `Hello\nWorld`}, // Lexer preserves escape sequences as-is
		{token.STRING, `Tab\there`},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexMultilineInput(t *testing.T) {
	input := `func test() error {
	let x = 5
	return nil
}`
	l := New(input)

	// Just check that we can lex it without errors and get proper line numbers
	var lastLine int
	for {
		tok := l.NextToken()
		if tok.Type == token.EOF {
			break
		}
		// Verify position tracking
		if tok.Pos.Line > lastLine {
			lastLine = tok.Pos.Line
		}
	}

	if lastLine == 0 {
		t.Error("Expected line numbers to be tracked")
	}
}

func TestLexNegativeNumbers(t *testing.T) {
	input := "-5 -3.14"
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.MINUS, "-"},
		{token.INT, "5"},
		{token.MINUS, "-"},
		{token.FLOAT, "3.14"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
	}
}

func TestLexBacktickStrings(t *testing.T) {
	input := "`backtick string` `multi\nline`"
	l := New(input)

	tests := []struct {
		expectedType  token.TokenType
		wantMultiline bool
	}{
		{token.STRING, false},
		{token.STRING, true},
		{token.EOF, false},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
	}
}

func TestLexAllDelimiters(t *testing.T) {
	input := "( ) { } [ ] , : . ;"
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LPAREN, "("},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.LBRACKET, "["},
		{token.RBRACKET, "]"},
		{token.COMMA, ","},
		{token.COLON, ":"},
		{token.DOT, "."},
		{token.SEMICOLON, ";"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

// Tests added to reach 95%+ coverage

func TestLexServiceTokens(t *testing.T) {
	input := `service Database { provider: "postgres" }`
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.SERVICE, "service"},
		{token.IDENT, "Database"},
		{token.LBRACE, "{"},
		{token.IDENT, "provider"},
		{token.COLON, ":"},
		{token.STRING, "postgres"},
		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexEmptyString(t *testing.T) {
	input := `""`
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.STRING {
		t.Fatalf("expected STRING, got %s", tok.Type)
	}
	if tok.Literal != "" {
		t.Fatalf("expected empty string, got %q", tok.Literal)
	}
}

func TestLexStringWithNewlines(t *testing.T) {
	input := "`multi\nline\nstring`"
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.STRING {
		t.Fatalf("expected STRING, got %s", tok.Type)
	}
	if !strings.Contains(tok.Literal, "\n") {
		t.Fatalf("expected multiline string, got %q", tok.Literal)
	}
}

func TestLexUnterminatedString(t *testing.T) {
	input := `"unterminated`
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.STRING {
		t.Fatalf("expected STRING even if unterminated, got %s", tok.Type)
	}
	// Should return what it got until EOF
	if tok.Literal != "unterminated" {
		t.Fatalf("expected 'unterminated', got %q", tok.Literal)
	}
}

func TestLexTokensWithoutSpaces(t *testing.T) {
	input := `func(id:uuid)`
	l := New(input)

	tests := []token.TokenType{
		token.FUNC,
		token.LPAREN,
		token.IDENT, // id
		token.COLON,
		token.IDENT, // uuid
		token.RPAREN,
		token.EOF,
	}

	for i, expected := range tests {
		tok := l.NextToken()
		if tok.Type != expected {
			t.Fatalf("tests[%d] - expected %s, got %s (lit=%q)", i, expected, tok.Type, tok.Literal)
		}
	}
}

func TestLexSingleAmpersandIllegal(t *testing.T) {
	input := "a & b"
	l := New(input)

	_ = l.NextToken()    // a
	tok := l.NextToken() // &
	if tok.Type != token.ILLEGAL {
		t.Fatalf("expected single & to be ILLEGAL, got %s", tok.Type)
	}
}

func TestLexSinglePipeIllegal(t *testing.T) {
	input := "a | b"
	l := New(input)

	_ = l.NextToken()    // a
	tok := l.NextToken() // |
	if tok.Type != token.ILLEGAL {
		t.Fatalf("expected single | to be ILLEGAL, got %s", tok.Type)
	}
}

func TestLexAnnotationEnvExample(t *testing.T) {
	input := `@env("DATABASE_URL")`
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.AT, "@"},
		{token.IDENT, "env"},
		{token.LPAREN, "("},
		{token.STRING, "DATABASE_URL"},
		{token.RPAREN, ")"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - expected %s, got %s", i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - expected literal %q, got %q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexAnnotationDefaultUUIDV4(t *testing.T) {
	input := `@default(uuid_v4)`
	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.AT, "@"},
		{token.IDENT, "default"},
		{token.LPAREN, "("},
		{token.IDENT, "uuid_v4"},
		{token.RPAREN, ")"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - expected %s, got %s", i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - expected literal %q, got %q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexAnnotationRelationComplex(t *testing.T) {
	input := `@relation(references: [id])`
	l := New(input)

	// Just check we can lex it correctly
	tok := l.NextToken()
	if tok.Type != token.AT {
		t.Fatalf("expected AT, got %s", tok.Type)
	}
	tok = l.NextToken()
	if tok.Literal != "relation" {
		t.Fatalf("expected 'relation', got %q", tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != token.LPAREN {
		t.Fatalf("expected LPAREN, got %s", tok.Type)
	}
	tok = l.NextToken()
	if tok.Literal != "references" {
		t.Fatalf("expected 'references', got %q", tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != token.COLON {
		t.Fatalf("expected COLON, got %s", tok.Type)
	}
	tok = l.NextToken()
	if tok.Type != token.LBRACKET {
		t.Fatalf("expected LBRACKET, got %s", tok.Type)
	}
	tok = l.NextToken()
	if tok.Literal != "id" {
		t.Fatalf("expected 'id', got %q", tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != token.RBRACKET {
		t.Fatalf("expected RBRACKET, got %s", tok.Type)
	}
	tok = l.NextToken()
	if tok.Type != token.RPAREN {
		t.Fatalf("expected RPAREN, got %s", tok.Type)
	}
}

func TestLexKeywordVsIdentifier(t *testing.T) {
	input := "service services Service"
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.SERVICE {
		t.Fatalf("expected SERVICE keyword, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT {
		t.Fatalf("expected IDENT for 'services', got %s", tok.Type)
	}
	if tok.Literal != "services" {
		t.Fatalf("expected 'services', got %q", tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT {
		t.Fatalf("expected IDENT for 'Service', got %s", tok.Type)
	}
	if tok.Literal != "Service" {
		t.Fatalf("expected 'Service', got %q", tok.Literal)
	}
}

func TestLexUnterminatedBacktickString(t *testing.T) {
	input := "`unterminated"
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.STRING {
		t.Fatalf("expected STRING, got %s", tok.Type)
	}
	// Should return what it got until EOF
	if tok.Literal != "unterminated" {
		t.Fatalf("expected 'unterminated', got %q", tok.Literal)
	}
}

func TestLexNumberFollowedByDot(t *testing.T) {
	// Edge case: 5. vs 5.0
	input := "5 5."
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.INT || tok.Literal != "5" {
		t.Fatalf("expected INT 5, got %s %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.INT || tok.Literal != "5" {
		t.Fatalf("expected INT 5, got %s %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.DOT {
		t.Fatalf("expected DOT after 5, got %s", tok.Type)
	}
}

func TestLexStyleWithoutScoped(t *testing.T) {
	input := `<style>
body { color: red; }
</style>`
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.RAW_STYLE {
		t.Fatalf("expected RAW_STYLE, got %s", tok.Type)
	}

	// Should NOT have SCOPED: prefix
	if strings.HasPrefix(tok.Literal, "SCOPED:") {
		t.Errorf("expected non-scoped style, got SCOPED prefix")
	}
}
