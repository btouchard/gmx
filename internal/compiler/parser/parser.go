package parser

import (
	"fmt"
	"gmx/internal/compiler/ast"
	"gmx/internal/compiler/lexer"
	"gmx/internal/compiler/script"
	"gmx/internal/compiler/token"
	"strings"
)

type Parser struct {
	l         *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
	errors    []string
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) addError(msg string) {
	errMsg := fmt.Sprintf("%d:%d: %s", p.curToken.Pos.Line, p.curToken.Pos.Column, msg)
	p.errors = append(p.errors, errMsg)
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
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
	p.addError(fmt.Sprintf("expected %s, got %s (%q)", t, p.peekToken.Type, p.peekToken.Literal))
	return false
}

// synchronize skips tokens until a top-level statement boundary is found.
// Used for error recovery to avoid infinite loops on malformed input.
func (p *Parser) synchronize() {
	for !p.curTokenIs(token.EOF) {
		// Stop at tokens that can start a new top-level declaration
		switch p.curToken.Type {
		case token.MODEL, token.SERVICE, token.RAW_GO, token.RAW_TEMPLATE, token.RAW_STYLE:
			return
		}
		// Also stop at RBRACE which closes a block
		if p.curTokenIs(token.RBRACE) {
			p.nextToken() // consume the closing brace
			return
		}
		p.nextToken()
	}
}

// ParseGMXFile is the main entry point for parsing a .gmx file
func (p *Parser) ParseGMXFile() *ast.GMXFile {
	file := &ast.GMXFile{
		Models:   []*ast.ModelDecl{},
		Services: []*ast.ServiceDecl{},
	}

	// Parse all tokens - models and sections can appear in any order
	for !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
		case token.MODEL:
			pos := p.curToken.Pos
			model := p.parseModelDecl()
			if model != nil {
				file.Models = append(file.Models, model)
			} else if p.curToken.Pos.Line == pos.Line && p.curToken.Pos.Column == pos.Column {
				// Parse failed without advancing — force progress
				p.nextToken()
				p.synchronize()
			}

		case token.SERVICE:
			pos := p.curToken.Pos
			svc := p.parseServiceDecl()
			if svc != nil {
				file.Services = append(file.Services, svc)
			} else if p.curToken.Pos.Line == pos.Line && p.curToken.Pos.Column == pos.Column {
				p.nextToken()
				p.synchronize()
			}

		case token.RAW_GO:
			source := p.curToken.Literal
			lineOffset := p.curToken.Pos.Line

			// Try to parse the script
			funcs, parseErrors := script.Parse(source, lineOffset)

			scriptBlock := &ast.ScriptBlock{
				Source:    source,
				Funcs:     funcs,
				StartLine: lineOffset,
			}

			// Add parse errors to parser errors (but don't fail - fallback to raw source)
			for _, err := range parseErrors {
				p.errors = append(p.errors, fmt.Sprintf("script parsing: %s", err))
			}

			file.Script = scriptBlock
			p.nextToken()

		case token.RAW_TEMPLATE:
			file.Template = &ast.TemplateBlock{
				Source: p.curToken.Literal,
			}
			p.nextToken()

		case token.RAW_STYLE:
			content := p.curToken.Literal
			scoped := false
			// Check for SCOPED: prefix
			if strings.HasPrefix(content, "SCOPED:") {
				scoped = true
				content = content[len("SCOPED:"):]
			}
			file.Style = &ast.StyleBlock{
				Source: content,
				Scoped: scoped,
			}
			p.nextToken()

		default:
			p.nextToken()
		}
	}

	return file
}

// parseModelDecl parses: model Task { ... }
func (p *Parser) parseModelDecl() *ast.ModelDecl {
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	model := &ast.ModelDecl{
		Name:   p.curToken.Literal,
		Fields: []*ast.FieldDecl{},
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // move past {

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		prevPos := p.curToken.Pos
		field := p.parseFieldDecl()
		if field != nil {
			model.Fields = append(model.Fields, field)
		}
		// Safety: ensure progress
		if p.curToken.Pos.Line == prevPos.Line && p.curToken.Pos.Column == prevPos.Column {
			p.nextToken()
		}
	}

	if !p.curTokenIs(token.RBRACE) {
		p.addError("expected '}' at end of model")
		// Still return the partial model for error recovery
		return model
	}

	p.nextToken() // consume }
	return model
}

// parseFieldDecl parses: title: string @min(3) @max(255)
func (p *Parser) parseFieldDecl() *ast.FieldDecl {
	if !p.curTokenIs(token.IDENT) {
		p.nextToken()
		return nil
	}

	field := &ast.FieldDecl{
		Name:        p.curToken.Literal,
		Annotations: []*ast.Annotation{},
	}

	if !p.expectPeek(token.COLON) {
		return nil
	}
	p.nextToken() // move to type

	// Parse type - could be keyword or IDENT
	// Handle missing type error case
	if p.curTokenIs(token.AT) || p.curTokenIs(token.RBRACE) || p.curTokenIs(token.EOF) {
		p.addError(fmt.Sprintf("expected type after ':', got %s", p.curToken.Type))
		// Still parse annotations if present to avoid hanging
		for p.curTokenIs(token.AT) && !p.curTokenIs(token.EOF) {
			ann := p.parseAnnotation()
			if ann != nil {
				field.Annotations = append(field.Annotations, ann)
			}
		}
		return field
	}

	field.Type = p.curToken.Literal

	// Check for array type: Post[]
	if p.peekTokenIs(token.LBRACKET) {
		p.nextToken() // move to [
		if p.expectPeek(token.RBRACKET) {
			field.Type = field.Type + "[]"
		}
	}

	p.nextToken() // move past type or ]

	// Parse annotations
	for p.curTokenIs(token.AT) && !p.curTokenIs(token.EOF) {
		ann := p.parseAnnotation()
		if ann != nil {
			field.Annotations = append(field.Annotations, ann)
		}
	}

	return field
}

// parseAnnotation parses: @pk, @default(uuid_v4), @relation(references: [id])
func (p *Parser) parseAnnotation() *ast.Annotation {
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	ann := &ast.Annotation{
		Name: p.curToken.Literal,
		Args: make(map[string]string),
	}

	p.nextToken() // move past annotation name

	// Check for arguments
	if p.curTokenIs(token.LPAREN) {
		p.nextToken() // consume (
		p.parseAnnotationArgs(ann)
		if p.curTokenIs(token.RPAREN) {
			p.nextToken() // consume )
		}
	}

	return ann
}

// parseAnnotationArgs parses annotation arguments into the annotation's Args map
func (p *Parser) parseAnnotationArgs(ann *ast.Annotation) {
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		prevPos := p.curToken.Pos
		// Check for named argument: key: value
		if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.COLON) {
			key := p.curToken.Literal
			p.nextToken() // move to :
			p.nextToken() // move past :

			// Parse value
			value := p.parseAnnotationValue()
			ann.Args[key] = value
		} else {
			// Simple argument (no key): store with "_" key
			value := p.parseAnnotationValue()
			ann.Args["_"] = value
		}

		// Skip comma if present
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
		// Safety: ensure progress
		if p.curToken.Pos.Line == prevPos.Line && p.curToken.Pos.Column == prevPos.Column {
			p.nextToken()
		}
	}
}

// parseAnnotationValue parses a single annotation value (could be simple or array)
func (p *Parser) parseAnnotationValue() string {
	// Array value: [id, name]
	if p.curTokenIs(token.LBRACKET) {
		p.nextToken() // consume [
		var parts []string
		for !p.curTokenIs(token.RBRACKET) && !p.curTokenIs(token.EOF) {
			prevPos := p.curToken.Pos
			if !p.curTokenIs(token.COMMA) {
				parts = append(parts, p.curToken.Literal)
			}
			p.nextToken()
			// Safety: ensure progress
			if p.curToken.Pos.Line == prevPos.Line && p.curToken.Pos.Column == prevPos.Column {
				p.nextToken()
			}
		}
		if p.curTokenIs(token.RBRACKET) {
			p.nextToken() // consume ]
		}
		return strings.Join(parts, ", ")
	}

	// String value (may have quotes)
	if p.curTokenIs(token.STRING) {
		val := p.curToken.Literal
		p.nextToken()
		return val
	}

	// Simple value (identifier, number, boolean)
	val := p.curToken.Literal
	p.nextToken()
	return val
}

// parseServiceDecl parses: service Database { provider: "postgres"; url: string @env("DATABASE_URL") }
func (p *Parser) parseServiceDecl() *ast.ServiceDecl {
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	svc := &ast.ServiceDecl{
		Name:     p.curToken.Literal,
		Fields:   []*ast.ServiceField{},
		Methods:  []*ast.ServiceMethod{},
		Provider: "",
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // move past {

	// Parse body
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		prevPos := p.curToken.Pos
		if p.curTokenIs(token.FUNC) {
			// Method declaration
			method := p.parseServiceMethod()
			if method != nil {
				svc.Methods = append(svc.Methods, method)
			}
		} else if p.curTokenIs(token.IDENT) {
			name := p.curToken.Literal
			if name == "provider" {
				// Special field: provider: "value"
				if !p.expectPeek(token.COLON) {
					break
				}
				p.nextToken() // move to value
				svc.Provider = strings.Trim(p.curToken.Literal, "\"")
				p.nextToken() // move past value
			} else {
				// Config field: name: type @env("VAR")
				field := p.parseServiceField()
				if field != nil {
					svc.Fields = append(svc.Fields, field)
				}
			}
		} else {
			p.nextToken() // skip unknown tokens
		}
		// Safety: ensure progress
		if p.curToken.Pos.Line == prevPos.Line && p.curToken.Pos.Column == prevPos.Column {
			p.nextToken()
		}
	}

	if !p.curTokenIs(token.RBRACE) {
		p.addError("expected '}' at end of service")
		// Still return the partial service for error recovery
		return svc
	}
	p.nextToken() // consume }
	return svc
}

// parseServiceField parses: url: string @env("DATABASE_URL")
func (p *Parser) parseServiceField() *ast.ServiceField {
	field := &ast.ServiceField{
		Name:        p.curToken.Literal,
		Annotations: []*ast.Annotation{},
	}

	if !p.expectPeek(token.COLON) {
		return nil
	}
	p.nextToken() // move to type

	field.Type = p.curToken.Literal
	p.nextToken() // move past type

	// Parse annotations (reuse existing parseAnnotation)
	for p.curTokenIs(token.AT) && !p.curTokenIs(token.EOF) {
		ann := p.parseAnnotation()
		if ann != nil {
			field.Annotations = append(field.Annotations, ann)
			// Extract @env special annotation
			if ann.Name == "env" {
				field.EnvVar = strings.Trim(ann.SimpleArg(), "\"")
			}
		}
	}

	return field
}

// parseServiceMethod parses: func send(to: string, subject: string, body: string) error
func (p *Parser) parseServiceMethod() *ast.ServiceMethod {
	// Current token is FUNC
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	method := &ast.ServiceMethod{
		Name:   p.curToken.Literal,
		Params: []*ast.Param{},
	}

	// Parse parameters: (name: type, name: type)
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken() // move past (

	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		prevPos := p.curToken.Pos
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
			continue
		}
		// Parse param: name: type
		if p.curTokenIs(token.IDENT) {
			paramName := p.curToken.Literal
			if !p.expectPeek(token.COLON) {
				break
			}
			p.nextToken() // move to type
			paramType := p.curToken.Literal
			method.Params = append(method.Params, &ast.Param{
				Name: paramName,
				Type: paramType,
			})
			p.nextToken() // move past type
		} else {
			p.nextToken()
		}
		// Safety: ensure progress
		if p.curToken.Pos.Line == prevPos.Line && p.curToken.Pos.Column == prevPos.Column {
			p.nextToken()
		}
	}

	if p.curTokenIs(token.RPAREN) {
		p.nextToken() // consume )
	}

	// Optional return type (can be IDENT or keyword like error, bool, etc.)
	if p.curTokenIs(token.IDENT) || p.curTokenIs(token.ERROR) || p.curTokenIs(token.TRUE) || p.curTokenIs(token.FALSE) {
		method.ReturnType = p.curToken.Literal
		p.nextToken()
	}

	return method
}
