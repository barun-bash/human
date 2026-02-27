package storybook

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

func generateComponentStory(comp *ComponentMeta, app *ir.Application, fw string) string {
	var b strings.Builder

	frameworkStr := "@storybook/react"
	if fw == "vue" {
		frameworkStr = "@storybook/vue3"
	} else if fw == "svelte" {
		frameworkStr = "@storybook/svelte"
	} else if fw == "angular" {
		frameworkStr = "@storybook/angular"
	}

	b.WriteString(fmt.Sprintf("import type { Meta, StoryObj } from '%s';\n", frameworkStr))

	if comp.HasClick {
		b.WriteString("import { fn } from '@storybook/test';\n")
	}

	if fw == "angular" {
		b.WriteString("import { applicationConfig } from '@storybook/angular';\n")
		b.WriteString("import { provideHttpClient } from '@angular/common/http';\n")
		b.WriteString("import { provideRouter } from '@angular/router';\n")
		kebab := toKebabCase(comp.Name)
		fmt.Fprintf(&b, "import { %sComponent } from '../../app/components/%s/%s.component';\n", comp.Name, kebab, kebab)
	} else {
		ext := ""
		if fw == "vue" {
			ext = ".vue"
		} else if fw == "svelte" {
			ext = ".svelte"
		}
		fmt.Fprintf(&b, "import %s from '../../components/%s%s';\n", comp.Name, comp.Name, ext)
	}

	needsMock := false
	for _, prop := range comp.Props {
		if isDataModel(prop.Type, app) {
			needsMock = true
		}
	}
	if needsMock {
		b.WriteString("import * as mocks from '../../mocks/data';\n")
	}

	b.WriteString("\n")

	compRef := comp.Name
	if fw == "angular" {
		compRef = comp.Name + "Component"
	}

	b.WriteString("const meta = {\n")
	fmt.Fprintf(&b, "  title: 'Components/%s',\n", comp.Name)
	fmt.Fprintf(&b, "  component: %s,\n", compRef)
	b.WriteString("  tags: ['autodocs'],\n")

	if fw == "angular" {
		b.WriteString("  decorators: [\n")
		b.WriteString("    applicationConfig({ providers: [provideHttpClient(), provideRouter([])] }),\n")
		b.WriteString("  ],\n")
	}

	if comp.HasClick {
		b.WriteString("  args: { onClick: fn() },\n")
	}

	if hasArgTypes(comp) {
		b.WriteString("  argTypes: {\n")
		for _, prop := range comp.Props {
			if prop.Type == "enum" {
				fmt.Fprintf(&b, "    %s: { control: 'select' },\n", prop.Name)
			} else if prop.Type == "boolean" {
				fmt.Fprintf(&b, "    %s: { control: 'boolean' },\n", prop.Name)
			}
		}
		b.WriteString("  },\n")
	}
	fmt.Fprintf(&b, "} satisfies Meta<typeof %s>;\n\n", compRef)

	b.WriteString("export default meta;\n")
	b.WriteString("type Story = StoryObj<typeof meta>;\n\n")

	b.WriteString("export const Default: Story = {\n")
	if len(comp.Props) > 0 {
		b.WriteString("  args: {\n")
		for _, prop := range comp.Props {
			if isDataModel(prop.Type, app) {
				b.WriteString(fmt.Sprintf("    %s: mocks.mock%s(),\n", prop.Name, prop.Type))
			} else {
				b.WriteString(fmt.Sprintf("    %s: %s,\n", prop.Name, defaultArgValue(prop)))
			}
		}
		b.WriteString("  },\n")
	}
	b.WriteString("};\n")

	return b.String()
}

// hasArgTypes checks whether any prop needs a custom argType control.
func hasArgTypes(comp *ComponentMeta) bool {
	for _, prop := range comp.Props {
		if prop.Type == "enum" || prop.Type == "boolean" {
			return true
		}
	}
	return false
}

// defaultArgValue returns a sensible default arg literal for a prop type.
func defaultArgValue(prop *ir.Prop) string {
	switch strings.ToLower(prop.Type) {
	case "boolean":
		return "false"
	case "number", "decimal":
		return "0"
	default:
		return fmt.Sprintf("'Sample %s'", prop.Name)
	}
}

func generatePageStory(page *PageMeta, app *ir.Application, fw string) string {
	var b strings.Builder

	frameworkStr := "@storybook/react"
	if fw == "vue" {
		frameworkStr = "@storybook/vue3"
	} else if fw == "svelte" {
		frameworkStr = "@storybook/svelte"
	} else if fw == "angular" {
		frameworkStr = "@storybook/angular"
	}

	fmt.Fprintf(&b, "import type { Meta, StoryObj } from '%s';\n", frameworkStr)

	if fw == "angular" {
		b.WriteString("import { applicationConfig } from '@storybook/angular';\n")
		b.WriteString("import { provideHttpClient } from '@angular/common/http';\n")
		b.WriteString("import { provideRouter } from '@angular/router';\n")
		kebab := toKebabCase(page.Name)
		fmt.Fprintf(&b, "import { %sComponent } from '../../app/pages/%s/%s.component';\n", page.Name, kebab, kebab)
	} else {
		ext := ""
		if fw == "vue" {
			ext = ".vue"
		} else if fw == "svelte" {
			ext = ".svelte"
		}
		fmt.Fprintf(&b, "import %sPage from '../../pages/%sPage%s';\n", page.Name, page.Name, ext)
	}
	b.WriteString("\n")

	pageRef := page.Name + "Page"
	if fw == "angular" {
		pageRef = page.Name + "Component"
	}

	b.WriteString("const meta = {\n")
	fmt.Fprintf(&b, "  title: 'Pages/%s',\n", page.Name)
	fmt.Fprintf(&b, "  component: %s,\n", pageRef)
	b.WriteString("  parameters: {\n")
	b.WriteString("    layout: 'fullscreen',\n")
	b.WriteString("  },\n")
	if fw == "angular" {
		b.WriteString("  decorators: [\n")
		b.WriteString("    applicationConfig({ providers: [provideHttpClient(), provideRouter([])] }),\n")
		b.WriteString("  ],\n")
	}
	fmt.Fprintf(&b, "} satisfies Meta<typeof %s>;\n\n", pageRef)

	b.WriteString("export default meta;\n")
	b.WriteString("type Story = StoryObj<typeof meta>;\n\n")

	b.WriteString("export const Default: Story = {};\n")

	return b.String()
}

func isDataModel(typeName string, app *ir.Application) bool {
	for _, m := range app.Data {
		if m.Name == typeName {
			return true
		}
	}
	return false
}
