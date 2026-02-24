package token

type TokenType string

type Position struct {
	Line   int
	Column int
	Offset int
}

type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}

const (
	// Special
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"
	COMMENT TokenType = "COMMENT"

	// Identifiers + literals
	IDENT  TokenType = "IDENT"
	INT    TokenType = "INT"
	FLOAT  TokenType = "FLOAT"
	STRING TokenType = "STRING"
	BOOL   TokenType = "BOOL"

	// Operators
	ASSIGN   TokenType = "="
	PLUS     TokenType = "+"
	MINUS    TokenType = "-"
	BANG     TokenType = "!"
	ASTERISK TokenType = "*"
	SLASH    TokenType = "/"
	PERCENT  TokenType = "%"

	// Comparison
	EQ     TokenType = "=="
	NOT_EQ TokenType = "!="
	LT     TokenType = "<"
	GT     TokenType = ">"
	LT_EQ  TokenType = "<="
	GT_EQ  TokenType = ">="

	// Logical
	AND TokenType = "&&"
	OR  TokenType = "||"

	// Delimiters
	COLON     TokenType = ":"
	SEMICOLON TokenType = ";"
	COMMA     TokenType = ","
	DOT       TokenType = "."

	LPAREN   TokenType = "("
	RPAREN   TokenType = ")"
	LBRACE   TokenType = "{"
	RBRACE   TokenType = "}"
	LBRACKET TokenType = "["
	RBRACKET TokenType = "]"

	AT TokenType = "@"

	// Keywords
	FUNC    TokenType = "FUNC"
	LET     TokenType = "LET"
	CONST   TokenType = "CONST"
	TRUE    TokenType = "TRUE"
	FALSE   TokenType = "FALSE"
	IF      TokenType = "IF"
	ELSE    TokenType = "ELSE"
	RETURN  TokenType = "RETURN"
	MODEL   TokenType = "MODEL"
	SERVICE TokenType = "SERVICE"
	IMPORT  TokenType = "IMPORT"
	TASK    TokenType = "TASK"
	AS      TokenType = "AS"
	TRY     TokenType = "TRY"
	RENDER  TokenType = "RENDER"
	CTX     TokenType = "CTX"
	ERROR   TokenType = "ERROR"

	// Raw blocks (captured as-is)
	RAW_GO       TokenType = "RAW_GO"
	RAW_TEMPLATE TokenType = "RAW_TEMPLATE"
	RAW_STYLE    TokenType = "RAW_STYLE"
)

var keywords = map[string]TokenType{
	"func":    FUNC,
	"let":     LET,
	"const":   CONST,
	"true":    TRUE,
	"false":   FALSE,
	"if":      IF,
	"else":    ELSE,
	"return":  RETURN,
	"model":   MODEL,
	"service": SERVICE,
	"import":  IMPORT,
	"task":    TASK,
	"as":      AS,
	"try":     TRY,
	"render":  RENDER,
	"ctx":     CTX,
	"error":   ERROR,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
