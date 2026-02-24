package generator

import (
	"fmt"
	"github.com/btouchard/gmx/internal/compiler/ast"
	"strings"
)

// genVars generates Go variable and constant declarations from GMX let/const
func (g *Generator) genVars(vars []*ast.VarDecl) string {
	var b strings.Builder

	for _, v := range vars {
		if v.IsConst {
			// const NAME = value
			b.WriteString(fmt.Sprintf("const %s = %s\n", v.Name, g.transpileVarValue(v.Value)))
		} else {
			// var name Type = value
			goType := g.transpileVarType(v.Type, v.Value)
			if goType != "" {
				b.WriteString(fmt.Sprintf("var %s %s = %s\n", v.Name, goType, g.transpileVarValue(v.Value)))
			} else {
				// Type inference - let Go infer the type
				b.WriteString(fmt.Sprintf("var %s = %s\n", v.Name, g.transpileVarValue(v.Value)))
			}
		}
	}

	return b.String()
}

// transpileVarType converts GMX type to Go type for variables
func (g *Generator) transpileVarType(gmxType string, value ast.Expression) string {
	if gmxType == "" {
		// No explicit type - try to infer from value
		return g.inferGoType(value)
	}

	// Map GMX types to Go types
	switch gmxType {
	case "int":
		return "int"
	case "float":
		return "float64"
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "uuid":
		return "string"
	default:
		// For unknown types, return empty string to use type inference
		return ""
	}
}

// inferGoType tries to infer the Go type from the expression
func (g *Generator) inferGoType(expr ast.Expression) string {
	switch expr.(type) {
	case *ast.IntLit:
		return "int"
	case *ast.FloatLit:
		return "float64"
	case *ast.StringLit:
		return "string"
	case *ast.BoolLit:
		return "bool"
	default:
		// For complex expressions, let Go infer the type
		return ""
	}
}

// transpileVarValue converts an expression to its Go representation for variable initialization
func (g *Generator) transpileVarValue(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.IntLit:
		return e.Value
	case *ast.FloatLit:
		return e.Value
	case *ast.StringLit:
		return fmt.Sprintf("%q", e.Value)
	case *ast.BoolLit:
		if e.Value {
			return "true"
		}
		return "false"
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", g.transpileVarValue(e.Left), e.Op, g.transpileVarValue(e.Right))
	case *ast.UnaryExpr:
		return fmt.Sprintf("%s%s", e.Op, g.transpileVarValue(e.Operand))
	case *ast.Ident:
		return e.Name
	default:
		// For more complex expressions, return a placeholder
		return fmt.Sprintf("/* unsupported expression: %T */", expr)
	}
}
