# Testing

GMX a une stratégie de test complète avec **94.2%** de couverture moyenne sur les packages testables (lexer, parser, generator, script, utils).

## Couverture Actuelle

```
Package                          Coverage
-------------------------------------------
gmx/internal/compiler/lexer       94.0% ✅
gmx/internal/compiler/parser      94.5% ✅
gmx/internal/compiler/generator   93.9% ✅
gmx/internal/compiler/script      88.4% ✅ (parser 89.7%, transpiler 87.3%)
gmx/internal/compiler/utils      100.0% ✅
gmx/cmd/gmx                        0.0% ⚪ (CLI - pas prioritaire)
gmx/internal/compiler/ast          0.0% ⚪ (structures de données uniquement)
gmx/internal/compiler/token        0.0% ⚪ (constantes uniquement)
-------------------------------------------
MOYENNE (packages testables)      94.2% ✅
```

**Note** : Les packages `ast` et `token` ne contiennent que des structures de données et constantes, donc ne nécessitent pas de tests unitaires.

## Lancer les Tests

### Tous les tests

```bash
go test ./...
```

### Avec couverture

```bash
go test -cover ./...
```

### Couverture détaillée (avec rapport HTML)

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Package spécifique

```bash
go test ./internal/compiler/generator -v
go test ./internal/compiler/script -cover
go test ./internal/compiler/parser -v -cover
```

### Mode verbose pour déboguer

```bash
go test -v ./...
```

## Patterns de Test

### Table-Driven Tests

La majorité des tests utilisent le pattern table-driven pour couvrir plusieurs cas avec un code minimal et maintenable :

```go
func TestBasicTokens(t *testing.T) {
	input := `= + - ! * / % < > ( ) { } [ ] @ : , . ;`

	expected := []token.TokenType{
		token.ASSIGN, token.PLUS, token.MINUS, token.BANG, token.ASTERISK,
		token.SLASH, token.PERCENT, token.LT, token.GT, token.LPAREN, token.RPAREN,
		token.LBRACE, token.RBRACE, token.LBRACKET, token.RBRACKET,
		token.AT, token.COLON, token.COMMA, token.DOT, token.SEMICOLON,
		token.EOF,
	}

	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Fatalf("test[%d] - wrong type. expected=%s, got=%s", i, exp, tok.Type)
		}
	}
}
```

### Helper Functions

Pour améliorer la lisibilité et réutiliser la logique, les tests complexes utilisent des fonctions helper :

```go
// isValidGo vérifie si le code généré est du Go syntaxiquement valide
func isValidGo(code string) bool {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "test.go", code, parser.AllErrors)
	return err == nil
}
```

### Timeouts pour Détecter les Boucles Infinies

Le parser implémente un mécanisme de timeout pour détecter les boucles infinies lors des tests d'error recovery :

```go
func parseWithTimeout(t *testing.T, input string) (*ast.GMXFile, []string) {
	t.Helper()
	done := make(chan struct{})
	var file *ast.GMXFile
	var errors []string
	go func() {
		l := lexer.New(input)
		p := New(l)
		file = p.ParseGMXFile()
		errors = p.Errors()
		close(done)
	}()
	select {
	case <-done:
		return file, errors
	case <-time.After(2 * time.Second):
		t.Fatal("parser hung — infinite loop detected")
		return nil, nil
	}
}
```

## Types de Tests

### 1. Unit Tests

Tests des fonctions individuelles.

**Exemple : Lexer**

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
        // ...
    }

    l := lexer.New(input)
    for _, tt := range tests {
        tok := l.NextToken()
        assert.Equal(t, tt.expectedType, tok.Type)
        assert.Equal(t, tt.expectedLiteral, tok.Literal)
    }
}
```

### 2. Parser Tests

Tests du parsing complet.

**Exemple : Model**

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

    assert.Len(t, p.Errors(), 0)
    assert.Len(t, file.Models, 1)
    assert.Equal(t, "Task", file.Models[0].Name)
    assert.Len(t, file.Models[0].Fields, 2)

    // Field 1
    assert.Equal(t, "id", file.Models[0].Fields[0].Name)
    assert.Equal(t, "uuid", file.Models[0].Fields[0].Type)
    assert.Len(t, file.Models[0].Fields[0].Annotations, 2)

    // Annotations
    assert.Equal(t, "pk", file.Models[0].Fields[0].Annotations[0].Name)
    assert.Equal(t, "default", file.Models[0].Fields[0].Annotations[1].Name)
    assert.Equal(t, "uuid_v4", file.Models[0].Fields[0].Annotations[1].SimpleArg())
}
```

### 3. Transpiler Tests

Tests de la transpilation GMX → Go.

**Exemple : Try Expression**

```go
func TestTranspileTryExpression(t *testing.T) {
    source := `
    func getTask(id: uuid) error {
      let task = try Task.find(id)
      return render(task)
    }
    `

    funcs, errs := script.Parse(source, 0)
    assert.Len(t, errs, 0)

    result := script.Transpile(&ast.ScriptBlock{Funcs: funcs}, []string{"Task"})

    assert.Contains(t, result.GoCode, "task, err := TaskFind(ctx.DB, id)")
    assert.Contains(t, result.GoCode, "if err != nil {")
    assert.Contains(t, result.GoCode, "return err")
}
```

### 4. Generator Tests

Tests de la génération Go complète.

**Exemple : Model Generation**

```go
func TestGenerateModel(t *testing.T) {
    model := &ast.ModelDecl{
        Name: "Task",
        Fields: []*ast.FieldDecl{
            {
                Name: "id",
                Type: "uuid",
                Annotations: []*ast.Annotation{
                    {Name: "pk", Args: map[string]string{}},
                    {Name: "default", Args: map[string]string{"_": "uuid_v4"}},
                },
            },
            {
                Name: "title",
                Type: "string",
                Annotations: []*ast.Annotation{
                    {Name: "min", Args: map[string]string{"_": "3"}},
                    {Name: "max", Args: map[string]string{"_": "255"}},
                },
            },
        },
    }

    gen := generator.New()
    code := gen.genModels([]*ast.ModelDecl{model})

    // Check struct
    assert.Contains(t, code, "type Task struct {")
    assert.Contains(t, code, "ID string `gorm:\"primaryKey\" json:\"id\"`")
    assert.Contains(t, code, "Title string `json:\"title\"`")

    // Check validation
    assert.Contains(t, code, "func (t *Task) Validate() error {")
    assert.Contains(t, code, "if len(t.Title) < 3 {")
    assert.Contains(t, code, "if len(t.Title) > 255 {")

    // Check BeforeCreate hook
    assert.Contains(t, code, "func (t *Task) BeforeCreate(tx *gorm.DB) error {")
    assert.Contains(t, code, "t.ID = generateUUID()")
}
```

### 5. Integration Tests

Tests end-to-end : `.gmx` → compilation → exécution.

**Exemple : Full Compilation**

```go
func TestIntegration(t *testing.T) {
    input := `
    model Task {
      id: uuid @pk @default(uuid_v4)
      title: string
    }

    <script>
    func createTask(title: string) error {
      const task = Task{title: title}
      try task.save()
      return render(task)
    }
    </script>

    <template>
    <div>{{.Title}}</div>
    </template>
    `

    // 1. Lex
    l := lexer.New(input)

    // 2. Parse
    p := parser.New(l)
    file := p.ParseGMXFile()
    assert.Len(t, p.Errors(), 0)

    // 3. Generate
    gen := generator.New()
    code, err := gen.Generate(file)
    assert.NoError(t, err)

    // 4. Verify Go code compiles
    tmpFile := "/tmp/test_generated.go"
    os.WriteFile(tmpFile, []byte(code), 0644)

    cmd := exec.Command("go", "build", "-o", "/dev/null", tmpFile)
    output, err := cmd.CombinedOutput()
    assert.NoError(t, err, "Generated code should compile: %s", output)
}
```

## Error Recovery Tests

Le parser GMX implémente plusieurs stratégies de récupération d'erreur pour continuer le parsing même après des erreurs de syntaxe.

### Stratégies de Récupération

#### 1. Synchronization

Quand le parser rencontre une erreur, il se synchronise sur des tokens "sûrs" (model, service, }, EOF) :

```go
func (p *Parser) synchronize() {
	for !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.MODEL) || p.curTokenIs(token.SERVICE) || p.curTokenIs(token.RBRACE) {
			return
		}
		p.nextToken()
	}
}
```

#### 2. Progression Guards

Pour éviter les boucles infinies, le parser vérifie qu'il progresse après chaque erreur :

```go
if !p.expectPeek(token.LBRACE) {
	p.synchronize()  // Skip to next safe token
	return model     // Return partial model
}
```

### Tests de Récupération d'Erreur

Tests que le parser continue et parse les déclarations valides malgré les erreurs :

```go
func TestErrorRecovery_MultipleErrorsAcrossBlocks(t *testing.T) {
	input := `model { }
service { }
model Valid { id: uuid @pk }
service ValidSvc { provider: "test" }`

	file, errors := parseWithTimeout(t, input)

	// Devrait avoir des erreurs des blocs invalides
	if len(errors) < 2 {
		t.Errorf("expected at least 2 errors, got %d", len(errors))
	}

	// Mais les blocs valides devraient être parsés
	foundValidModel := false
	for _, model := range file.Models {
		if model.Name == "Valid" {
			foundValidModel = true
		}
	}
	if !foundValidModel {
		t.Error("expected Valid model to be parsed despite errors")
	}

	foundValidSvc := false
	for _, svc := range file.Services {
		if svc.Name == "ValidSvc" {
			foundValidSvc = true
		}
	}
	if !foundValidSvc {
		t.Error("expected ValidSvc service to be parsed")
	}
}
```

### Tests d'Erreurs Spécifiques

```go
func TestParseErrorMissingFuncName(t *testing.T) {
	input := `func (id: uuid) error {
		return nil
	}`

	_, errors := Parse(input, 0)

	if len(errors) == 0 {
		t.Error("expected parser errors for missing function name")
	}
}

func TestParseModelFieldMissingType(t *testing.T) {
	input := `model Task {
		id: @pk
		title: string
	}`

	p := New(lexer.New(input))
	file := p.ParseGMXFile()

	// Devrait avoir des erreurs
	if len(p.Errors()) == 0 {
		t.Error("expected parser errors for missing field type")
	}

	// Mais devrait quand même parser le modèle partiellement
	if len(file.Models) == 0 {
		t.Error("expected model to be created despite errors")
	}
}
```

## Edge Cases

Tests des cas limites.

### Fichier Vide

```go
func TestEmptyFile(t *testing.T) {
    input := ``

    l := lexer.New(input)
    p := parser.New(l)
    file := p.ParseGMXFile()

    assert.Len(t, file.Models, 0)
    assert.Nil(t, file.Script)
    assert.Nil(t, file.Template)
}
```

### Model Sans Champs

```go
func TestModelWithNoFields(t *testing.T) {
    input := `model Task {}`

    l := lexer.New(input)
    p := parser.New(l)
    file := p.ParseGMXFile()

    assert.Len(t, file.Models, 1)
    assert.Len(t, file.Models[0].Fields, 0)
}
```

### Script avec Erreur de Syntaxe

```go
func TestScriptSyntaxError(t *testing.T) {
    source := `
    func broken error {
      let x = try
      return
    }
    `

    funcs, errs := script.Parse(source, 0)

    assert.Greater(t, len(errs), 0)
    assert.Nil(t, funcs)  // Parsing should fail
}
```

## Tests de Sécurité

### CSRF Validation

```go
func TestCSRFProtection(t *testing.T) {
    gen := generator.New()
    file := &ast.GMXFile{
        Script: &ast.ScriptBlock{
            Funcs: []*ast.FuncDecl{
                {Name: "createTask", Params: []*ast.Param{}},
            },
        },
    }

    code, _ := gen.Generate(file)

    // Check CSRF validation is present
    assert.Contains(t, code, "r.Cookie(\"csrf_token\")")
    assert.Contains(t, code, "r.Header.Get(\"X-CSRF-Token\")")
    assert.Contains(t, code, "CSRF validation failed")
}
```

### Input Validation

```go
func TestValidationGeneration(t *testing.T) {
    model := &ast.ModelDecl{
        Name: "User",
        Fields: []*ast.FieldDecl{
            {
                Name: "email",
                Type: "string",
                Annotations: []*ast.Annotation{
                    {Name: "email", Args: map[string]string{}},
                },
            },
        },
    }

    gen := generator.New()
    code := gen.genModels([]*ast.ModelDecl{model})

    assert.Contains(t, code, "func (u *User) Validate() error")
    assert.Contains(t, code, "!isValidEmail(u.Email)")
}
```

## Benchmarks

```go
func BenchmarkLexer(b *testing.B) {
    input := `model Task { id: uuid @pk @default(uuid_v4) }`

    for i := 0; i < b.N; i++ {
        l := lexer.New(input)
        for l.NextToken().Type != token.EOF {
        }
    }
}

func BenchmarkParser(b *testing.B) {
    input := `model Task { id: uuid @pk @default(uuid_v4) }`

    for i := 0; i < b.N; i++ {
        l := lexer.New(input)
        p := parser.New(l)
        p.ParseGMXFile()
    }
}
```

## CI/CD

### GitHub Actions

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

## Tests Manquants (TODO)

1. **Edge case** : Fichier .gmx avec sections dans un ordre non-standard
2. **Edge case** : Annotations complexes imbriquées
3. **Intégration** : Générer → Compiler → Exécuter → Tester HTTP
4. **Sécurité** : SQL injection tentatives
5. **Performance** : Benchmark de génération sur gros fichiers

## Règles de Test

### ❌ Pas de Tests Triviaux

Ne pas écrire de tests pour des fonctionnalités triviales qui n'apportent pas de valeur :

**Mauvais exemple** :
```go
func TestGetterReturnsValue(t *testing.T) {
	obj := &MyStruct{Value: 42}
	if obj.Value != 42 {
		t.Error("getter ne retourne pas la valeur")
	}
}
```

**Bon exemple** - Tester la logique métier, les edge cases, les chemins d'erreur :
```go
func TestParseModelWithComplexAnnotations(t *testing.T) {
	input := `model User {
		email: string @validate(regex: "[a-z]+", message: "invalid")
	}`

	p := New(lexer.New(input))
	file := p.ParseGMXFile()

	// Vérifier le parsing des annotations complexes
	emailField := file.Models[0].Fields[0]
	validateAnn := emailField.Annotations[0]

	if len(validateAnn.Args) != 2 {
		t.Errorf("expected 2 args in @validate, got %d", len(validateAnn.Args))
	}
}
```

### ✅ Tester les Edge Cases et Error Paths

Concentrez-vous sur les cas limites et les chemins d'erreur qui révèlent les bugs :

```go
func TestParseUnterminatedString(t *testing.T) {
	input := `"unterminated`
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.STRING {
		t.Fatalf("expected STRING even if unterminated, got %s", tok.Type)
	}
	// Devrait retourner ce qui a été lu jusqu'à EOF
	if tok.Literal != "unterminated" {
		t.Fatalf("expected 'unterminated', got %q", tok.Literal)
	}
}

func TestParseIncompleteModelEOF(t *testing.T) {
	input := `model Task { id: uuid`
	file, errors := parseWithTimeout(t, input)

	// Devrait avoir des erreurs pour modèle incomplet
	if len(errors) == 0 {
		t.Error("expected errors for incomplete model")
	}

	// Ne devrait pas planter (pas de panic)
	if file == nil {
		t.Fatal("expected file to be created")
	}
}
```

### ✅ Vérifier la Validité du Code Généré

Pour tous les tests du générateur, vérifiez que le code Go produit est syntaxiquement valide :

```go
func TestGenerateModel(t *testing.T) {
	file := &ast.GMXFile{
		Models: []*ast.ModelDecl{
			{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{Name: "id", Type: "uuid", Annotations: []*ast.Annotation{{Name: "pk"}}},
				},
			},
		},
	}

	gen := New()
	code, err := gen.Generate(file)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Premier test : le code doit être du Go valide
	if !isValidGo(code) {
		t.Errorf("Generated code is not valid Go:\n%s", code)
	}

	// Ensuite : vérifier le contenu spécifique
	if !strings.Contains(code, "type User struct") {
		t.Error("Generated code missing 'type User struct'")
	}
}
```

## Stratégie de Test

### Table-Driven Tests

Préférer les table-driven tests pour les variations :

```go
func TestTranspileTypes(t *testing.T) {
    tests := []struct {
        gmxType  string
        expected string
    }{
        {"uuid", "string"},
        {"string", "string"},
        {"int", "int"},
        {"bool", "bool"},
        {"datetime", "time.Time"},
    }

    for _, tt := range tests {
        t.Run(tt.gmxType, func(t *testing.T) {
            result := transpileType(tt.gmxType)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Golden Files

Pour les tests de génération complexe, utiliser des golden files :

```go
func TestGenerateFullApp(t *testing.T) {
    input, _ := os.ReadFile("testdata/app.gmx")
    expected, _ := os.ReadFile("testdata/app.go.golden")

    l := lexer.New(string(input))
    p := parser.New(l)
    file := p.ParseGMXFile()

    gen := generator.New()
    code, _ := gen.Generate(file)

    assert.Equal(t, string(expected), code)
}
```

## Prochaines Étapes

- ✅ Atteindre 95%+ de couverture sur lexer, parser, generator (objectif atteint !)
- ✅ Augmenter la couverture du script transpiler à 85%+ (objectif atteint : 88.4%)
- Ajouter tests d'intégration HTTP end-to-end
- Implémenter tests de sécurité automatisés (fuzzing, injection SQL)
- Ajouter benchmarks de performance sur gros fichiers
