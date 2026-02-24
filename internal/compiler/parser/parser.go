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
		case token.RAW_GO, token.RAW_TEMPLATE, token.RAW_STYLE:
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

	// Parse all sections
	for !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
		case token.RAW_GO:
			source := p.curToken.Literal
			lineOffset := p.curToken.Pos.Line

			// Parse the script using enhanced script parser
			result, parseErrors := script.Parse(source, lineOffset)

			scriptBlock := &ast.ScriptBlock{
				Source:    source,
				Models:    result.Models,
				Services:  result.Services,
				Funcs:     result.Funcs,
				StartLine: lineOffset,
			}

			// Add parse errors to parser errors (but don't fail - fallback to raw source)
			for _, err := range parseErrors {
				p.errors = append(p.errors, fmt.Sprintf("script parsing: %s", err))
			}

			file.Script = scriptBlock

			// Extract models and services to top-level for generator compatibility
			file.Models = append(file.Models, result.Models...)
			file.Services = append(file.Services, result.Services...)

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
