package shared

import (
	"gmx/internal/compiler/ast"
	"gmx/internal/compiler/token"
	"strings"
)

// ParseAnnotation parses: @pk, @default(uuid_v4), @relation(references: [id])
func (p *ParserCore) ParseAnnotation() *ast.Annotation {
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
func (p *ParserCore) parseAnnotationArgs(ann *ast.Annotation) {
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
func (p *ParserCore) parseAnnotationValue() string {
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
