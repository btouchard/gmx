package script

import (
	"fmt"
	"gmx/internal/compiler/ast"
	"gmx/internal/compiler/lexer"
	"gmx/internal/compiler/parser/shared"
	"gmx/internal/compiler/token"
	"strings"
	"unicode"
)

// Precedence levels for Pratt parser
const (
	_ int = iota
	LOWEST
	OR          // ||
	AND         // &&
	EQUALS      // == !=
	LESSGREATER // < > <= >=
	SUM         // + -
	PRODUCT     // * / %
	UNARY       // ! -
	CALL        // . ()
)

var precedences = map[token.TokenType]int{
	token.OR:       OR,
	token.AND:      AND,
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.LT_EQ:    LESSGREATER,
	token.GT_EQ:    LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.ASTERISK: PRODUCT,
	token.SLASH:    PRODUCT,
	token.PERCENT:  PRODUCT,
	token.DOT:      CALL,
	token.LPAREN:   CALL,
}

type Parser struct {
	l          *lexer.Lexer
	curToken   token.Token
	peekToken  token.Token
	errors     []string
	lineOffset int // offset to add for source maps

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

// ParseResult contains all parsed declarations from a script block
type ParseResult struct {
	Imports  []*ast.ImportDecl
	Models   []*ast.ModelDecl
	Services []*ast.ServiceDecl
	Vars     []*ast.VarDecl
	Funcs    []*ast.FuncDecl
}

// Parse takes the raw script source and returns parsed declarations (models, services, functions)
func Parse(source string, lineOffset int) (*ParseResult, []string) {
	l := lexer.New(source)
	p := &Parser{
		l:          l,
		lineOffset: lineOffset,
		errors:     []string{},
	}

	// Initialize prefix parsers
	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.TASK, p.parseIdentifier) // TASK is a keyword in GMX but an identifier in scripts
	p.registerPrefix(token.INT, p.parseIntLiteral)
	p.registerPrefix(token.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(token.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(token.BANG, p.parseUnaryExpression)
	p.registerPrefix(token.MINUS, p.parseUnaryExpression)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.TRY, p.parseTryExpression)
	p.registerPrefix(token.RENDER, p.parseRenderExpression)
	p.registerPrefix(token.ERROR, p.parseErrorExpression)
	p.registerPrefix(token.CTX, p.parseCtxExpression)

	// Initialize infix parsers
	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseBinaryExpression)
	p.registerInfix(token.MINUS, p.parseBinaryExpression)
	p.registerInfix(token.ASTERISK, p.parseBinaryExpression)
	p.registerInfix(token.SLASH, p.parseBinaryExpression)
	p.registerInfix(token.PERCENT, p.parseBinaryExpression)
	p.registerInfix(token.EQ, p.parseBinaryExpression)
	p.registerInfix(token.NOT_EQ, p.parseBinaryExpression)
	p.registerInfix(token.LT, p.parseBinaryExpression)
	p.registerInfix(token.GT, p.parseBinaryExpression)
	p.registerInfix(token.LT_EQ, p.parseBinaryExpression)
	p.registerInfix(token.GT_EQ, p.parseBinaryExpression)
	p.registerInfix(token.AND, p.parseBinaryExpression)
	p.registerInfix(token.OR, p.parseBinaryExpression)
	p.registerInfix(token.DOT, p.parseMemberExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)

	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()

	// Parse all top-level declarations (import, model, service, let, const, func)
	result := &ParseResult{
		Imports:  []*ast.ImportDecl{},
		Models:   []*ast.ModelDecl{},
		Services: []*ast.ServiceDecl{},
		Vars:     []*ast.VarDecl{},
		Funcs:    []*ast.FuncDecl{},
	}

	// Track if we've seen non-import declarations for ordering validation
	hasNonImport := false

	for p.curToken.Type != token.EOF {
		switch p.curToken.Type {
		case token.IMPORT:
			// Validate that imports come before other declarations
			if hasNonImport {
				p.error("import declarations must appear before model, service, func, let, or const declarations")
				p.nextToken()
				continue
			}
			importDecl := p.parseImportDecl()
			if importDecl != nil {
				result.Imports = append(result.Imports, importDecl)
			}
			p.nextToken()

		case token.MODEL:
			hasNonImport = true
			model := p.parseModelDecl()
			if model != nil {
				result.Models = append(result.Models, model)
			}
			// parseModelDecl already consumes the closing brace and moves past it

		case token.SERVICE:
			hasNonImport = true
			svc := p.parseServiceDecl()
			if svc != nil {
				result.Services = append(result.Services, svc)
			}
			// parseServiceDecl already consumes the closing brace and moves past it

		case token.LET:
			hasNonImport = true
			varDecl := p.parseVarDecl(false)
			if varDecl != nil {
				result.Vars = append(result.Vars, varDecl)
			}
			p.nextToken() // Move past the declaration

		case token.CONST:
			hasNonImport = true
			varDecl := p.parseVarDecl(true)
			if varDecl != nil {
				result.Vars = append(result.Vars, varDecl)
			}
			p.nextToken() // Move past the declaration

		case token.FUNC:
			hasNonImport = true
			fn := p.parseFuncDecl()
			if fn != nil {
				result.Funcs = append(result.Funcs, fn)
			}
			p.nextToken() // Move past the closing brace

		default:
			p.error(fmt.Sprintf("expected import, model, service, let, const, or func declaration, got %s", p.curToken.Type))
			p.nextToken()
		}
	}

	return result, p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) error(msg string) {
	p.errors = append(p.errors, fmt.Sprintf("line %d: %s", p.curToken.Pos.Line, msg))
}

func (p *Parser) peekError(t token.TokenType) {
	p.error(fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type))
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// ============ IMPORT DECLARATION ============

// parseImportDecl handles three import syntaxes:
// 1. Default: import TaskItem from './components/TaskItem.gmx'
// 2. Destructured: import { sendEmail, MailerConfig } from './services/mailer.gmx'
// 3. Native Go: import "github.com/stripe/stripe-go" as Stripe
func (p *Parser) parseImportDecl() *ast.ImportDecl {
	// Move past 'import' keyword
	p.nextToken()

	// Detect which syntax based on current token
	switch p.curToken.Type {
	case token.LBRACE:
		// Syntax 2: Destructured import { x, y } from '...'
		return p.parseDestructuredImport()

	case token.STRING:
		// Syntax 3: Native Go import "pkg" as Alias
		return p.parseNativeImport()

	case token.IDENT:
		// Syntax 1: Default import X from '...'
		return p.parseDefaultImport()

	default:
		p.error(fmt.Sprintf("expected '{', string, or identifier after 'import', got %s", p.curToken.Type))
		return nil
	}
}

// parseDefaultImport parses: import TaskItem from './path.gmx'
func (p *Parser) parseDefaultImport() *ast.ImportDecl {
	importDecl := &ast.ImportDecl{}

	// Current token is the default import name
	importDecl.Default = p.curToken.Literal

	// Expect 'from' keyword (contextual - check IDENT with literal "from")
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	if p.curToken.Literal != "from" {
		p.error(fmt.Sprintf("expected 'from' after default import name, got %s", p.curToken.Literal))
		return nil
	}

	// Expect path string
	if !p.expectPeek(token.STRING) {
		return nil
	}
	importDecl.Path = p.curToken.Literal

	return importDecl
}

// parseDestructuredImport parses: import { x, y } from './path.gmx'
func (p *Parser) parseDestructuredImport() *ast.ImportDecl {
	importDecl := &ast.ImportDecl{
		Members: []string{},
	}

	// Current token is '{'
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	// Parse first member
	importDecl.Members = append(importDecl.Members, p.curToken.Literal)

	// Parse remaining members
	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // consume comma
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		importDecl.Members = append(importDecl.Members, p.curToken.Literal)
	}

	// Expect '}'
	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	// Expect 'from' keyword (contextual - check IDENT with literal "from")
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	if p.curToken.Literal != "from" {
		p.error(fmt.Sprintf("expected 'from' after destructured import, got %s", p.curToken.Literal))
		return nil
	}

	// Expect path string
	if !p.expectPeek(token.STRING) {
		return nil
	}
	importDecl.Path = p.curToken.Literal

	return importDecl
}

// parseNativeImport parses: import "github.com/pkg" as Alias
func (p *Parser) parseNativeImport() *ast.ImportDecl {
	importDecl := &ast.ImportDecl{
		IsNative: true,
	}

	// Current token is the package path string
	importDecl.Path = p.curToken.Literal

	// Expect 'as' keyword (check for either token.AS or IDENT with literal "as")
	p.nextToken()
	if p.curToken.Type == token.AS || (p.curToken.Type == token.IDENT && p.curToken.Literal == "as") {
		// Good, we have 'as'
	} else {
		p.error(fmt.Sprintf("expected 'as' after package path in native import, got %s", p.curToken.Type))
		return nil
	}

	// Expect alias name
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	importDecl.Alias = p.curToken.Literal

	return importDecl
}

// ============ FUNCTION DECLARATION ============

func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	fn := &ast.FuncDecl{
		Line: p.curToken.Pos.Line + p.lineOffset,
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	fn.Name = p.curToken.Literal

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	fn.Params = p.parseFuncParams()

	// Check for return type (can be IDENT or keyword like ERROR, STRING, etc.)
	if p.peekTokenIs(token.IDENT) || p.isTypeToken(p.peekToken.Type) {
		p.nextToken()
		fn.ReturnType = p.curToken.Literal
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	fn.Body = p.parseBlockStatement()

	// parseBlockStatement leaves us at RBRACE, consume it
	if !p.curTokenIs(token.RBRACE) {
		p.error(fmt.Sprintf("expected } after function body, got %s", p.curToken.Type))
		return nil
	}

	return fn
}

func (p *Parser) parseFuncParams() []*ast.Param {
	params := []*ast.Param{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()

	// Parse first param
	param := &ast.Param{}
	if !p.curTokenIs(token.IDENT) {
		p.error(fmt.Sprintf("expected parameter name, got %s", p.curToken.Type))
		return nil
	}
	param.Name = p.curToken.Literal

	if !p.expectPeek(token.COLON) {
		return nil
	}

	if !p.expectPeek(token.IDENT) && !p.expectPeekType() {
		return nil
	}
	param.Type = p.curToken.Literal

	params = append(params, param)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()

		param := &ast.Param{}
		if !p.curTokenIs(token.IDENT) {
			p.error(fmt.Sprintf("expected parameter name, got %s", p.curToken.Type))
			return nil
		}
		param.Name = p.curToken.Literal

		if !p.expectPeek(token.COLON) {
			return nil
		}

		if !p.expectPeek(token.IDENT) && !p.expectPeekType() {
			return nil
		}
		param.Type = p.curToken.Literal

		params = append(params, param)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return params
}

// isTypeToken returns true if the token type represents a type keyword
func (p *Parser) isTypeToken(t token.TokenType) bool {
	return t == token.ERROR || t == token.STRING || t == token.IDENT || t == token.TASK
}

// expectPeekType expects the next token to be a type token and advances if so
func (p *Parser) expectPeekType() bool {
	if p.isTypeToken(p.peekToken.Type) {
		p.nextToken()
		return true
	}
	return false
}

// parseVarDecl parses a top-level let or const declaration
// Syntax: let name: type = value or let name = value
func (p *Parser) parseVarDecl(isConst bool) *ast.VarDecl {
	varDecl := &ast.VarDecl{
		IsConst: isConst,
	}

	// Expect variable name
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	varDecl.Name = p.curToken.Literal

	// Check for optional type annotation: : type
	if p.peekTokenIs(token.COLON) {
		p.nextToken() // consume ':'

		// Expect type name
		if !p.expectPeek(token.IDENT) && !p.expectPeekType() {
			p.error(fmt.Sprintf("expected type after ':', got %s", p.peekToken.Type))
			return nil
		}
		varDecl.Type = p.curToken.Literal
	}

	// Expect '='
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	// Parse initial value
	p.nextToken()
	varDecl.Value = p.parseExpression(LOWEST)

	if varDecl.Value == nil {
		p.error("expected initial value for variable declaration")
		return nil
	}

	return varDecl
}

// ============ STATEMENTS ============

func (p *Parser) parseBlockStatement() []ast.Statement {
	statements := []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		p.nextToken()
	}

	return statements
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement(false)
	case token.CONST:
		return p.parseLetStatement(true)
	case token.RETURN:
		return p.parseReturnStatement()
	case token.IF:
		return p.parseIfStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseLetStatement(isConst bool) *ast.LetStmt {
	stmt := &ast.LetStmt{
		Line:  p.curToken.Pos.Line + p.lineOffset,
		Const: isConst,
	}

	p.nextToken()

	// Accept IDENT or TASK (since task is a GMX keyword but valid variable name in scripts)
	if !p.curTokenIs(token.IDENT) && !p.curTokenIs(token.TASK) {
		p.error(fmt.Sprintf("expected variable name, got %s", p.curToken.Type))
		return nil
	}

	stmt.Name = p.curToken.Literal

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStmt {
	stmt := &ast.ReturnStmt{
		Line: p.curToken.Pos.Line + p.lineOffset,
	}

	p.nextToken()

	// Check if it's a bare return
	if p.curTokenIs(token.RBRACE) || p.curTokenIs(token.EOF) {
		return stmt
	}

	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseIfStatement() *ast.IfStmt {
	stmt := &ast.IfStmt{
		Line: p.curToken.Pos.Line + p.lineOffset,
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		stmt.Alternative = p.parseBlockStatement()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() ast.Statement {
	line := p.curToken.Pos.Line + p.lineOffset
	expr := p.parseExpression(LOWEST)

	if expr == nil {
		return nil
	}

	// Check if this is an assignment
	if p.peekTokenIs(token.ASSIGN) {
		p.nextToken() // consume '='
		p.nextToken() // move to value expression

		return &ast.AssignStmt{
			Target: expr,
			Value:  p.parseExpression(LOWEST),
			Line:   line,
		}
	}

	return &ast.ExprStmt{
		Expr: expr,
		Line: line,
	}
}

// ============ EXPRESSIONS ============

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.error(fmt.Sprintf("no prefix parse function for %s", p.curToken.Type))
		return nil
	}

	leftExp := prefix()

	// Stop at statement terminators
	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) &&
	    !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expression {
	ident := &ast.Ident{
		Name: p.curToken.Literal,
		Line: p.curToken.Pos.Line + p.lineOffset,
	}

	// Check for struct literal: TypeName{...}
	if p.peekTokenIs(token.LBRACE) && len(p.curToken.Literal) > 0 && unicode.IsUpper(rune(p.curToken.Literal[0])) {
		return p.parseStructLiteral(p.curToken.Literal)
	}

	return ident
}

func (p *Parser) parseStructLiteral(typeName string) ast.Expression {
	lit := &ast.StructLit{
		TypeName: typeName,
		Fields:   make(map[string]ast.Expression),
		Line:     p.curToken.Pos.Line + p.lineOffset,
	}

	p.nextToken() // consume '{'

	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return lit
	}

	p.nextToken()

	// Parse first field
	if !p.curTokenIs(token.IDENT) {
		p.error(fmt.Sprintf("expected field name, got %s", p.curToken.Type))
		return nil
	}

	fieldName := p.curToken.Literal

	if !p.expectPeek(token.COLON) {
		return nil
	}

	p.nextToken()
	lit.Fields[fieldName] = p.parseExpression(LOWEST)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to field name

		if !p.curTokenIs(token.IDENT) {
			p.error(fmt.Sprintf("expected field name, got %s", p.curToken.Type))
			return nil
		}

		fieldName := p.curToken.Literal

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		lit.Fields[fieldName] = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return lit
}

func (p *Parser) parseIntLiteral() ast.Expression {
	return &ast.IntLit{
		Value: p.curToken.Literal,
		Line:  p.curToken.Pos.Line + p.lineOffset,
	}
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	return &ast.FloatLit{
		Value: p.curToken.Literal,
		Line:  p.curToken.Pos.Line + p.lineOffset,
	}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	lit := &ast.StringLit{
		Value: p.curToken.Literal,
		Line:  p.curToken.Pos.Line + p.lineOffset,
	}

	// Parse string interpolation
	if strings.Contains(p.curToken.Literal, "{") {
		lit.Parts = p.parseStringInterpolation(p.curToken.Literal)
	}

	return lit
}

func (p *Parser) parseStringInterpolation(s string) []ast.StringPart {
	parts := []ast.StringPart{}

	i := 0
	for i < len(s) {
		// Find next '{'
		start := i
		for i < len(s) && s[i] != '{' {
			i++
		}

		// Add text before '{'
		if i > start {
			parts = append(parts, ast.StringPart{
				IsExpr: false,
				Text:   s[start:i],
			})
		}

		if i >= len(s) {
			break
		}

		// Find matching '}'
		i++ // skip '{'
		exprStart := i
		braceCount := 1
		for i < len(s) && braceCount > 0 {
			if s[i] == '{' {
				braceCount++
			} else if s[i] == '}' {
				braceCount--
			}
			if braceCount > 0 {
				i++
			}
		}

		if braceCount == 0 {
			// Parse the expression
			exprText := s[exprStart:i]
			expr := p.parseExpressionFromString(exprText)
			if expr != nil {
				parts = append(parts, ast.StringPart{
					IsExpr: true,
					Expr:   expr,
				})
			}
			i++ // skip '}'
		}
	}

	return parts
}

func (p *Parser) parseExpressionFromString(s string) ast.Expression {
	// Create a sub-parser for the expression
	subLexer := lexer.New(s)
	subParser := &Parser{
		l:          subLexer,
		lineOffset: p.lineOffset,
		errors:     []string{},
	}

	// Copy the prefix and infix parsers
	subParser.prefixParseFns = p.prefixParseFns
	subParser.infixParseFns = p.infixParseFns

	// Initialize tokens
	subParser.nextToken()
	subParser.nextToken()

	return subParser.parseExpression(LOWEST)
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BoolLit{
		Value: p.curToken.Type == token.TRUE,
		Line:  p.curToken.Pos.Line + p.lineOffset,
	}
}

func (p *Parser) parseUnaryExpression() ast.Expression {
	expr := &ast.UnaryExpr{
		Op:   p.curToken.Literal,
		Line: p.curToken.Pos.Line + p.lineOffset,
	}

	p.nextToken()

	expr.Operand = p.parseExpression(UNARY)

	return expr
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseTryExpression() ast.Expression {
	expr := &ast.TryExpr{
		Line: p.curToken.Pos.Line + p.lineOffset,
	}

	p.nextToken()

	expr.Expr = p.parseExpression(UNARY)

	return expr
}

func (p *Parser) parseRenderExpression() ast.Expression {
	expr := &ast.RenderExpr{
		Line: p.curToken.Pos.Line + p.lineOffset,
		Args: []ast.Expression{},
	}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return expr
	}

	p.nextToken()
	expr.Args = append(expr.Args, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		expr.Args = append(expr.Args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return expr
}

func (p *Parser) parseErrorExpression() ast.Expression {
	expr := &ast.ErrorExpr{
		Line: p.curToken.Pos.Line + p.lineOffset,
	}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()

	expr.Message = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return expr
}

func (p *Parser) parseCtxExpression() ast.Expression {
	expr := &ast.CtxExpr{
		Line: p.curToken.Pos.Line + p.lineOffset,
	}

	if !p.expectPeek(token.DOT) {
		return nil
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	expr.Field = p.curToken.Literal

	return expr
}

func (p *Parser) parseBinaryExpression(left ast.Expression) ast.Expression {
	expr := &ast.BinaryExpr{
		Left:  left,
		Op:    p.curToken.Literal,
		Line:  p.curToken.Pos.Line + p.lineOffset,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

func (p *Parser) parseMemberExpression(left ast.Expression) ast.Expression {
	expr := &ast.MemberExpr{
		Object: left,
		Line:   p.curToken.Pos.Line + p.lineOffset,
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	expr.Property = p.curToken.Literal

	return expr
}

func (p *Parser) parseCallExpression(left ast.Expression) ast.Expression {
	expr := &ast.CallExpr{
		Function: left,
		Line:     p.curToken.Pos.Line + p.lineOffset,
		Args:     []ast.Expression{},
	}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return expr
	}

	p.nextToken()
	expr.Args = append(expr.Args, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		expr.Args = append(expr.Args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return expr
}

// ============ MODEL/SERVICE PARSING (delegated to shared package) ============

// parseModelDecl delegates model parsing to the shared package
func (p *Parser) parseModelDecl() *ast.ModelDecl {
	// Create a shared parser core that wraps our current state
	core := shared.NewParserCoreFromTokens(p.l, p.curToken, p.peekToken)

	// Delegate to shared package
	model := core.ParseModelDecl()

	// Sync our state back from the core
	p.curToken = core.GetCurrentToken()
	p.peekToken = core.GetPeekToken()

	// Merge errors
	for _, err := range core.Errors() {
		p.errors = append(p.errors, err)
	}

	return model
}

// parseServiceDecl delegates service parsing to the shared package
func (p *Parser) parseServiceDecl() *ast.ServiceDecl {
	// Create a shared parser core that wraps our current state
	core := shared.NewParserCoreFromTokens(p.l, p.curToken, p.peekToken)

	// Delegate to shared package
	svc := core.ParseServiceDecl()

	// Sync our state back from the core
	p.curToken = core.GetCurrentToken()
	p.peekToken = core.GetPeekToken()

	// Merge errors
	for _, err := range core.Errors() {
		p.errors = append(p.errors, err)
	}

	return svc
}
