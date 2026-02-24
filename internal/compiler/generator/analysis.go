package generator

import (
	"gmx/internal/compiler/ast"
)

// hasAnnotationMatch scans all fields of all models and returns true
// if at least one annotation satisfies the predicate.
func (g *Generator) hasAnnotationMatch(file *ast.GMXFile, predicate func(*ast.Annotation) bool) bool {
	for _, model := range file.Models {
		for _, field := range model.Fields {
			for _, ann := range field.Annotations {
				if predicate(ann) {
					return true
				}
			}
		}
	}
	return false
}

// hasFieldMatch scans all fields of all models and returns true
// if at least one field satisfies the predicate (for cases that depend on field type + annotations).
func (g *Generator) hasFieldMatch(file *ast.GMXFile, predicate func(*ast.FieldDecl) bool) bool {
	for _, model := range file.Models {
		for _, field := range model.Fields {
			if predicate(field) {
				return true
			}
		}
	}
	return false
}

// hasValidation checks if a model has any validation annotations
func (g *Generator) hasValidation(model *ast.ModelDecl) bool {
	for _, field := range model.Fields {
		for _, ann := range field.Annotations {
			if ann.Name == "min" || ann.Name == "max" || ann.Name == "email" {
				return true
			}
		}
	}
	return false
}

// needsStrconv checks if script functions have int or bool parameters
func (g *Generator) needsStrconv(file *ast.GMXFile) bool {
	if file.Script == nil || file.Script.Funcs == nil {
		return false
	}
	for _, fn := range file.Script.Funcs {
		for _, param := range fn.Params {
			if param.Type == "int" || param.Type == "bool" {
				return true
			}
		}
	}
	return false
}

// needsUUIDValidation checks if UUID validation is needed
func (g *Generator) needsUUIDValidation(file *ast.GMXFile) bool {
	// Check if any model has a uuid @pk field
	hasPKUUID := g.hasFieldMatch(file, func(field *ast.FieldDecl) bool {
		if field.Type != "uuid" {
			return false
		}
		for _, ann := range field.Annotations {
			if ann.Name == "pk" {
				return true
			}
		}
		return false
	})

	if hasPKUUID {
		return true
	}

	// Check if any script parameter is of type uuid
	if file.Script == nil || file.Script.Funcs == nil {
		return false
	}
	for _, fn := range file.Script.Funcs {
		for _, param := range fn.Params {
			if param.Type == "uuid" {
				return true
			}
		}
	}
	return false
}

// scriptFuncNames returns a set of all script function names for quick lookup
func (g *Generator) scriptFuncNames(file *ast.GMXFile) map[string]bool {
	names := make(map[string]bool)
	if file.Script != nil && file.Script.Funcs != nil {
		for _, fn := range file.Script.Funcs {
			names[fn.Name] = true
		}
	}
	return names
}

// extractModelNames extracts model names from the model list
func (g *Generator) extractModelNames(models []*ast.ModelDecl) []string {
	names := make([]string, len(models))
	for i, model := range models {
		names[i] = model.Name
	}
	return names
}

// hasServicesWithEnv checks if any service has fields with @env annotations
func (g *Generator) hasServicesWithEnv(file *ast.GMXFile) bool {
	for _, svc := range file.Services {
		for _, field := range svc.Fields {
			if field.EnvVar != "" {
				return true
			}
		}
	}
	return false
}

// findDatabaseService returns the Database service declaration if one exists
func (g *Generator) findDatabaseService(services []*ast.ServiceDecl) *ast.ServiceDecl {
	for _, svc := range services {
		// Match by name "Database" or by provider type
		if svc.Name == "Database" || svc.Provider == "postgres" || svc.Provider == "sqlite" || svc.Provider == "mysql" {
			return svc
		}
	}
	return nil
}

// hasServiceWithProvider checks if any service uses the specified provider
func (g *Generator) hasServiceWithProvider(file *ast.GMXFile, provider string) bool {
	for _, svc := range file.Services {
		if svc.Provider == provider {
			return true
		}
	}
	return false
}
