package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m model) wrappedLines() []string {
	lines := []string{}
	for _, line := range m.viewLines() {
		lines = append(lines, strings.Split(ansi.Hardwrap(ansi.Strip(line), max(1, m.width), true), "\n")...)
	}
	return lines
}

func (m model) viewOffset() int {
	lines := m.wrappedLines()
	height := max(1, m.height-2)
	limit := max(0, len(lines)-height)
	if m.manualScroll {
		return min(m.scroll, limit)
	}
	anchor := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "> ") {
			anchor = i
		}
		if m.editingPath && strings.Contains(line, "│") {
			anchor = i
		}
	}
	// Dialog choices and errors take priority over the selected page row.
	if m.unsaved || len(m.dialogs) > 0 || m.err != "" {
		anchor = len(lines) - 1
	}
	return min(limit, max(0, anchor-height+2))
}

func (m model) View() tea.View {
	lines := m.wrappedLines()
	height := max(1, m.height-2)
	start := m.viewOffset()
	end := min(len(lines), start+height)
	footer := "↑/↓ move · space toggle · tab next · s save · esc back · q quit"
	if m.editingPath {
		footer = "Enter accepts path · Esc restores · Ctrl+u clears"
	}
	if start > 0 || end < len(lines) {
		footer = fmt.Sprintf("pgup/pgdown scroll (%d–%d/%d) · %s", start+1, end, len(lines), footer)
	}
	content := append([]string{ansi.Truncate("agent-toolkit · setup and settings", max(1, m.width), "")}, lines[start:end]...)
	content = append(content, ansi.Truncate(footer, max(1, m.width), ""))
	if m.height < 3 {
		content = content[len(content)-max(1, m.height):]
	}
	v := tea.NewView(strings.Join(content, "\n"))
	v.AltScreen = true
	return v
}
