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
	if m.loaded && m.page == managementPage && !m.help && !m.conflictReview && !m.errorReview && len(m.dialogs) == 0 {
		return m.overviewRows(width)
	}

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
	width, height = m.width-4, m.height-8
	if m.width < 20 || m.height < 10 {
		return max(1, m.width), max(1, m.height-1)
	}
	if m.skillDescriptionVisible() {
		width, _ = m.skillPaneWidths()
		if m.width < 110 {
			height -= m.skillPanelHeight() + 1
		}
	}
	return max(1, width), max(1, height)
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
		if strings.HasPrefix(ansi.Strip(line), "> ") {
			anchor = i
		}
		if m.editingPath && strings.Contains(line, "│") {
			anchor = i
		}
	}
	// Keep the resulting layout visible while changing LLM separation.
	if m.page == 2 && !m.help && m.focusedSetting("split") {
		for i, line := range lines {
			if strings.HasPrefix(line, "Layout:") {
				anchor = i
			}
		}
	}
	// Dialog choices and errors take priority over the selected page row.
	if m.unsaved || len(m.dialogs) > 0 || m.err != "" {
		anchor = len(lines) - 1
	}
	return min(limit, max(0, anchor-height+2))
}

// View keeps navigation and actions outside the independently scrolling body.
func (m model) View() tea.View {
	if m.unsaved {
		background := m
		background.unsaved, background.refreshPending = false, false
		v := tea.NewView(m.unsavedOverlay(background.View().Content))
		v.AltScreen = true
		return v
	}
	width, height := max(1, m.width), max(1, m.height)
	if width < 20 || height < 10 {
		lines := m.bodyRows(width)
		start := min(m.viewOffset(), max(0, len(lines)-max(1, height-1)))
		end := min(len(lines), start+max(1, height-1))
		content := strings.Join(lines[start:end], "\n")
		if height > 1 {
			shortcuts := "pgup/pgdown · y confirm · Esc back"
			if len(m.dialogs) > 0 {
				shortcuts = "[y/N] · " + shortcuts
			}
			content += "\n" + m.style(padLine(shortcuts, width), "muted")
		}
		v := tea.NewView(content)
		v.AltScreen = true
		return v
	}
	inner := width - 4
	bodyWidth := inner
	showDescription := m.skillDescriptionVisible()
	split := width >= 110 && showDescription
	_, descriptionWidth := m.skillPaneWidths()
	if split {
		bodyWidth, _ = m.skillPaneWidths()
	}
	bodyHeight := height - 8
	if showDescription && !split {
		bodyHeight -= m.skillPanelHeight() + 1
	}
	bodyHeight = max(1, bodyHeight)
	var description []string
	if split {
		description = m.skillDescriptionRows(descriptionWidth)
		description = clippedRows(description, bodyHeight, descriptionWidth)
	}
	rows := m.bodyRows(bodyWidth)
	start := min(m.scroll, max(0, len(rows)-bodyHeight))
	if !m.manualScroll {
		anchor := 0
		for i, row := range rows {
			if strings.HasPrefix(ansi.Strip(row), "> ") {
				anchor = i
			}
		}
		if m.page == 2 && !m.help && len(m.dialogs) == 0 && m.focusedSetting("split") {
			for i, row := range rows {
				if strings.HasPrefix(ansi.Strip(row), "Layout:") {
					anchor = i
				}
			}
		}
		start = min(max(0, anchor-bodyHeight+2), max(0, len(rows)-bodyHeight))
	}
	border := m.style("│", "muted")
	rule := func(left, right string) string { return m.style(left+strings.Repeat("─", width-2)+right, "muted") }
	line := func(text string) string { return border + " " + padLine(text, inner) + " " + border }
	out := []string{rule("╭", "╮"), line(m.style("agent-toolkit", "title") + " · " + m.screenTitle()), line(m.tabs(inner)), rule("├", "┤")}
	for i := 0; i < bodyHeight; i++ {
		text := ""
		if start+i < len(rows) {
			text = rows[start+i]
		}
		if split {
			detail := ""
			if i < len(description) {
				detail = description[i]
			}
			text = padLine(text, bodyWidth) + " " + border + " " + padLine(detail, descriptionWidth)
		}
		out = append(out, line(text))
	}
	if showDescription && !split {
		out = append(out, rule("├", "┤"))
		panel := clippedRows(m.skillDescriptionRows(inner), m.skillPanelHeight(), inner)
		for i := 0; i < m.skillPanelHeight(); i++ {
			text := ""
			if i < len(panel) {
				text = panel[i]
			}
			out = append(out, line(text))
		}
	}

	status := "Saved configuration"
	if m.wizard {
		status = "Setup · Choose LLMs, skills, output, and guidance"
	}
	if m.dirty() {
		status = m.style("Draft changed · Installed files unchanged", "accent")
	}
	if m.notice != "" {
		status = m.style(terminalText(m.notice), "notice")
	}
	if m.err != "" {
		status = m.style("e: full error · "+terminalText(m.err), "error")
	}
	if m.dirty() {
		status = m.style("Unsaved configuration · ", "accent") + status
	}
	if len(m.dialogs) > 0 {
		prompt := strings.TrimSuffix(strings.Split(m.dialogs[0].text, "\n")[0], " [y/N]")
		status = m.style(ansi.Truncate(terminalText(prompt), max(1, inner-7), "…")+" [y/N]", "accent")
	}
	if m.busy {
		status = "Working… Please wait for the operation to finish."
	}
	if start > 0 || start+bodyHeight < len(rows) {
		status += fmt.Sprintf(" · %d–%d/%d", start+1, min(start+bodyHeight, len(rows)), len(rows))
	}
	out = append(out, rule("├", "┤"), line(status), line(m.style(m.shortcuts(), "muted")), rule("╰", "╯"))
	v := tea.NewView(strings.Join(out, "\n"))
	v.AltScreen = true
	return v
}

func (m model) screenTitle() string {
	if !m.loaded {
		return "Loading"
	}
	if m.errorReview {
		return "Operation error"
	}
	if m.conflictReview {
		return "Installation blocked"
	}
	if len(m.dialogs) > 0 {
		return "Review changes"
	}
	if m.page == managementPage {
		return "Overview"
	}
	if m.page >= 0 && m.page < len(pageNames) {
		return pageNames[m.page]
	}
	return "Configuration"
}

func (m model) tabs(width int) string {
	if len(m.dialogs) > 0 {
		if m.reviewDetails {
			return m.style("Summary", "muted") + "   " + m.style("[Details]", "selected")
		}
		return m.style("[Summary]", "selected") + "   " + m.style("Details", "muted")
	}
	names := []string{"Overview", "LLMs", "Skills", "Global settings"}
	if width < 60 {
		names = []string{"Home", "LLMs", "Skills", "Settings"}
	}
	pages := []int{managementPage, 0, 1, 2}
	tabs := []string{}
	for i, name := range names {
		text := fmt.Sprintf("%d %s", i+1, name)
		if m.page == pages[i] {
			text = m.style("["+text+"]", "selected")
		} else {
			text = m.style(text, "muted")
		}
		tabs = append(tabs, text)
	}
	return strings.Join(tabs, "  ")
}

func (m model) shortcuts() string {
	if m.busy {
		return "Applying or rolling back · Input paused"
	}
	if m.help {
		return "? / Esc close help"
	}
	if m.editingPath {
		return "Enter accept · Esc restore · Ctrl+u clear"
	}
	if m.conflictReview || m.errorReview {
		return "↑↓ / PgUp/PgDn scroll · Enter/Esc back · q quit"
	}
	if len(m.dialogs) > 0 {
		if m.reviewDetails {
			return "pgup/pgdown scroll · v summary · y confirm · Enter/Esc cancel"
		}
		return "v view details · y confirm · Enter/Esc cancel"
	}
	if m.page == 1 {
		return "↑↓ move · Space toggle · ←→ fold · a all · s review · ? help"
	}
	return "↑↓ move · Enter select · Tab section · s review · ? help · q quit"
}

func (m model) bodyRows(width int) []string {
	if m.loaded && m.page == managementPage && !m.help && !m.conflictReview && !m.errorReview && len(m.dialogs) == 0 {
		return m.overviewRows(width)
	}
	lines := m.viewLines()
	if m.help {
		lines = []string{"Keyboard help", "", "Tab / Shift+Tab: next / previous section", "1–4: jump to a section", "Up / Down: move between rows", "Space / Enter: toggle selection", "Left / Right: collapse / expand a category", "a: select or deselect all skills", "s: review configuration", "v: switch review summary and details", "Page Up / Page Down: scroll", "q: quit · unsaved changes are protected"}
	}
	rows := []string{}
	for index, text := range lines {
		for _, line := range strings.Split(text, "\n") {
			line = terminalText(line)
			line = m.contentLine(line, width, index == 0)
			rows = append(rows, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
		}
	}
	return rows
}

// overviewLine receives escaped text so saved paths cannot inject terminal styles.
func (m model) overviewLine(line string, width int) string {
	switch line {
	case "Overview", "Storage", "Installation", "Source", "Actions", "Codex", "Claude Code":
		return m.sectionHeading(line, width)
	case "Development checkout":
		return m.style(line, "value")
	}
	for _, key := range []string{"LLMs", "Skills", "Markdown", "Guidance"} {
		label := fmt.Sprintf("%-16s", key)
		if strings.HasPrefix(line, label) {
			value := strings.TrimPrefix(line, label)
			role := "value"
			if key == "Guidance" && value == "Enabled" {
				role = "notice"
			}
			return m.style(label, "muted") + m.style(value, role)
		}
	}
	if strings.HasPrefix(line, "> ") {
		return m.style(line+strings.Repeat(" ", max(0, min(width, 56)-ansi.StringWidth(line))), "selected")
	}
	if strings.HasPrefix(line, "  ") {
		role := "value"
		if line == "  Uninstall" {
			role = "error"
		}
		return m.style(line, role)
	}
	return m.contentLine(line, width, false)
}

func (m model) sectionHeading(text string, width int) string {
	remaining := min(width, 56) - ansi.StringWidth(text) - 1
	if remaining > 0 {
		text += " " + strings.Repeat("─", remaining)
	}
	return m.style(text, "heading")
}

// contentLine styles escaped layout text before wrapping. Review excerpts keep
// their indentation and diff markers so source content remains distinguishable.
func (m model) contentLine(line string, width int, heading bool) string {
	switch {
	case m.err != "" && strings.Contains(line, terminalText(m.err)):
		return m.style(line, "error")
	case m.notice != "" && strings.Contains(line, terminalText(m.notice)):
		return m.style(line, "notice")
	case strings.Contains(line, "[y/N]"):
		return m.style(line, "accent")
	case !m.help && len(m.dialogs) == 0 && m.page == 2 && (line == "  Codex" || line == "  Claude Code"):
		return m.style(line, "heading")
	case !m.help && len(m.dialogs) == 0 && m.page == 2 && (strings.HasPrefix(line, "  Installed guidance:") || strings.HasPrefix(line, "Applies ") || strings.HasPrefix(line, "Changing location ") || strings.HasPrefix(line, "Turning off ")):
		return m.style(line, "muted")
	case !m.help && len(m.dialogs) == 0 && (line == "Codex" || line == "Claude Code"):
		return m.sectionHeading(line, width)
	case !m.help && len(m.dialogs) == 0 && m.page == 2 && (line == "Artifact storage" || line == "Agent guidance"):
		return m.sectionHeading(line, width)
	case heading:
		return m.sectionHeading(line, width)
	case len(m.dialogs) > 0 && m.reviewDetails:
		switch {
		case strings.HasPrefix(line, "    +"):
			return m.style(line, "notice")
		case strings.HasPrefix(line, "    -"):
			return m.style(line, "error")
		case line == "Content changes (guidance shows managed blocks only):":
			return m.sectionHeading(line, width)
		case line != "" && !strings.HasPrefix(line, " "):
			if key, value, ok := strings.Cut(line, ": "); ok {
				return m.style(key+": ", "muted") + m.style(value, "value")
			}
			return m.style(line, "heading")
		default:
			return m.style(line, "muted")
		}
	case strings.HasPrefix(line, "> "):
		return m.style(line+strings.Repeat(" ", max(0, min(width, 56)-ansi.StringWidth(line))), "selected")
	}
	if !m.help && len(m.dialogs) == 0 && m.page == 2 && strings.HasPrefix(line, "    ") && !strings.HasPrefix(strings.TrimSpace(line), "Path: ") && !strings.Contains(line, "[x]") && !strings.Contains(line, "[ ]") {
		return m.style(line, "muted")
	}
	if !m.help && len(m.dialogs) == 0 && strings.HasPrefix(line, "  ") {
		if strings.Contains(line, "▸ ") || strings.Contains(line, "▾ ") {
			return m.style(line, "heading")
		}
		for _, mark := range []string{"[x]", "[ ]", "[-]"} {
			if before, after, ok := strings.Cut(line, mark); ok {
				role := "muted"
				if mark == "[x]" {
					role = "notice"
				}
				name, description, hasDescription := strings.Cut(after, " — ")
				name, suffix, _ := strings.Cut(name, " (")
				text := m.style(before+mark, role) + m.style(name, "value")
				if suffix != "" {
					text += m.style(" ("+suffix, "muted")
				}
				if hasDescription {
					text += m.style(" — "+description, "muted")
				}
				return text
			}
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "Path: ") {
			return m.style(line, "value")
		}
	}
	if key, value, ok := strings.Cut(line, ": "); ok && (m.help || len(m.dialogs) > 0 || strings.HasPrefix(strings.TrimSpace(line), "Path: ") || m.page == 3) {
		return m.style(key+": ", "muted") + m.style(value, "value")
	}
	return m.style(line, "muted")
}

func clippedRows(rows []string, height, width int) []string {
	if len(rows) <= height {
		return rows
	}
	rows = append([]string(nil), rows[:height]...)
	rows[height-1] = ansi.Truncate(rows[height-1], max(0, width-1), "") + "…"
	return rows
}

func (m model) skillPaneWidths() (list, description int) {
	inner := max(1, m.width-4)
	if m.width >= 110 {
		list = min(56, inner/2)
		return list, inner - list - 3
	}
	return inner, 0
}

func (m model) skillPanelHeight() int {
	return min(10, max(4, (m.height-8)/2))
}

func (m model) skillDescriptionRows(width int) []string {
	rows := m.skillRows()
	if m.cursor < 0 || m.cursor >= len(rows) {
		return nil
	}
	r := rows[m.cursor]
	description := "Select a skill to read what it does."
	if r.name != "" {
		description = "No description available."
		for _, item := range m.catalog.Skills {
			if item.Name == r.name && strings.TrimSpace(item.Description) != "" {
				description = item.Description
				break
			}
		}
	}
	// Escape metadata before adding our own ANSI styles or wrapping it.
	text := strings.Join(strings.Fields(terminalText(description)), " ")
	wrapped := strings.Split(ansi.Wrap(text, width, ""), "\n")
	out := []string{m.sectionHeading("Description", width), ""}
	for _, row := range wrapped {
		out = append(out, m.style(row, "value"))
	}
	return out
}

func (m model) skillDescriptionVisible() bool {
	if m.height < 14 || m.page != 1 || m.unsaved || m.conflictReview || m.errorReview || m.help || len(m.dialogs) > 0 {
		return false
	}
	return m.cursor >= 0 && m.cursor < len(m.skillRows())
}

func (m model) unsavedOverlay(background string) string {
	width, height := max(1, m.width), max(1, m.height)
	boxWidth := min(72, width)
	innerWidth := max(1, boxWidth-4)
	text := "Unsaved changes\n\n" + unsavedPrompt + "\n\nEsc/⌫: keep editing"
	if m.refreshPending {
		text = "Unsaved configuration\n\nRefresh uses saved settings.\n[d] Discard draft and refresh\nEnter/Esc: return to editing"
	}
	rows := strings.Split(ansi.Hardwrap(text, innerWidth, true), "\n")
	if len(rows)+2 > height {
		text = "Save changes?\ns save / d discard\nEsc/⌫: keep editing"
		if m.refreshPending {
			text = "Discard draft and refresh?\nd discard and refresh\nEnter/Esc: keep editing"
		}
		rows = strings.Split(ansi.Hardwrap(text, innerWidth, true), "\n")
	}

	box := []string{}
	if boxWidth >= 6 && height >= 5 {
		box = append(box, m.style("╭"+strings.Repeat("─", boxWidth-2)+"╮", "heading"))
		for i, line := range rows[:min(len(rows), height-2)] {
			box = append(box, m.style("│ ", "heading")+padLine(m.contentLine(line, innerWidth, i == 0), innerWidth)+m.style(" │", "heading"))
		}
		box = append(box, m.style("╰"+strings.Repeat("─", boxWidth-2)+"╯", "heading"))
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
		canvas[top+i] = ansi.Cut(base, 0, left) + m.style(padLine(line, boxWidth), "panel") + ansi.Cut(base, left+boxWidth, width)
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
	case "panel":
		style = style.Background(lipgloss.Color(theme.selectedBG))
	case "value":
		style = style.Bold(true)
	case "selected":
		style = style.Bold(true).Foreground(lipgloss.Color(theme.selectedFG)).
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
