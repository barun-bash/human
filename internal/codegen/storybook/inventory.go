package storybook

import (
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

type ComponentInventory struct {
	Components []*ComponentMeta
	Pages      []*PageMeta
}

type ComponentMeta struct {
	Name  string
	Props []*ir.Prop
	HasClick bool
}

type PageMeta struct {
	Name       string
	HasLoading bool
	HasEmpty   bool
	EmptyText  string
}

func BuildInventory(app *ir.Application) *ComponentInventory {
	inv := &ComponentInventory{}

	for _, comp := range app.Components {
		meta := &ComponentMeta{
			Name:  comp.Name,
			Props: comp.Props,
		}
		for _, a := range comp.Content {
			lower := strings.ToLower(a.Text)
			if strings.Contains(lower, "click") || strings.Contains(lower, "on_click") {
				meta.HasClick = true
			}
		}
		inv.Components = append(inv.Components, meta)
	}

	for _, page := range app.Pages {
		meta := &PageMeta{
			Name: page.Name,
		}
		for _, a := range page.Content {
			if a.Type == "condition" {
				lower := strings.ToLower(a.Text)
				if strings.Contains(lower, "loading") {
					meta.HasLoading = true
				}
				if strings.Contains(lower, "if no") || strings.Contains(lower, "empty") {
					meta.HasEmpty = true
					meta.EmptyText = a.Text
				}
			}
		}
		inv.Pages = append(inv.Pages, meta)
	}

	return inv
}
