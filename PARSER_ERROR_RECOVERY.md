# Parser Error Recovery Implementation

## Summary

Fixed infinite loop bugs in the GMX parser by implementing comprehensive error recovery mechanisms.

## Problem

The parser would enter infinite loops on malformed input because `expectPeek()` doesn't advance the token cursor when it fails. Example:

```
Input: model { }  (name missing)

Iteration 1: curToken = MODEL
  → parseModelDecl() → expectPeek(IDENT) fails → return nil
  → curToken is still MODEL

Iteration 2: curToken = MODEL ← infinite loop
```

## Solution - 3 Levels of Protection

### 1. `synchronize()` - Skip to Next Top-Level Block

Added a method that skips tokens until reaching a boundary that can start a new declaration:

```go
func (p *Parser) synchronize() {
    for !p.curTokenIs(token.EOF) {
        switch p.curToken.Type {
        case token.MODEL, token.SERVICE, token.RAW_GO, token.RAW_TEMPLATE, token.RAW_STYLE:
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

### 2. Progress Guarantee in `ParseGMXFile()`

Track token position before parsing each block. If parsing fails without advancing:

```go
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
```

### 3. Safety Nets in All Internal Loops

Added position checks in every `for` loop to prevent stalls:

- `parseModelDecl` - field parsing loop
- `parseServiceDecl` - body parsing loop
- `parseAnnotationArgs` - argument parsing loop
- `parseAnnotationValue` - array value loop
- `parseServiceMethod` - parameter parsing loop

Example:
```go
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
```

### 4. Return Partial Results on Error

Modified `parseModelDecl()` and `parseServiceDecl()` to return partial structures instead of `nil` when encountering errors. This enables multi-error reporting:

```go
if !p.curTokenIs(token.RBRACE) {
    p.addError("expected '}' at end of model")
    // Still return the partial model for error recovery
    return model  // was: return nil
}
```

## Test Coverage

Added 12 comprehensive error recovery tests:

1. **TestErrorRecovery_ModelMissingName** - `model { field: string }`
2. **TestErrorRecovery_ServiceMissingName** - `service { provider: "x" }`
3. **TestErrorRecovery_ModelMissingBrace** - `model Task field: string }`
4. **TestErrorRecovery_MultipleBlocksWithError** - First block fails, second parses
5. **TestErrorRecovery_ServiceThenModel** - Service fails, model still parses
6. **TestErrorRecovery_FieldMissingType** - `model Task { name: @pk }`
7. **TestErrorRecovery_AnnotationMissingClose** - `@min(3 }` missing `)`
8. **TestErrorRecovery_MultipleErrorsAcrossBlocks** - Multiple failures in one file
9. **TestErrorRecovery_AnnotationMissingBracket** - `@default([a, b }` missing `]`
10. **TestErrorRecovery_ServiceMethodBadParams** - Malformed method parameters
11. **TestErrorRecovery_IncompleteModelEOF** - File ends mid-declaration
12. **TestErrorRecovery_MultipleInvalidModels** - Sequence of malformed blocks

All tests use a 2-second timeout to detect infinite loops.

## Verification

```bash
# All parser tests pass
go test ./internal/compiler/parser/ -count=1 -timeout=30s
# ok  	gmx/internal/compiler/parser	0.002s

# No infinite loops on malformed input
go test ./internal/compiler/parser/ -run TestErrorRecovery
# PASS - all 12 tests

# Clean code
go vet ./internal/...
# (no warnings)
```

## Behavior

### Before
- ❌ Infinite loop on `model { }`
- ❌ Infinite loop on `service { }`
- ❌ Infinite loop on malformed annotations
- ❌ Single error stops all parsing

### After
- ✅ No infinite loops on any malformed input
- ✅ Multi-error reporting (continues after errors)
- ✅ Valid blocks parse correctly despite errors in other blocks
- ✅ Graceful degradation with partial results
- ✅ Meaningful error messages collected in `p.Errors()`

## Design Principles

1. **Never panic, never hang** - Errors are collected, parsing continues
2. **Guarantee forward progress** - Every loop iteration must advance the cursor
3. **Preserve valid work** - Parse what you can, skip what you can't
4. **User-friendly** - Report all errors at once, not just the first one
5. **No regressions** - All existing valid inputs still parse correctly
