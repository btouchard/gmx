package errors

import "fmt"

// Position represents a location in source code
type Position struct {
	File   string
	Line   int
	Column int
}

func (p Position) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// CompileError represents a compilation error with source position
type CompileError struct {
	Pos     Position
	Message string
	Phase   string // "lexer", "parser", "generator"
}

func (e *CompileError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Phase, e.Pos, e.Message)
}

// ErrorList collects multiple compilation errors
type ErrorList struct {
	Errors []*CompileError
}

func NewErrorList() *ErrorList {
	return &ErrorList{}
}

func (el *ErrorList) Add(pos Position, phase, message string) {
	el.Errors = append(el.Errors, &CompileError{Pos: pos, Message: message, Phase: phase})
}

func (el *ErrorList) HasErrors() bool {
	return len(el.Errors) > 0
}

func (el *ErrorList) String() string {
	s := ""
	for _, e := range el.Errors {
		s += e.Error() + "\n"
	}
	return s
}
