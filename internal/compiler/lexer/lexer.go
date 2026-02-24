package lexer

import (
	"github.com/btouchard/gmx/internal/compiler/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	input        string
	position     int  // current offset in input (bytes)
	readPosition int  // next reading position (bytes)
	ch           rune // current character
	line         int  // current line (1-based)
	column       int  // current column (1-based)
	braceDepth   int  // track brace depth to know if we're inside a model block
}

func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
		l.position = l.readPosition
	} else {
		r, size := utf8.DecodeRuneInString(l.input[l.readPosition:])
		l.ch = r
		l.position = l.readPosition
		l.readPosition += size

		if l.ch == '\n' {
			l.line++
			l.column = 0
		} else {
			l.column++
		}
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

func (l *Lexer) currentPos() token.Position {
	return token.Position{
		Line:   l.line,
		Column: l.column,
		Offset: l.position,
	}
}

func (l *Lexer) NextToken() token.Token {
	// Normal tokenization mode
	l.skipWhitespaceAndComments()

	pos := l.currentPos()

	// Check for section tags at top level (brace depth 0)
	if l.ch == '<' && l.braceDepth == 0 {
		if tok := l.trySectionTag(); tok.Type != token.ILLEGAL {
			return tok
		}
	}

	var tok token.Token

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch), Pos: pos}
			l.readChar()
			return tok
		}
		tok = l.makeToken(token.ASSIGN, string(l.ch))
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch), Pos: pos}
			l.readChar()
			return tok
		}
		tok = l.makeToken(token.BANG, string(l.ch))
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: string(ch) + string(l.ch), Pos: pos}
			l.readChar()
			return tok
		}
		tok = l.makeToken(token.ILLEGAL, string(l.ch))
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch), Pos: pos}
			l.readChar()
			return tok
		}
		tok = l.makeToken(token.ILLEGAL, string(l.ch))
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LT_EQ, Literal: string(ch) + string(l.ch), Pos: pos}
			l.readChar()
			return tok
		}
		tok = l.makeToken(token.LT, string(l.ch))
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GT_EQ, Literal: string(ch) + string(l.ch), Pos: pos}
			l.readChar()
			return tok
		}
		tok = l.makeToken(token.GT, string(l.ch))
	case '+':
		tok = l.makeToken(token.PLUS, string(l.ch))
	case '-':
		tok = l.makeToken(token.MINUS, string(l.ch))
	case '*':
		tok = l.makeToken(token.ASTERISK, string(l.ch))
	case '/':
		tok = l.makeToken(token.SLASH, string(l.ch))
	case '%':
		tok = l.makeToken(token.PERCENT, string(l.ch))
	case ':':
		tok = l.makeToken(token.COLON, string(l.ch))
	case ';':
		tok = l.makeToken(token.SEMICOLON, string(l.ch))
	case ',':
		tok = l.makeToken(token.COMMA, string(l.ch))
	case '.':
		tok = l.makeToken(token.DOT, string(l.ch))
	case '(':
		tok = l.makeToken(token.LPAREN, string(l.ch))
	case ')':
		tok = l.makeToken(token.RPAREN, string(l.ch))
	case '{':
		l.braceDepth++
		tok = l.makeToken(token.LBRACE, string(l.ch))
	case '}':
		l.braceDepth--
		tok = l.makeToken(token.RBRACE, string(l.ch))
	case '[':
		tok = l.makeToken(token.LBRACKET, string(l.ch))
	case ']':
		tok = l.makeToken(token.RBRACKET, string(l.ch))
	case '@':
		tok = l.makeToken(token.AT, string(l.ch))
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
		tok.Pos = pos
		return tok
	case '`':
		tok.Type = token.STRING
		tok.Literal = l.readBacktickString()
		tok.Pos = pos
		return tok
	case 0:
		tok.Type = token.EOF
		tok.Literal = ""
		tok.Pos = pos
		return tok
	default:
		if isLetter(l.ch) {
			tok.Pos = pos
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		}
		if isDigit(l.ch) {
			tok.Pos = pos
			lit, isFloat := l.readNumber()
			tok.Literal = lit
			if isFloat {
				tok.Type = token.FLOAT
			} else {
				tok.Type = token.INT
			}
			return tok
		}
		tok = l.makeToken(token.ILLEGAL, string(l.ch))
	}

	l.readChar()
	return tok
}

func (l *Lexer) makeToken(typ token.TokenType, lit string) token.Token {
	return token.Token{
		Type:    typ,
		Literal: lit,
		Pos:     l.currentPos(),
	}
}

// trySectionTag checks if we're at the start of a section tag and lexes it if so
func (l *Lexer) trySectionTag() token.Token {
	// Save position for backtracking
	savedPos := l.position
	savedReadPos := l.readPosition
	savedCh := l.ch
	savedLine := l.line
	savedCol := l.column

	// Must start with '<'
	if l.ch != '<' {
		return token.Token{Type: token.ILLEGAL}
	}

	l.readChar() // consume '<'

	// Try to match "script", "template", or "style"
	tag := ""
	for isLetter(l.ch) {
		tag += string(l.ch)
		l.readChar()
	}

	// Check if we have a valid section tag
	var tokType token.TokenType
	var closingTag string
	scoped := false

	switch tag {
	case "script":
		tokType = token.RAW_GO
		closingTag = "</script>"
	case "template":
		tokType = token.RAW_TEMPLATE
		closingTag = "</template>"
	case "style":
		tokType = token.RAW_STYLE
		closingTag = "</style>"
		// Check for "scoped" attribute
		l.skipWhitespace()
		if l.ch == 's' {
			// Try to match "scoped"
			scopedWord := ""
			tempPos := l.position
			tempReadPos := l.readPosition
			tempCh := l.ch
			for isLetter(l.ch) {
				scopedWord += string(l.ch)
				l.readChar()
			}
			if scopedWord == "scoped" {
				scoped = true
			} else {
				// Backtrack if not "scoped"
				l.position = tempPos
				l.readPosition = tempReadPos
				l.ch = tempCh
			}
		}
	default:
		// Not a section tag, backtrack
		l.position = savedPos
		l.readPosition = savedReadPos
		l.ch = savedCh
		l.line = savedLine
		l.column = savedCol
		return token.Token{Type: token.ILLEGAL}
	}

	// Skip to '>'
	for l.ch != '>' && l.ch != 0 {
		l.readChar()
	}
	if l.ch != '>' {
		// Malformed tag, backtrack
		l.position = savedPos
		l.readPosition = savedReadPos
		l.ch = savedCh
		l.line = savedLine
		l.column = savedCol
		return token.Token{Type: token.ILLEGAL}
	}
	l.readChar() // consume '>'

	// Now read everything until we find the closing tag
	pos := token.Position{Line: savedLine, Column: savedCol, Offset: savedPos}

	content := l.readUntilClosingTag(closingTag)

	// For scoped style, prefix the content with a marker
	if scoped {
		content = "SCOPED:" + content
	}

	return token.Token{
		Type:    tokType,
		Literal: content,
		Pos:     pos,
	}
}

// readUntilClosingTag reads all characters until finding the closing tag
func (l *Lexer) readUntilClosingTag(closingTag string) string {
	start := l.position
	closingLen := len(closingTag)

	for l.ch != 0 {
		// Check if we're at the closing tag
		if l.ch == '<' && l.position+closingLen <= len(l.input) {
			if l.input[l.position:l.position+closingLen] == closingTag {
				// Found closing tag
				content := l.input[start:l.position]
				content = strings.TrimSpace(content)
				// Consume the closing tag
				for i := 0; i < closingLen; i++ {
					l.readChar()
				}
				return content
			}
		}
		l.readChar()
	}

	// EOF reached without finding closing tag
	content := l.input[start:l.position]
	return strings.TrimSpace(content)
}

// skipWhitespace skips only spaces and tabs (not newlines)
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		// Skip whitespace
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.readChar()
		}

		// Skip single-line comments
		if l.ch == '/' && l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			continue
		}

		// Skip multi-line comments
		if l.ch == '/' && l.peekChar() == '*' {
			l.readChar() // consume /
			l.readChar() // consume *
			for {
				if l.ch == 0 {
					break
				}
				if l.ch == '*' && l.peekChar() == '/' {
					l.readChar() // consume *
					l.readChar() // consume /
					break
				}
				l.readChar()
			}
			continue
		}

		break
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.position

	// Read characters while they are letters, digits, underscores, or hyphens
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '-' {
		l.readChar()
	}

	return l.input[start:l.position]
}

func (l *Lexer) readNumber() (string, bool) {
	start := l.position
	isFloat := false

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar() // consume .
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[start:l.position], isFloat
}

func (l *Lexer) readString() string {
	l.readChar() // consume opening "
	start := l.position

	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar() // consume backslash
			if l.ch != 0 {
				l.readChar() // consume escaped character
			}
		} else {
			l.readChar()
		}
	}

	str := l.input[start:l.position]

	if l.ch == '"' {
		l.readChar() // consume closing "
	}

	return str
}

func (l *Lexer) readBacktickString() string {
	l.readChar() // consume opening `
	start := l.position

	for l.ch != '`' && l.ch != 0 {
		l.readChar()
	}

	str := l.input[start:l.position]

	if l.ch == '`' {
		l.readChar() // consume closing `
	}

	return str
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}
