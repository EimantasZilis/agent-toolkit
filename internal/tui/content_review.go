package tui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

// terminalText escapes controls visibly so source bytes cannot issue terminal
// commands, hide review rows, or reorder names through Unicode format controls.
func terminalText(value string) string {
	var out strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029' {
			quoted := strconv.QuoteRune(r)
			out.WriteString(quoted[1 : len(quoted)-1])
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func contentReviewLines(changes []install.ContentChange) []string {
	if len(changes) == 0 {
		return []string{"Installed content: unchanged."}
	}
	lines := []string{"Content changes (guidance shows managed blocks only):"}
	provider, name := "", ""
	for _, change := range changes {
		if change.Provider != provider || change.Name != name {
			provider, name = change.Provider, change.Name
			lines = append(lines, terminalText(provider+" / "+name))
		}
		lines = append(lines, "  "+terminalText(change.Kind+" "+change.Path), "    "+reviewMode(change.Before)+" -> "+reviewMode(change.After))
		if change.Binary {
			lines = append(lines, "    Binary content:", "    before: "+reviewBytes(change.Before), "    after:  "+reviewBytes(change.After))
		}
		for _, line := range change.Lines {
			lines = append(lines, "    "+terminalText(line))
		}
		if change.Truncated {
			lines = append(lines, "    [Text excerpt truncated: limited to 100 lines / 8192 bytes; path and modes retained.]")
		}
	}
	return lines
}

func (m model) reviewSummaryLines() []string {
	providers := make([]string, 0, len(m.preview.Desired.Providers))
	for _, id := range m.preview.Desired.Providers {
		providers = append(providers, providerName(id))
	}
	counts := map[string]int{}
	for _, op := range m.preview.Operations {
		counts[op.Action]++
	}
	lines := []string{
		"Review changes",
		"",
		"LLMs: " + strings.Join(providers, ", "),
		fmt.Sprintf("Skills selected: %d", len(m.preview.Selection)),
		"",
	}
	if m.preview.Desired.ByProvider != nil {
		for _, id := range m.preview.Desired.Providers {
			p := m.preview.Desired.ForProvider(id)
			lines = append(lines, fmt.Sprintf("%s: %d selected skills · coding guidance %t · documentation %t · concise replies %t", providerName(id), len(m.preview.Plan.SelectionByProvider[id]), p.FullTemplate, p.DocumentationEnabled(), p.Concise))
			if !m.preview.Desired.SharedStorage {
				lines = append(lines, "  Markdown output: "+outputDescription(p.Output))
			}
		}
	} else {
		lines = append(lines, fmt.Sprintf("Coding guidance: %t · Documentation: %t · Concise replies: %t", m.preview.Desired.FullTemplate, m.preview.Desired.DocumentationEnabled(), m.preview.Desired.Concise), "Markdown output: "+outputDescription(m.preview.Desired.Output))
	}
	if m.preview.Desired.SharedStorage {
		lines = append(lines, "Shared Markdown output: "+outputDescription(m.preview.Desired.Output), "Layout: "+storageLayout(m.preview.Desired))
	}
	var actions []string
	for _, action := range []string{"install", "update", "convert", "remove"} {
		if counts[action] > 0 {
			actions = append(actions, fmt.Sprintf("%s: %d", action, counts[action]))
		}
	}
	if len(actions) > 0 {
		lines = append(lines, strings.Join(actions, " · "))
	}
	if m.preview.Desired.ByProvider == nil && m.preview.Desired.FullTemplate != m.preview.Desired.Concise {
		lines = append(lines, fmt.Sprintf("Saved preferences: coding %t, documentation %t, concise-output %t", m.preview.Desired.FullTemplate, m.preview.Desired.DocumentationEnabled(), m.preview.Desired.Concise))
	}
	lines = append(lines, fmt.Sprintf("Changed paths: %d", len(m.preview.Changes)), "v: view paths and file diffs")
	if m.preview.Native.Message != "" {
		lines = append(lines, "", m.preview.Native.Message)
	}
	for i := range lines {
		lines[i] = terminalText(lines[i])
	}
	return lines
}

func reviewMode(state *install.ContentState) string {
	if state == nil {
		return "absent"
	}
	return fmt.Sprintf("%s mode %04o", state.Type, state.Mode)
}

func reviewBytes(state *install.ContentState) string {
	if state == nil {
		return "absent"
	}
	if state.Type == "directory" {
		return "directory"
	}
	return fmt.Sprintf("%d bytes, sha256 %s", state.Size, state.SHA256)
}
