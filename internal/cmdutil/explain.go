package cmdutil

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/syntax"
)

// topicMatch maps a user-friendly topic name to categories and an optional filter.
type topicMatch struct {
	cats   []syntax.Category
	filter string
}

var topicAliases = map[string]topicMatch{
	// Category direct names
	"app":           {cats: []syntax.Category{syntax.CatApp}},
	"application":   {cats: []syntax.Category{syntax.CatApp}},
	"data":          {cats: []syntax.Category{syntax.CatData}},
	"models":        {cats: []syntax.Category{syntax.CatData}},
	"model":         {cats: []syntax.Category{syntax.CatData}},
	"pages":         {cats: []syntax.Category{syntax.CatPages}},
	"page":          {cats: []syntax.Category{syntax.CatPages}},
	"navigation":    {cats: []syntax.Category{syntax.CatPages}},
	"components":    {cats: []syntax.Category{syntax.CatComponents}},
	"component":     {cats: []syntax.Category{syntax.CatComponents}},
	"events":        {cats: []syntax.Category{syntax.CatEvents}},
	"interactions":  {cats: []syntax.Category{syntax.CatEvents}},
	"styling":       {cats: []syntax.Category{syntax.CatStyling}},
	"style":         {cats: []syntax.Category{syntax.CatStyling}},
	"css":           {cats: []syntax.Category{syntax.CatStyling}},
	"layout":        {cats: []syntax.Category{syntax.CatStyling}},
	"forms":         {cats: []syntax.Category{syntax.CatForms}},
	"form":          {cats: []syntax.Category{syntax.CatForms}},
	"inputs":        {cats: []syntax.Category{syntax.CatForms}},
	"input":         {cats: []syntax.Category{syntax.CatForms}},
	"apis":          {cats: []syntax.Category{syntax.CatAPIs}},
	"api":           {cats: []syntax.Category{syntax.CatAPIs}},
	"endpoints":     {cats: []syntax.Category{syntax.CatAPIs}},
	"security":      {cats: []syntax.Category{syntax.CatSecurity}},
	"auth":          {cats: []syntax.Category{syntax.CatSecurity}},
	"authentication": {cats: []syntax.Category{syntax.CatSecurity}},
	"policies":      {cats: []syntax.Category{syntax.CatPolicies}},
	"policy":        {cats: []syntax.Category{syntax.CatPolicies}},
	"authorization": {cats: []syntax.Category{syntax.CatPolicies}},
	"database":      {cats: []syntax.Category{syntax.CatDatabase}},
	"db":            {cats: []syntax.Category{syntax.CatDatabase}},
	"workflows":     {cats: []syntax.Category{syntax.CatWorkflows}},
	"workflow":      {cats: []syntax.Category{syntax.CatWorkflows}},
	"integrations":  {cats: []syntax.Category{syntax.CatIntegrations}},
	"integration":   {cats: []syntax.Category{syntax.CatIntegrations}},
	"architecture":  {cats: []syntax.Category{syntax.CatArchitecture}},
	"devops":        {cats: []syntax.Category{syntax.CatDevOps}},
	"deploy":        {cats: []syntax.Category{syntax.CatDevOps}},
	"deployment":    {cats: []syntax.Category{syntax.CatDevOps}},
	"cicd":          {cats: []syntax.Category{syntax.CatDevOps}},
	"theme":         {cats: []syntax.Category{syntax.CatTheme}},
	"themes":        {cats: []syntax.Category{syntax.CatTheme}},
	"design":        {cats: []syntax.Category{syntax.CatTheme}},
	"build":         {cats: []syntax.Category{syntax.CatBuild}},
	"targets":       {cats: []syntax.Category{syntax.CatBuild}},
	"conditional":   {cats: []syntax.Category{syntax.CatConditional}},
	"conditions":    {cats: []syntax.Category{syntax.CatConditional}},
	"errors":        {cats: []syntax.Category{syntax.CatErrors}},
	"error":         {cats: []syntax.Category{syntax.CatErrors}},

	// Cross-category aliases
	"button":  {cats: []syntax.Category{syntax.CatEvents, syntax.CatForms}, filter: "button"},
	"buttons": {cats: []syntax.Category{syntax.CatEvents, syntax.CatForms}, filter: "button"},
	"click":   {cats: []syntax.Category{syntax.CatEvents}, filter: "click"},
	"hover":   {cats: []syntax.Category{syntax.CatEvents}, filter: "hover"},
	"color":   {cats: []syntax.Category{syntax.CatStyling, syntax.CatTheme}, filter: "color"},
	"colors":  {cats: []syntax.Category{syntax.CatStyling, syntax.CatTheme}, filter: "color"},
	"entity":  {cats: []syntax.Category{syntax.CatData}},
	"route":   {cats: []syntax.Category{syntax.CatAPIs}},
	"routes":  {cats: []syntax.Category{syntax.CatAPIs}},
}

// ExplainTopicNames returns all valid topic names for tab completion.
func ExplainTopicNames() []string {
	names := make([]string, 0, len(topicAliases))
	for name := range topicAliases {
		names = append(names, name)
	}
	return names
}

var placeholderRe = regexp.MustCompile(`<([^>]+)>`)

// RunExplain displays an explanation for the given topic.
// If topic is empty, lists all available topics.
func RunExplain(out io.Writer, topic string) {
	if topic == "" {
		printTopicList(out)
		return
	}

	topic = strings.ToLower(topic)

	match, ok := topicAliases[topic]
	if !ok {
		// Try searching.
		results := syntax.Search(topic)
		if len(results) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintf(out, "No exact topic %q — showing search results:\n\n", topic)
			printSearchResults(out, results, 10)
			return
		}
		fmt.Fprintf(out, "Unknown topic: %s\n", topic)
		fmt.Fprintln(out, "Run 'human explain' to see available topics.")
		return
	}

	fmt.Fprintln(out)

	for _, cat := range match.cats {
		patterns := syntax.ByCategory(cat)
		if match.filter != "" {
			patterns = filterPatterns(patterns, match.filter)
		}
		if len(patterns) == 0 {
			continue
		}

		printCategorySection(out, cat, patterns)
	}

	// Related topics footer.
	related := relatedTopics(match.cats)
	if len(related) > 0 {
		parts := make([]string, len(related))
		for i, r := range related {
			parts[i] = fmt.Sprintf("human explain %s", r)
		}
		fmt.Fprintf(out, "%s\n\n", cli.Muted("Related: "+strings.Join(parts, " │ ")))
	}
}

func printTopicList(out io.Writer) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, cli.Heading("Available Topics"))
	fmt.Fprintln(out, strings.Repeat("─", 50))
	fmt.Fprintln(out)

	for _, cat := range syntax.AllCategories() {
		count := len(syntax.ByCategory(cat))
		fmt.Fprintf(out, "  %-20s %s\n",
			cli.Accent(string(cat)),
			cli.Muted(fmt.Sprintf("(%d patterns) %s", count, syntax.CategoryLabel(cat))))
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, cli.Muted("Usage: human explain <topic>"))
	fmt.Fprintln(out, cli.Muted("  e.g.: human explain data, human explain apis, human explain color"))
	fmt.Fprintln(out)
}

func printCategorySection(out io.Writer, cat syntax.Category, patterns []syntax.Pattern) {
	header := fmt.Sprintf("── %s in Human ", syntax.CategoryLabel(cat))
	pad := 50 - len([]rune(header))
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(out, "%s%s\n\n", cli.Heading(header), strings.Repeat("─", pad))

	for _, p := range patterns {
		template := highlightPlaceholders(p.Template)
		fmt.Fprintf(out, "  %-42s %s\n", template, cli.Muted(p.Description))
		if p.Example != "" {
			fmt.Fprintf(out, "    %s\n", cli.Muted("e.g.: "+p.Example))
		}
	}
	fmt.Fprintln(out)
}

func printSearchResults(out io.Writer, patterns []syntax.Pattern, max int) {
	if len(patterns) > max {
		patterns = patterns[:max]
	}
	for _, p := range patterns {
		template := highlightPlaceholders(p.Template)
		fmt.Fprintf(out, "  %-42s %s\n", template, cli.Muted(fmt.Sprintf("(%s)", p.Category)))
	}
	fmt.Fprintln(out)
}

func highlightPlaceholders(template string) string {
	if !cli.ColorEnabled {
		return template
	}
	infoColor := cli.Colorize(cli.RoleInfo, "")
	// Colorize just the placeholder parts.
	return placeholderRe.ReplaceAllStringFunc(template, func(match string) string {
		_ = infoColor
		return cli.Colorize(cli.RoleInfo, match)
	})
}

func filterPatterns(patterns []syntax.Pattern, term string) []syntax.Pattern {
	term = strings.ToLower(term)
	var result []syntax.Pattern
	for _, p := range patterns {
		if strings.Contains(strings.ToLower(p.Template), term) {
			result = append(result, p)
			continue
		}
		for _, tag := range p.Tags {
			if strings.Contains(strings.ToLower(tag), term) {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

func relatedTopics(current []syntax.Category) []string {
	// Map categories to related topic names.
	relatedMap := map[syntax.Category][]string{
		syntax.CatData:         {"apis", "database", "pages"},
		syntax.CatPages:        {"events", "forms", "styling"},
		syntax.CatEvents:       {"pages", "forms"},
		syntax.CatStyling:      {"theme", "pages"},
		syntax.CatForms:        {"events", "pages", "apis"},
		syntax.CatAPIs:         {"data", "security", "database"},
		syntax.CatSecurity:     {"apis", "policies"},
		syntax.CatPolicies:     {"security"},
		syntax.CatDatabase:     {"data", "apis"},
		syntax.CatWorkflows:    {"integrations", "apis"},
		syntax.CatIntegrations: {"workflows", "apis"},
		syntax.CatArchitecture: {"devops", "build"},
		syntax.CatDevOps:       {"architecture", "build"},
		syntax.CatTheme:        {"styling"},
		syntax.CatBuild:        {"architecture", "devops"},
		syntax.CatConditional:  {"pages", "errors"},
		syntax.CatErrors:       {"conditional", "apis"},
		syntax.CatComponents:   {"pages", "styling"},
		syntax.CatApp:          {"data", "build"},
	}

	currentSet := make(map[syntax.Category]bool)
	for _, c := range current {
		currentSet[c] = true
	}

	seen := make(map[string]bool)
	var result []string
	for _, cat := range current {
		for _, r := range relatedMap[cat] {
			if !seen[r] {
				seen[r] = true
				result = append(result, r)
			}
		}
	}

	// Cap at 3.
	if len(result) > 3 {
		result = result[:3]
	}
	return result
}
