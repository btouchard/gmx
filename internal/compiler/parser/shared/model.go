package shared

import (
	"fmt"
	"github.com/btouchard/gmx/internal/compiler/ast"
	"github.com/btouchard/gmx/internal/compiler/token"
)

// ParseModelDecl parses: model Task { ... }
func (p *ParserCore) ParseModelDecl() *ast.ModelDecl {
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
func (p *ParserCore) parseFieldDecl() *ast.FieldDecl {
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
			ann := p.ParseAnnotation()
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
		ann := p.ParseAnnotation()
		if ann != nil {
			field.Annotations = append(field.Annotations, ann)
		}
	}

	return field
}
