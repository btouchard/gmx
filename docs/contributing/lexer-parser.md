# Lexer & Parser

Le lexer et le parser transforment le texte `.gmx` brut en AST utilisable par le generator.

## Lexer

**Fichier** : `internal/compiler/lexer/lexer.go`

### Responsabilité

Transformer le texte source en stream de tokens.

### Token Structure

```go
type Token struct {
    Type    TokenType
    Literal string
    Pos     Position
}

type Position struct {
    Line   int
    Column int
}
```

### Tokens Principaux

| Type | Literal | Usage |
|------|---------|-------|
| `MODEL` | `"model"` | Déclaration model |
| `SERVICE` | `"service"` | Déclaration service |
| `FUNC` | `"func"` | Déclaration fonction |
| `IDENT` | variable | Identifiants |
| `STRING` | `"..."` | Chaînes |
| `INT` | `42` | Nombres entiers |
| `FLOAT` | `3.14` | Nombres décimaux |
| `AT` | `@` | Annotations |
| `COLON` | `:` | Séparateur type |
| `LBRACE` | `{` | Ouverture bloc |
| `RBRACE` | `}` | Fermeture bloc |
| `RAW_GO` | script content | Contenu `<script>` |
| `RAW_TEMPLATE` | template content | Contenu `<template>` |
| `RAW_STYLE` | style content | Contenu `<style>` |

### Traitement des Sections

Le lexer détecte `<script>`, `<template>`, `<style>` et retourne un **token unique** avec tout le contenu :

```go
if strings.HasPrefix(l.input[l.position:], "<script>") {
    content := l.readUntil("</script>")
    return Token{Type: RAW_GO, Literal: content}
}
```

**Avantage** : Le parser principal n'a pas besoin de gérer la syntaxe HTML/CSS.

## Parser

**Fichier** : `internal/compiler/parser/parser.go`

### Responsabilité

Construire l'AST depuis le stream de tokens.

### Structure

```go
type Parser struct {
    l         *lexer.Lexer
    curToken  token.Token
    peekToken token.Token
    errors    []string
}
```

### Entrée : ParseGMXFile

```go
func (p *Parser) ParseGMXFile() *ast.GMXFile {
    file := &ast.GMXFile{
        Models:   []*ast.ModelDecl{},
        Services: []*ast.ServiceDecl{},
    }

    for !p.curTokenIs(token.EOF) {
        switch p.curToken.Type {
        case token.MODEL:
            model := p.parseModelDecl()
            file.Models = append(file.Models, model)
        case token.SERVICE:
            svc := p.parseServiceDecl()
            file.Services = append(file.Services, svc)
        case token.RAW_GO:
            file.Script = p.parseScriptBlock()
        case token.RAW_TEMPLATE:
            file.Template = &ast.TemplateBlock{Source: p.curToken.Literal}
        case token.RAW_STYLE:
            file.Style = p.parseStyleBlock()
        }
        p.nextToken()
    }

    return file
}
```

### Parsing Models

```go
func (p *Parser) parseModelDecl() *ast.ModelDecl {
    // model Task { ... }
    p.expectPeek(token.IDENT)  // "Task"
    modelName := p.curToken.Literal

    p.expectPeek(token.LBRACE)
    p.nextToken()

    fields := []*ast.FieldDecl{}
    for !p.curTokenIs(token.RBRACE) {
        field := p.parseFieldDecl()
        fields = append(fields, field)
    }

    return &ast.ModelDecl{Name: modelName, Fields: fields}
}
```

### Parsing Fields

```go
func (p *Parser) parseFieldDecl() *ast.FieldDecl {
    // title: string @min(3) @max(255)
    fieldName := p.curToken.Literal

    p.expectPeek(token.COLON)
    p.nextToken()
    fieldType := p.curToken.Literal

    annotations := []*ast.Annotation{}
    for p.curTokenIs(token.AT) {
        ann := p.parseAnnotation()
        annotations = append(annotations, ann)
    }

    return &ast.FieldDecl{
        Name:        fieldName,
        Type:        fieldType,
        Annotations: annotations,
    }
}
```

### Parsing Annotations

```go
func (p *Parser) parseAnnotation() *ast.Annotation {
    // @min(3) ou @relation(references: [id])
    p.expectPeek(token.IDENT)
    annName := p.curToken.Literal

    args := make(map[string]string)
    if p.peekTokenIs(token.LPAREN) {
        p.nextToken()
        p.parseAnnotationArgs(args)
    }

    return &ast.Annotation{Name: annName, Args: args}
}
```

### Error Recovery

Le parser utilise `synchronize()` pour continuer après une erreur :

```go
func (p *Parser) synchronize() {
    for !p.curTokenIs(token.EOF) {
        switch p.curToken.Type {
        case token.MODEL, token.SERVICE, token.RAW_GO, token.RAW_TEMPLATE:
            return
        }
        if p.curTokenIs(token.RBRACE) {
            p.nextToken()
            return
        }
        p.nextToken()
    }
}
```

**Avantage** : Affiche toutes les erreurs en une seule passe.

## Script Parser

**Fichier** : `internal/compiler/script/parser.go`

### Responsabilité

Parser le contenu GMX Script (TypeScript-inspired) en AST de fonctions.

### Différence avec le Parser Principal

Le script parser utilise **Pratt parsing** pour gérer les expressions avec précédence :

```go
const (
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
```

### Parsing d'Expressions

```go
func (p *Parser) parseExpression(precedence int) ast.Expression {
    // Prefix parsing
    prefix := p.prefixParseFns[p.curToken.Type]
    if prefix == nil {
        return nil
    }
    leftExp := prefix()

    // Infix parsing (while precedence is lower)
    for precedence < p.peekPrecedence() {
        infix := p.infixParseFns[p.peekToken.Type]
        if infix == nil {
            return leftExp
        }
        p.nextToken()
        leftExp = infix(leftExp)
    }

    return leftExp
}
```

### Prefix Parsers

```go
p.registerPrefix(token.IDENT, p.parseIdentifier)
p.registerPrefix(token.INT, p.parseIntLiteral)
p.registerPrefix(token.STRING, p.parseStringLiteral)
p.registerPrefix(token.TRY, p.parseTryExpression)
p.registerPrefix(token.RENDER, p.parseRenderExpression)
p.registerPrefix(token.ERROR, p.parseErrorExpression)
```

### Infix Parsers

```go
p.registerInfix(token.PLUS, p.parseBinaryExpression)
p.registerInfix(token.MINUS, p.parseBinaryExpression)
p.registerInfix(token.EQ, p.parseBinaryExpression)
p.registerInfix(token.AND, p.parseBinaryExpression)
p.registerInfix(token.DOT, p.parseMemberExpression)
p.registerInfix(token.LPAREN, p.parseCallExpression)
```

## Tests

### Lexer Tests

```go
func TestLexer(t *testing.T) {
    input := `model Task { id: uuid @pk }`

    tests := []struct {
        expectedType    token.TokenType
        expectedLiteral string
    }{
        {token.MODEL, "model"},
        {token.IDENT, "Task"},
        {token.LBRACE, "{"},
        {token.IDENT, "id"},
        {token.COLON, ":"},
        {token.IDENT, "uuid"},
        {token.AT, "@"},
        {token.IDENT, "pk"},
        {token.RBRACE, "}"},
    }

    l := lexer.New(input)
    for _, tt := range tests {
        tok := l.NextToken()
        assert.Equal(t, tt.expectedType, tok.Type)
        assert.Equal(t, tt.expectedLiteral, tok.Literal)
    }
}
```

### Parser Tests

```go
func TestParseModel(t *testing.T) {
    input := `
    model Task {
      id: uuid @pk @default(uuid_v4)
      title: string @min(3) @max(255)
    }
    `

    l := lexer.New(input)
    p := parser.New(l)
    file := p.ParseGMXFile()

    assert.Len(t, file.Models, 1)
    assert.Equal(t, "Task", file.Models[0].Name)
    assert.Len(t, file.Models[0].Fields, 2)
}
```

## Prochaines Étapes

- **[Generator](generator.md)** — Utilisation de l'AST pour générer Go
- **[Script Transpiler](script-transpiler.md)** — Transpilation du script AST
- **[Testing](testing.md)** — Stratégie de test complète
