package generator

import (
	"fmt"
	"gmx/internal/compiler/ast"
	"gmx/internal/compiler/utils"
	"strings"
)

// genModels generates Go struct definitions from model declarations
func (g *Generator) genModels(models []*ast.ModelDecl) string {
	var b strings.Builder

	for i, model := range models {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("type %s struct {\n", model.Name))

		for _, field := range model.Fields {
			// Convert field name to PascalCase
			fieldName := utils.ToPascalCase(field.Name)
			goType := g.mapType(field.Type)
			jsonTag := field.Name
			gormTags := g.genGormTags(field, model.Name)

			// Build the tag string
			var tags []string
			if gormTags != "" {
				tags = append(tags, fmt.Sprintf("gorm:\"%s\"", gormTags))
			}
			tags = append(tags, fmt.Sprintf("json:\"%s\"", jsonTag))
			tagString := strings.Join(tags, " ")

			b.WriteString(fmt.Sprintf("\t%s %s `%s`\n", fieldName, goType, tagString))
		}

		b.WriteString("}\n")

		// Generate Validate method
		validation := g.genValidation(model)
		if validation != "" {
			b.WriteString("\n")
			b.WriteString(validation)
		}

		// Generate BeforeCreate hook
		beforeCreate := g.genBeforeCreate(model)
		if beforeCreate != "" {
			b.WriteString(beforeCreate)
		}
	}

	return b.String()
}

// genValidation generates a Validate() method for a model
func (g *Generator) genValidation(model *ast.ModelDecl) string {
	var validations []string

	// Scan fields for validation annotations
	for _, field := range model.Fields {
		fieldName := utils.ToPascalCase(field.Name)
		fieldType := field.Type

		for _, ann := range field.Annotations {
			switch ann.Name {
			case "min":
				minVal := ann.SimpleArg()
				if minVal != "" {
					// For string fields, check length
					if fieldType == "string" {
						validations = append(validations, fmt.Sprintf(
							"\tif len(%s.%s) < %s {\n\t\treturn fmt.Errorf(\"%s: minimum length is %s, got %%d\", len(%s.%s))\n\t}",
							utils.ReceiverName(model.Name), fieldName, minVal, field.Name, minVal, utils.ReceiverName(model.Name), fieldName,
						))
					} else if fieldType == "int" || fieldType == "float" {
						// For numeric fields, check value
						validations = append(validations, fmt.Sprintf(
							"\tif %s.%s < %s {\n\t\treturn fmt.Errorf(\"%s: minimum value is %s, got %%v\", %s.%s)\n\t}",
							utils.ReceiverName(model.Name), fieldName, minVal, field.Name, minVal, utils.ReceiverName(model.Name), fieldName,
						))
					}
				}

			case "max":
				maxVal := ann.SimpleArg()
				if maxVal != "" {
					// For string fields, check length
					if fieldType == "string" {
						validations = append(validations, fmt.Sprintf(
							"\tif len(%s.%s) > %s {\n\t\treturn fmt.Errorf(\"%s: maximum length is %s, got %%d\", len(%s.%s))\n\t}",
							utils.ReceiverName(model.Name), fieldName, maxVal, field.Name, maxVal, utils.ReceiverName(model.Name), fieldName,
						))
					} else if fieldType == "int" || fieldType == "float" {
						// For numeric fields, check value
						validations = append(validations, fmt.Sprintf(
							"\tif %s.%s > %s {\n\t\treturn fmt.Errorf(\"%s: maximum value is %s, got %%v\", %s.%s)\n\t}",
							utils.ReceiverName(model.Name), fieldName, maxVal, field.Name, maxVal, utils.ReceiverName(model.Name), fieldName,
						))
					}
				}

			case "email":
				validations = append(validations, fmt.Sprintf(
					"\tif %s.%s != \"\" && !isValidEmail(%s.%s) {\n\t\treturn fmt.Errorf(\"%s: invalid email format\")\n\t}",
					utils.ReceiverName(model.Name), fieldName, utils.ReceiverName(model.Name), fieldName, field.Name,
				))
			}
		}
	}

	// Only generate method if there are validations
	if len(validations) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("// Validate checks all field constraints defined in the model\n"))
	b.WriteString(fmt.Sprintf("func (%s *%s) Validate() error {\n", utils.ReceiverName(model.Name), model.Name))
	for _, validation := range validations {
		b.WriteString(validation + "\n")
	}
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n\n")

	return b.String()
}

// genBeforeCreate generates a GORM BeforeCreate hook for a model
func (g *Generator) genBeforeCreate(model *ast.ModelDecl) string {
	var uuidFields []string

	// Scan fields for @default(uuid_v4)
	for _, field := range model.Fields {
		for _, ann := range field.Annotations {
			if ann.Name == "default" && ann.SimpleArg() == "uuid_v4" {
				uuidFields = append(uuidFields, utils.ToPascalCase(field.Name))
			}
		}
	}

	// Only generate hook if there are uuid_v4 defaults
	if len(uuidFields) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("// BeforeCreate is a GORM hook that sets default values before inserting\n"))
	b.WriteString(fmt.Sprintf("func (%s *%s) BeforeCreate(tx *gorm.DB) error {\n", utils.ReceiverName(model.Name), model.Name))
	for _, fieldName := range uuidFields {
		b.WriteString(fmt.Sprintf("\tif %s.%s == \"\" {\n", utils.ReceiverName(model.Name), fieldName))
		b.WriteString(fmt.Sprintf("\t\t%s.%s = generateUUID()\n", utils.ReceiverName(model.Name), fieldName))
		b.WriteString("\t}\n")
	}
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n\n")

	return b.String()
}

// genGormTags generates GORM tags for a field
func (g *Generator) genGormTags(field *ast.FieldDecl, modelName string) string {
	var tags []string

	for _, ann := range field.Annotations {
		switch ann.Name {
		case "pk":
			tags = append(tags, "primaryKey")
		case "unique":
			tags = append(tags, "unique")
		case "default":
			if val := ann.SimpleArg(); val != "" {
				tags = append(tags, fmt.Sprintf("default:%s", val))
			}
		case "relation":
			// Add foreign key tag
			if ref := ann.Args["references"]; ref != "" {
				// Generate FK field name: for "user: User", FK is "UserID"
				if !strings.HasSuffix(field.Type, "[]") {
					// Single relation: add FK
					fkFieldName := utils.ToPascalCase(field.Name) + "ID"
					tags = append(tags, fmt.Sprintf("foreignKey:%s", fkFieldName))
				}
			}
		}
	}

	return strings.Join(tags, ";")
}

// mapType converts GMX types to Go types
func (g *Generator) mapType(gmxType string) string {
	switch gmxType {
	case "uuid":
		return "string"
	case "string":
		return "string"
	case "int":
		return "int"
	case "float":
		return "float64"
	case "bool":
		return "bool"
	case "datetime":
		return "time.Time"
	default:
		// Check if it's an array type (e.g., "Post[]")
		if strings.HasSuffix(gmxType, "[]") {
			baseType := strings.TrimSuffix(gmxType, "[]")
			return "[]" + baseType
		}
		// Otherwise it's a relation to another model
		return gmxType
	}
}

// genPageData generates the PageData struct
func (g *Generator) genPageData(models []*ast.ModelDecl) string {
	var b strings.Builder

	b.WriteString("type PageData struct {\n")
	// CSRF token is always first for security
	b.WriteString("\tCSRFToken string\n")
	for _, model := range models {
		// Add a slice field for each model
		b.WriteString(fmt.Sprintf("\t%ss []%s\n", model.Name, model.Name))
	}
	b.WriteString("}\n")

	return b.String()
}
