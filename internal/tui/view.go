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
		// Newlines here separate trusted layout rows. Escape every other control
		// before wrapping; stripping ANSI alone would silently hide source bytes.
		for _, row := range strings.Split(line, "\n") {
			lines = append(lines, strings.Split(ansi.Hardwrap(terminalText(row), width, true), "\n")...)
		}
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
	if m.unsaved {
		background := m
		background.unsaved = false
		v := tea.NewView(m.unsavedOverlay(background.View().Content))
		v.AltScreen = true
		return v
	}
	lines := m.wrappedLines()
	_, height := m.viewportSize()
	start := m.viewOffset()
	end := min(len(lines), start+height)
	if m.width < 5 || m.height < 6 {
		v := tea.NewView(strings.Join(m.styleLines(lines[start:end]), "\n"))
		v.AltScreen = true
		return v
	}

	footer := "↑/↓ move · ↵ select · tab next · s save · Esc/⌫ back · q quit"
	if len(m.dialogs) > 0 {
		footer = "v view details · y confirm · Esc/⌫ cancel"
		if m.reviewDetails {
			footer = "↑/↓ scroll · v summary · y confirm · Esc/⌫ cancel"
		}
	}
	if m.conflictReview {
		footer = "↑/↓ scroll · ↵/Esc/⌫ back · q quit"
	}
	if m.editingPath {
		footer = "↵ accepts path · Esc/Ctrl+c restores · Ctrl+u clears"
	}
	if start > 0 || end < len(lines) {
		footer = fmt.Sprintf("pgup/pgdown scroll (%d–%d/%d) · %s", start+1, end, len(lines), footer)
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

func (m model) unsavedOverlay(background string) string {
	width, height := max(1, m.width), max(1, m.height)
	boxWidth := min(72, width)
	innerWidth := max(1, boxWidth-4)
	text := "Unsaved changes\n\n" + unsavedPrompt + "\n\nEsc/⌫: keep editing"
	rows := strings.Split(ansi.Hardwrap(text, innerWidth, true), "\n")
	if len(rows)+2 > height {
		text = "Save changes?\ns save / d discard\nEsc/⌫: keep editing"
		rows = strings.Split(ansi.Hardwrap(text, innerWidth, true), "\n")
	}

	box := []string{}
	if boxWidth >= 6 && height >= 5 {
		box = append(box, "╭"+strings.Repeat("─", boxWidth-2)+"╮")
		for _, line := range rows[:min(len(rows), height-2)] {
			box = append(box, "│ "+padLine(line, innerWidth)+" │")
		}
		box = append(box, "╰"+strings.Repeat("─", boxWidth-2)+"╯")
	} else {
		boxWidth = width
		rows = strings.Split(ansi.Hardwrap("s save / d discard / Esc/⌫ cancel", width, true), "\n")
		box = rows[:min(len(rows), height)]
	}

	left, top := (width-boxWidth)/2, (height-len(box))/2
	canvas := strings.Split(background, "\n")
	for len(canvas) < height {
		canvas = append(canvas, "")
	}
	for i, line := range box {
		base := padLine(canvas[top+i], width)
		canvas[top+i] = ansi.Cut(base, 0, left) + m.style(padLine(line, boxWidth), "selected") + ansi.Cut(base, left+boxWidth, width)
	}
	return strings.Join(canvas[:height], "\n")
}

func padLine(line string, width int) string {
	line = ansi.Truncate(line, width, "")
	return line + strings.Repeat(" ", max(0, width-ansi.StringWidth(line)))
}

func (m model) style(text, role string) string {
	theme := defaultTheme
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
