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
	b.WriteString("import { fn } from '@storybook/test';\n")

	ext := ""
	if fw == "vue" {
		ext = ".vue"
	} else if fw == "svelte" {
		ext = ".svelte"
	}
	b.WriteString(fmt.Sprintf("import %s from '../../components/%s%s';\n", comp.Name, comp.Name, ext))

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

	b.WriteString(fmt.Sprintf("const meta: Meta<typeof %s> = {\n", comp.Name))
	b.WriteString(fmt.Sprintf("  title: 'Components/%s',\n", comp.Name))
	b.WriteString(fmt.Sprintf("  component: %s,\n", comp.Name))
	b.WriteString("  tags: ['autodocs'],\n")
	
	if comp.HasClick {
		b.WriteString("  args: { onClick: fn() },\n")
	}

	b.WriteString("  argTypes: {\n")
	for _, prop := range comp.Props {
		if prop.Type == "enum" {
			b.WriteString(fmt.Sprintf("    %s: { control: 'select', options: ['a', 'b', 'c'] },\n", prop.Name))
		} else if prop.Type == "boolean" {
			b.WriteString(fmt.Sprintf("    %s: { control: 'boolean' },\n", prop.Name))
		}
	}
	b.WriteString("  },\n")
	b.WriteString("};\n\n")

	b.WriteString("export default meta;\n")
	b.WriteString(fmt.Sprintf("type Story = StoryObj<typeof %s>;\n\n", comp.Name))

	b.WriteString("export const Default: Story = {\n")
	b.WriteString("  args: {\n")
	for _, prop := range comp.Props {
		if isDataModel(prop.Type, app) {
			b.WriteString(fmt.Sprintf("    %s: mocks.mock%s(),\n", prop.Name, prop.Type))
		} else {
			b.WriteString(fmt.Sprintf("    %s: 'Sample %s',\n", prop.Name, prop.Name))
		}
	}
	b.WriteString("  },\n")
	b.WriteString("};\n\n")

	// Additional Component Variants
	b.WriteString("export const Loading: Story = {\n")
	b.WriteString("  args: {\n")
	b.WriteString("    ...Default.args,\n")
	b.WriteString("    loading: true,\n")
	b.WriteString("  },\n")
	b.WriteString("};\n\n")

	b.WriteString("export const Empty: Story = {\n")
	b.WriteString("  args: {\n")
	b.WriteString("    ...Default.args,\n")
	b.WriteString("    data: [],\n")
	b.WriteString("  },\n")
	b.WriteString("};\n\n")

	b.WriteString("export const ErrorState: Story = {\n")
	b.WriteString("  args: {\n")
	b.WriteString("    ...Default.args,\n")
	b.WriteString("    error: 'An unexpected error occurred',\n")
	b.WriteString("  },\n")
	b.WriteString("};\n")

	return b.String()
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

	b.WriteString(fmt.Sprintf("import type { Meta, StoryObj } from '%s';\n", frameworkStr))
	
	ext := ""
	if fw == "vue" {
		ext = ".vue"
	} else if fw == "svelte" {
		ext = ".svelte"
	}
	b.WriteString(fmt.Sprintf("import %sPage from '../../pages/%sPage%s';\n", page.Name, page.Name, ext))
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("const meta: Meta<typeof %sPage> = {\n", page.Name))
	b.WriteString(fmt.Sprintf("  title: 'Pages/%s',\n", page.Name))
	b.WriteString(fmt.Sprintf("  component: %sPage,\n", page.Name))
	b.WriteString("  parameters: {\n")
	b.WriteString("    layout: 'fullscreen',\n")
	b.WriteString("  },\n")
	b.WriteString("};\n\n")

	b.WriteString("export default meta;\n")
	b.WriteString(fmt.Sprintf("type Story = StoryObj<typeof %sPage>;\n\n", page.Name))

	b.WriteString("export const Default: Story = {};\n\n")

	if page.HasLoading {
		b.WriteString("export const Loading: Story = {\n")
		b.WriteString("  parameters: { mockData: [] },\n")
		b.WriteString("};\n\n")
	}

	if page.HasEmpty {
		b.WriteString("export const Empty: Story = {\n")
		b.WriteString("  parameters: { mockData: [] },\n")
		b.WriteString("};\n\n")
	}

	b.WriteString("export const Error: Story = {\n")
	b.WriteString("  parameters: { mockData: { error: true } },\n")
	b.WriteString("};\n")

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
