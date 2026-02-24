package shared

import (
	"fmt"
	"gmx/internal/compiler/lexer"
	"gmx/internal/compiler/token"
)

// ParserCore contains shared parsing utilities used by both main parser and script parser
type ParserCore struct {
	l         *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
	errors    []string
}

// NewParserCore creates a new parser core from a lexer
func NewParserCore(l *lexer.Lexer) *ParserCore {
	p := &ParserCore{
		l:      l,
		errors: []string{},
	}
	p.nextToken()
	p.nextToken()
	return p
}

// NewParserCoreFromTokens creates a parser core from existing token state
// Used by script parser to delegate to shared parsing logic
func NewParserCoreFromTokens(l *lexer.Lexer, cur, peek token.Token) *ParserCore {
	return &ParserCore{
		l:         l,
		curToken:  cur,
		peekToken: peek,
		errors:    []string{},
	}
}

func (p *ParserCore) Errors() []string {
	return p.errors
}

func (p *ParserCore) addError(msg string) {
	errMsg := fmt.Sprintf("%d:%d: %s", p.curToken.Pos.Line, p.curToken.Pos.Column, msg)
	p.errors = append(p.errors, errMsg)
}

func (p *ParserCore) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *ParserCore) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *ParserCore) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *ParserCore) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("expected %s, got %s (%q)", t, p.peekToken.Type, p.peekToken.Literal))
	return false
}

// GetCurrentToken returns the current token (for delegation back to parent parser)
func (p *ParserCore) GetCurrentToken() token.Token {
	return p.curToken
}

// GetPeekToken returns the peek token (for delegation back to parent parser)
func (p *ParserCore) GetPeekToken() token.Token {
	return p.peekToken
}
