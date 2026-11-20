package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m model) overviewLines() []string {
	unique := map[string]bool{}
	installations := 0
	for _, names := range m.status.Observed {
		installations += len(names)
		for _, name := range names {
			unique[name] = true
		}
	}
	general := []string{"Overview", "", "Actions"}
	for i, name := range m.managementActions() {
		general = append(general, row(i == m.cursor, name))
	}
	general = append(general, "", "Installation",
		fmt.Sprintf("Agents          %s", strings.Join(m.installedProviders(), ", ")),
		fmt.Sprintf("Skills          %d unique · %d installations across agents", len(unique), installations))
	for _, id := range m.installedProviders() {
		general = append(general, fmt.Sprintf("%s: Installed skills: %d", providerName(id), len(m.status.Observed[id])))
		if !m.saved.SharedStorage {
			general = append(general, "Storage: "+outputDescription(m.saved.ForProvider(id).Output))
		}
	}
	general = append(general, "", "Storage")
	if m.saved.SharedStorage {
		general = append(general, "Shared Markdown output: "+outputDescription(m.saved.Output), "Layout: "+storageLayout(m.saved))
	} else {
		general = append(general, "Markdown output: Configured per LLM")
	}
	general = append(general, "", "Source", "Development checkout", m.root)
	general = append(general, "", "Refresh reads the current Development working tree, including uncommitted edits.")
	for _, message := range []string{m.status.Native.Message, m.notice, m.err} {
		if message != "" {
			general = append(general, "", message)
		}
	}

	return general
}

func (m model) overviewRows(width int) []string {
	var rows []string
	for _, text := range m.overviewLines() {
		for _, line := range strings.Split(text, "\n") {
			line = m.overviewLine(terminalText(line), width)
			rows = append(rows, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
		}
	}
	return rows
}
