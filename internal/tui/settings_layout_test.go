package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestSettingsGroupingAndFocus(t *testing.T) {
	for _, providers := range [][]string{nil, {"codex"}, {"codex", "claude"}} {
		for _, external := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/external=%t", providers, external), func(t *testing.T) {
				m, _ := readyModel(t, false)
				m.page = 2
				m.draft.Providers = providers
				m.draft.SharedStorage = true
				m.globalOutput = external
				m.outputPath = "/tmp/artifacts/a-long-output-directory"
				text := strings.Join(m.settingsLines(), "\n")
				storage, guidance := strings.Index(text, "Artifact storage"), strings.Index(text, "Agent guidance")
				if storage < 0 || guidance <= storage || strings.Contains(text, "Other settings") {
					t.Fatal(text)
				}
				if strings.Index(text, "Separate by LLM") > guidance || strings.Index(text, "Layout:") > guidance {
					t.Fatal("storage controls outside storage section", text)
				}
				if len(providers) == 0 && !strings.Contains(text, "Enable an LLM") {
					t.Fatal("missing empty guidance message", text)
				}
				for _, id := range providers {
					if strings.Index(text, "  "+providerName(id)+"\n") < guidance {
						t.Fatal("provider outside guidance section", text)
					}
				}
				for _, size := range [][2]int{{80, 24}, {120, 35}} {
					m.width, m.height = size[0], size[1]
					for i, r := range m.globalSettingsRows() {
						m.cursor = i
						lines := m.settingsLines()
						var selected []string
						for _, line := range lines {
							if strings.HasPrefix(line, "> ") {
								selected = append(selected, line)
							}
						}
						if len(selected) != 1 {
							t.Fatalf("row %v: selections %v", r, selected)
						}
						visible := ansi.Strip(m.View().Content)
						if !strings.Contains(visible, "> ") {
							t.Fatalf("focus hidden for %v: %s", r, visible)
						}
						if r.provider != "" {
							name := map[string]string{"guidance": "Coding guidance", "documentation": "Documentation guidance", "concise": "Concise replies"}[r.kind]
							if !strings.Contains(selected[0], name) {
								t.Fatalf("wrong row for %v: %s", r, selected[0])
							}
						}
					}
				}
				heading := ansi.Strip(m.contentLine("Agent guidance", 76, false))
				provider := ansi.Strip(m.contentLine("  Codex", 76, false))
				if !strings.Contains(heading, "─") || strings.Contains(provider, "─") || provider != "  Codex" {
					t.Fatalf("incorrect hierarchy: %q / %q", heading, provider)
				}
			})
		}
	}
}
