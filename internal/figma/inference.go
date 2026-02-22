package figma

import (
	"strings"
)

// InferModels extracts data models from classified pages by analyzing forms,
// cards, and tables. Models inferred from multiple sources are merged.
func InferModels(pages []*ClassifiedPage) []*InferredModel {
	modelMap := make(map[string]*InferredModel)

	for _, page := range pages {
		for _, node := range page.Nodes {
			inferFromNode(node, modelMap)
		}
	}

	// Convert map to slice
	var models []*InferredModel
	for _, m := range modelMap {
		models = append(models, m)
	}
	return models
}

// inferFromNode recursively walks a classified tree to extract models.
func inferFromNode(node *ClassifiedNode, models map[string]*InferredModel) {
	if node == nil {
		return
	}

	switch node.Type {
	case ComponentForm:
		if m := inferFromForm(node); m != nil {
			mergeModel(models, m)
		}
	case ComponentCard:
		if m := inferFromCard(node); m != nil {
			mergeModel(models, m)
		}
	case ComponentTable:
		if m := inferFromTable(node); m != nil {
			mergeModel(models, m)
		}
	}

	for _, child := range node.Children {
		inferFromNode(child, models)
	}
}

// inferFromForm extracts a model from a form's input fields.
// A "Create Task" form produces a "Task" model with fields from each input.
func inferFromForm(node *ClassifiedNode) *InferredModel {
	name := guessModelName(node)
	if name == "" {
		return nil
	}

	model := &InferredModel{
		Name:   name,
		Source: "form",
	}

	for _, child := range node.Children {
		if child.Type == ComponentInput || isInputLike(child) {
			fieldName := extractFieldName(child)
			if fieldName != "" {
				model.Fields = append(model.Fields, &InferredField{
					Name: fieldName,
					Type: guessFieldType(fieldName),
				})
			}
		}
	}

	if len(model.Fields) == 0 {
		return nil
	}
	return model
}

// inferFromCard extracts a model from card content (headings, text, images).
func inferFromCard(node *ClassifiedNode) *InferredModel {
	name := guessModelName(node)
	if name == "" {
		return nil
	}

	model := &InferredModel{
		Name:   name,
		Source: "card",
	}

	for _, child := range node.Children {
		fieldName := ""
		fieldType := "text"

		switch child.Type {
		case ComponentHeading:
			fieldName = "title"
		case ComponentText:
			text := strings.ToLower(child.Text)
			if strings.Contains(text, "description") || len(text) > 50 {
				fieldName = "description"
			} else if strings.Contains(text, "date") || strings.Contains(text, "time") {
				fieldName = "date"
				fieldType = "date"
			} else if fieldName == "" && child.Text != "" {
				fieldName = "description"
			}
		case ComponentImage:
			fieldName = "image"
			fieldType = "image"
		case ComponentBadge:
			fieldName = "status"
		case ComponentAvatar:
			fieldName = "avatar"
			fieldType = "image"
		}

		if fieldName != "" {
			model.Fields = append(model.Fields, &InferredField{
				Name: fieldName,
				Type: fieldType,
			})
		}
	}

	if len(model.Fields) == 0 {
		return nil
	}
	return model
}

// inferFromTable extracts a model from table column headers.
func inferFromTable(node *ClassifiedNode) *InferredModel {
	name := guessModelName(node)
	if name == "" {
		return nil
	}

	model := &InferredModel{
		Name:   name,
		Source: "table",
	}

	// Look for the first row (header row) to extract column names
	if len(node.Children) > 0 {
		headerRow := node.Children[0]
		for _, cell := range headerRow.Children {
			text := strings.TrimSpace(cell.Text)
			if text != "" {
				model.Fields = append(model.Fields, &InferredField{
					Name: strings.ToLower(text),
					Type: guessFieldType(text),
				})
			}
		}
	}

	if len(model.Fields) == 0 {
		return nil
	}
	return model
}

// guessFieldType maps a field label to a Human language type based on common patterns.
func guessFieldType(label string) string {
	lower := strings.ToLower(label)

	typePatterns := []struct {
		keywords []string
		typ      string
	}{
		{[]string{"email", "e-mail"}, "email"},
		{[]string{"url", "link", "website", "homepage"}, "url"},
		{[]string{"price", "cost", "amount", "salary", "total", "balance"}, "decimal"},
		{[]string{"count", "quantity", "age", "number", "rating", "score"}, "number"},
		{[]string{"date", "birthday", "dob"}, "date"},
		{[]string{"time", "created", "updated", "timestamp"}, "datetime"},
		{[]string{"active", "enabled", "visible", "published", "completed", "done", "verified"}, "boolean"},
		{[]string{"avatar", "photo", "picture", "image", "thumbnail", "logo"}, "image"},
		{[]string{"file", "attachment", "document", "upload"}, "file"},
		{[]string{"password", "secret"}, "text"},
	}

	for _, p := range typePatterns {
		for _, keyword := range p.keywords {
			if strings.Contains(lower, keyword) {
				return p.typ
			}
		}
	}
	return "text"
}

// guessModelName extracts a model name from a node's name or text content.
// "Create Task" → "Task", "Products List" → "Product", "User Card" → "User".
func guessModelName(node *ClassifiedNode) string {
	name := node.Node.Name
	text := node.Text

	// Try the node name first
	if modelName := extractModelFromLabel(name); modelName != "" {
		return modelName
	}
	// Fall back to text content
	if modelName := extractModelFromLabel(text); modelName != "" {
		return modelName
	}
	return ""
}

// extractModelFromLabel pulls a model name from strings like "Create Task",
// "Edit User", "Products", "Task Card", "User Form".
func extractModelFromLabel(label string) string {
	if label == "" {
		return ""
	}

	// Remove common UI prefixes/suffixes
	lower := strings.ToLower(label)
	strip := []string{
		"create ", "edit ", "update ", "delete ", "new ", "add ",
		" form", " card", " list", " table", " modal", " dialog",
		" page", " view", " panel", " section", " detail", " details",
	}
	cleaned := lower
	for _, s := range strip {
		cleaned = strings.TrimPrefix(cleaned, s)
		cleaned = strings.TrimSuffix(cleaned, s)
	}
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		return ""
	}

	// Singularize and PascalCase
	return toPascalCase(singularize(cleaned))
}

// isInputLike checks if a node looks like a form input even if not classified as one.
func isInputLike(node *ClassifiedNode) bool {
	if node.Node == nil {
		return false
	}
	// Frames with strokes (border) that are wider than tall look like inputs
	n := node.Node
	if n.Type == "FRAME" && len(n.Strokes) > 0 && n.Width > n.Height {
		return true
	}
	return false
}

// extractFieldName gets a field name from an input node's label or placeholder text.
func extractFieldName(node *ClassifiedNode) string {
	// Check node name
	name := strings.ToLower(node.Node.Name)
	strip := []string{"input", "field", "text", "area"}
	cleaned := name
	for _, s := range strip {
		cleaned = strings.TrimSuffix(strings.TrimSpace(cleaned), s)
		cleaned = strings.TrimPrefix(strings.TrimSpace(cleaned), s)
	}
	cleaned = strings.TrimSpace(cleaned)
	if cleaned != "" {
		return strings.ReplaceAll(cleaned, " ", "_")
	}

	// Check text content (placeholder or label)
	if node.Text != "" {
		return strings.ReplaceAll(strings.ToLower(node.Text), " ", "_")
	}

	return ""
}

// mergeModel adds or merges a model into the map, combining fields from
// multiple sources (e.g., a form and a card for the same model).
func mergeModel(models map[string]*InferredModel, newModel *InferredModel) {
	existing, ok := models[newModel.Name]
	if !ok {
		models[newModel.Name] = newModel
		return
	}

	// Merge fields, avoiding duplicates
	existingFields := make(map[string]bool)
	for _, f := range existing.Fields {
		existingFields[f.Name] = true
	}
	for _, f := range newModel.Fields {
		if !existingFields[f.Name] {
			existing.Fields = append(existing.Fields, f)
			existingFields[f.Name] = true
		}
	}
	existing.Source += "+" + newModel.Source
}
