package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m model) wrappedLines() []string {
	lines := []string{}
	width, _ := m.viewportSize()

	for _, line := range m.viewLines() {
		lines = append(lines, strings.Split(ansi.Hardwrap(ansi.Strip(line), width, true), "\n")...)
	}

	return lines
}

func (m model) viewportSize() (width, height int) {
	// Omit the frame when it would leave no room for content.
	if m.width < 5 || m.height < 6 {
		return max(1, m.width), max(1, m.height)
	}
	return m.width - 4, m.height - 5
}

func (m model) viewOffset() int {
	lines := m.wrappedLines()
	_, height := m.viewportSize()
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
	_, height := m.viewportSize()
	start := m.viewOffset()
	end := min(len(lines), start+height)
	if m.width < 5 || m.height < 6 {
		v := tea.NewView(strings.Join(m.styleLines(lines[start:end]), "\n"))
		v.AltScreen = true
		return v
	}

	footer := "↑/↓ move · space toggle · tab next · s save · esc back · q quit"
	if m.editingPath {
		footer = "Enter accepts path · Esc restores · Ctrl+u clears"
	}
	if start > 0 || end < len(lines) {
		footer = fmt.Sprintf("pgup/pgdown scroll (%d–%d/%d) · %s · t theme", start+1, end, len(lines), footer)
	} else {
		footer += " · t theme: " + uiThemes[m.themeIndex].name
	}

	width := max(4, m.width)
	innerWidth := width - 4
	border := m.style("│", "muted")
	content := []string{
		m.style("╭"+strings.Repeat("─", width-2)+"╮", "muted"),
		border + " " + padLine(m.style("agent-toolkit · setup and settings", "title"), innerWidth) + " " + border,
	}

	for _, line := range m.styleLines(lines[start:end]) {
		content = append(content, border+" "+padLine(line, innerWidth)+" "+border)
	}

	separator := m.style("├"+strings.Repeat("─", width-2)+"┤", "muted")
	content = append(content, separator)
	content = append(content, border+" "+padLine(m.style(footer, "muted"), innerWidth)+" "+border)
	content = append(content, m.style("╰"+strings.Repeat("─", width-2)+"╯", "muted"))

	v := tea.NewView(strings.Join(content, "\n"))
	v.AltScreen = true
	return v
}

func padLine(line string, width int) string {
	line = ansi.Truncate(line, width, "")
	return line + strings.Repeat(" ", max(0, width-ansi.StringWidth(line)))
}

func (m model) style(text, role string) string {
	theme := uiThemes[m.themeIndex]
	style := lipgloss.NewStyle()
	switch role {
	case "title":
		style = style.Foreground(lipgloss.Color(theme.accent)).Bold(true)
	case "heading":
		style = style.Foreground(lipgloss.Color(theme.heading)).Bold(true)
	case "selected":
		style = style.Foreground(lipgloss.Color(theme.selectedFG)).
			Background(lipgloss.Color(theme.selectedBG))
	case "error":
		style = style.Foreground(lipgloss.Color(theme.error)).Bold(true)
	case "notice":
		style = style.Foreground(lipgloss.Color(theme.notice))
	case "accent":
		style = style.Foreground(lipgloss.Color(theme.warning)).Bold(true)
	case "muted":
		style = style.Foreground(lipgloss.Color(theme.muted))
	}
	return style.Render(text)
}

func (m model) styleLines(lines []string) []string {
	styled := make([]string, len(lines))
	for i, line := range lines {
		role := ""
		plain := ansi.Strip(line)
		switch {
		case strings.HasPrefix(plain, "> "):
			role = "selected"
		case strings.HasPrefix(plain, "1. ") || strings.HasPrefix(plain, "2. ") ||
			strings.HasPrefix(plain, "3. ") || strings.HasPrefix(plain, "4. ") || strings.HasPrefix(plain, "5. "):
			role = "heading"
		case strings.Contains(plain, "[y/N]"):
			role = "accent"
		case m.err != "" && strings.Contains(plain, m.err):
			role = "error"
		case m.notice != "" && strings.Contains(plain, m.notice):
			role = "notice"
		}
		if role == "" {
			styled[i] = line
		} else {
			styled[i] = m.style(line, role)
		}
	}
	return styled
}
