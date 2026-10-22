package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

func (m *model) syncOutput() {
	m.globalOutput = m.draft.Output != ""
	m.outputPath = m.draft.Output
	if m.outputPath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			m.outputPath = filepath.Join(home, "agent-toolkit-output")
		}
	}
	m.editingPath = false
}

// validateOutput only inspects the filesystem. Creation belongs to Bash apply.
func (m model) validateOutput() (bool, error) {
	if !m.globalOutput {
		return false, nil
	}
	path := m.draft.Output
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) || strings.ContainsFunc(path, unicode.IsControl) {
		return false, fmt.Errorf("Page 3: enter an absolute, single-line global output path")
	}

	missing := false
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Stat(current)
		if os.IsNotExist(err) {
			// A dangling symlink is not a directory we can create.
			if _, linkErr := os.Lstat(current); linkErr == nil {
				return false, fmt.Errorf("Page 3: output path contains a dangling link: %s", current)
			}
			missing = true
			continue
		}
		if err != nil {
			return false, fmt.Errorf("Page 3: inspect output path: %w", err)
		}
		if !info.IsDir() {
			return false, fmt.Errorf("Page 3: output path is not a directory: %s", current)
		}
		// Access checks write and search permissions without creating a probe file.
		// Supported release targets are macOS and Linux.
		if err := syscall.Access(current, 3); err != nil {
			return false, fmt.Errorf("Page 3: output directory is not writable/searchable: %w", err)
		}
		return missing, nil
	}
}

func (m *model) insertPath(text string) {
	if strings.ContainsFunc(text, unicode.IsControl) {
		m.err = "The output path must be a single line without control characters."
		return
	}
	runes := []rune(m.outputPath)
	position := min(m.pathCursor, len(runes))
	m.outputPath = string(runes[:position]) + text + string(runes[position:])
	m.pathCursor = position + len([]rune(text))
	m.draft.Output = m.outputPath
	m.err = ""
}

func (m model) editPath(key string) (tea.Model, tea.Cmd) {
	runes := []rune(m.outputPath)
	switch key {
	case "enter", "tab":
		m.editingPath = false
	case "esc", "ctrl+c":
		m.outputPath, m.draft.Output = m.originalPath, m.originalPath
		m.editingPath = false
		m.err = ""
	case "left":
		m.pathCursor = max(0, m.pathCursor-1)
	case "right":
		m.pathCursor = min(len(runes), m.pathCursor+1)
	case "home", "ctrl+a":
		m.pathCursor = 0
	case "end", "ctrl+e":
		m.pathCursor = len(runes)
	case "ctrl+u":
		m.outputPath, m.draft.Output, m.pathCursor = "", "", 0
	case "backspace":
		if m.pathCursor > 0 {
			m.outputPath = string(runes[:m.pathCursor-1]) + string(runes[m.pathCursor:])
			m.pathCursor--
		}
	case "delete":
		if m.pathCursor < len(runes) {
			m.outputPath = string(runes[:m.pathCursor]) + string(runes[m.pathCursor+1:])
		}
	default:
		// Printable input is handled via KeyPressMsg.Text and PasteMsg.
	}
	m.draft.Output = m.outputPath
	return m, nil
}

func (m *model) editSettings(key string) {
	if m.page == 3 {
		if key == "space" {
			m.draft.Concise = !m.draft.Concise
		}
		return
	}
	if m.page != 2 {
		return
	}
	last := 1
	if m.globalOutput {
		last = 2
	}
	switch key {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(last, m.cursor+1)
	case "space":
		if m.cursor < 2 {
			m.globalOutput = m.cursor == 1
			m.draft.Output = ""
			if m.globalOutput {
				m.draft.Output = m.outputPath
			}
			m.err = ""
		}
	}
}

func (m model) outputSummary() string {
	if m.globalOutput {
		return "global: " + m.draft.Output
	}
	return "repo-local"
}

func (m model) settingsLines() []string {
	providers := []string{}
	for _, id := range m.draft.Providers {
		providers = append(providers, providerName(id))
	}
	if m.page == 3 {
		return []string{
			"Select skills to load automatically each session in " + strings.Join(providers, ", ") + " global configuration.",
			"Enabling installs concise-output and its standing instruction. Disabling removes only the instruction.",
			"",
			row(true, check(m.draft.Concise)+"concise-output"),
		}
	}
	lines := []string{
		"Some skills produce Markdown. Repo-local output needs committing or discarding to keep your Git tree clean.",
		"A global path outside your repos is recommended; outputs are separated by repo, branch, and skill.",
		"Global output guidance is saved in " + strings.Join(providers, ", ") + " configuration.",
		"Switching to repo-local leaves existing global artifacts in place.",
		"",
		row(m.cursor == 0, check(!m.globalOutput)+"Local repo"),
		row(m.cursor == 1, check(m.globalOutput)+"A single global path (recommended)"),
	}
	if m.globalOutput {
		path := m.outputPath
		if m.editingPath {
			runes := []rune(path)
			path = string(runes[:m.pathCursor]) + "│" + string(runes[m.pathCursor:])
		}
		lines = append(lines, row(m.cursor == 2, "Path: "+path), "Enter on Path edits · Enter accepts · Esc restores · Ctrl+u clears")
	}
	return lines
}
