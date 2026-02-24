package shared

import (
	"github.com/btouchard/gmx/internal/compiler/ast"
	"github.com/btouchard/gmx/internal/compiler/token"
	"strings"
)

// ParseServiceDecl parses: service Database { provider: "postgres"; url: string @env("DATABASE_URL") }
func (p *ParserCore) ParseServiceDecl() *ast.ServiceDecl {
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
func (p *ParserCore) parseServiceField() *ast.ServiceField {
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

	// Parse annotations (reuse existing ParseAnnotation)
	for p.curTokenIs(token.AT) && !p.curTokenIs(token.EOF) {
		ann := p.ParseAnnotation()
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
func (p *ParserCore) parseServiceMethod() *ast.ServiceMethod {
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
