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
	m.globalOutput = m.draft.Output != "" || m.outputModes[m.outputKey()]
	m.outputPath = m.draft.Output
	if m.outputPath == "" {
		if cached, ok := m.outputPaths[m.outputKey()]; ok {
			m.outputPath = cached
		} else if home, err := os.UserHomeDir(); err == nil {
			m.outputPath = filepath.Join(home, "agent-toolkit-output")
		}
	}
	m.editingPath = false
}

// validateOutput only inspects the filesystem. Creation belongs to the installation transaction.
func (m model) validateOutput() (bool, error) {
	if !m.globalOutput {
		return false, nil
	}
	path := m.draft.Output
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) || strings.ContainsFunc(path, unicode.IsControl) {
		return false, fmt.Errorf("Global settings: enter an absolute, single-line global output path")
	}

	missing := false
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Stat(current)
		if os.IsNotExist(err) {
			// A dangling symlink is not a directory we can create.
			if _, linkErr := os.Lstat(current); linkErr == nil {
				return false, fmt.Errorf("Global settings: output path contains a dangling link: %s", current)
			}
			missing = true
			continue
		}
		if err != nil {
			return false, fmt.Errorf("Global settings: inspect output path: %w", err)
		}
		if !info.IsDir() {
			return false, fmt.Errorf("Global settings: output path is not a directory: %s", current)
		}
		// Access checks write and search permissions without creating a probe file.
		// Supported release targets are macOS and Linux.
		if err := syscall.Access(current, 3); err != nil {
			return false, fmt.Errorf("Global settings: output directory is not writable/searchable: %w", err)
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
	if m.page != 2 {
		return
	}
	rows := m.globalSettingsRows()
	if len(rows) == 0 {
		return
	}
	m.cursor = min(m.cursor, len(rows)-1)
	switch key {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(len(rows)-1, m.cursor+1)
	case "enter":
		r := rows[m.cursor]

		switch r.kind {
		case "guidance":
			m.focusProvider(r.provider)
			documentation := m.draft.DocumentationEnabled()
			m.draft.Documentation = &documentation
			m.draft.FullTemplate = !m.draft.FullTemplate
		case "documentation":
			m.focusProvider(r.provider)
			enabled := !m.draft.DocumentationEnabled()
			m.draft.Documentation = &enabled
		case "concise":
			m.focusProvider(r.provider)
			m.draft.Concise = !m.draft.Concise

		case "split":
			if m.shareStorage() {
				m.draft.SeparateByProvider = !m.draft.SeparateByProvider
			} else {
				m.err = "Choose one shared location before enabling LLM separation."
			}
		case "path":
			m.draft.SharedStorage = true
			m.editingPath = true
			m.originalPath = m.outputPath
			m.pathCursor = len([]rune(m.outputPath))
		case "local", "external":
			m.draft.SharedStorage = true
			m.globalOutput = r.kind == "external"
			m.draft.Output = ""
			if m.globalOutput {
				m.draft.Output = m.outputPath
			}
			m.err = ""
		}
	}

}

func (m model) guidanceEnabled() bool {
	return m.draft.Concise || m.draft.FullTemplate || m.draft.DocumentationEnabled()
}

func (m model) settingsLines() []string {
	lines := []string{
		"Artifact storage",
		"Applies to all LLMs.",
		"",
	}
	if !m.draft.SharedStorage {
		lines = append(lines, "Legacy storage: choose Local repo or External path to save one shared location.")
		for _, id := range m.enabledProviders() {
			lines = append(lines, providerName(id)+": "+outputDescription(m.draft.ForProvider(id).Output))
		}
	}

	for i, r := range m.storageRows() {
		switch r.kind {
		case "local":
			lines = append(lines, row(i == m.cursor, check(!m.globalOutput)+"Local repo"))
		case "external":
			lines = append(lines, row(i == m.cursor, check(m.globalOutput)+"External path (recommended)"))
		case "split":
			lines = append(lines, "", row(i == m.cursor, check(m.draft.SeparateByProvider)+"Separate by LLM"), "Layout: "+storageLayout(m.draft), "Changing location leaves existing artifacts in place.")
		case "path":
			path := m.outputPath
			if m.editingPath {
				runes := []rune(path)
				path = string(runes[:m.pathCursor]) + "│" + string(runes[m.pathCursor:])
			}
			lines = append(lines, row(i == m.cursor, "    Path: "+path), "      ↵ on Path edits · ↵ accepts · Esc restores · Ctrl+u clears")
		}
	}
	lines = append(lines, "", "Agent guidance", "Applies across repositories; configure each LLM separately.", "")
	for _, id := range m.enabledProviders() {
		provider := m.providerView(id)
		lines = append(lines, "  "+providerName(id))
		guide := m.status.Guidance[id]
		state := guide.State
		if state == "" {
			state = "Not installed"
		}
		lines = append(lines, "  Installed guidance: "+state)
		if guide.Path != "" {
			lines = append(lines, "  File: "+guide.Path)
		}
		for i, r := range m.globalSettingsRows() {
			if r.provider != id {
				continue
			}
			var label, target string
			switch r.kind {
			case "guidance":
				target = guide.CodingPath
				label = check(provider.draft.FullTemplate) + "Coding guidance — Follow coding standards and conventions."
			case "documentation":
				target = guide.DocumentationPath
				label = check(provider.draft.DocumentationEnabled()) + "Documentation guidance — Write clear, structured documentation."
			case "concise":
				target = guide.ConcisePath
				label = check(provider.draft.Concise) + "Concise replies — Apply concise-output to every prompt."
			}
			lines = append(lines, row(i == m.cursor, "  "+label))
			if target != "" {
				lines = append(lines, "    Installed target: "+target)
			}
		}
		if guide.OutputPath != "" {
			lines = append(lines, "  Markdown output instructions", "    Installed target: "+guide.OutputPath)
		}
		lines = append(lines, "")
	}
	if len(m.enabledProviders()) == 0 {
		lines = append(lines, "Enable an LLM on the LLMs tab to configure guidance.")
	}
	if len(m.enabledProviders()) > 0 {
		lines = append(lines, "Turning off Concise replies keeps its skill installed.")
	}
	return lines
}

func (m model) focusedSetting(kind string) bool {
	rows := m.globalSettingsRows()
	return m.cursor >= 0 && m.cursor < len(rows) && rows[m.cursor].kind == kind
}

func (m model) globalSettingsRows() []storageRow {
	rows := m.storageRows()
	for _, id := range m.enabledProviders() {
		rows = append(rows, storageRow{id, "guidance"}, storageRow{id, "documentation"}, storageRow{id, "concise"})
	}
	return rows
}

type storageRow struct{ provider, kind string }

func (m model) storageRows() []storageRow {
	rows := []storageRow{{"", "local"}, {"", "external"}}
	if m.globalOutput {
		rows = append(rows, storageRow{"", "path"})
	}
	rows = append(rows, storageRow{"", "split"})
	return rows
}

// missingOutputs validates every enabled provider before save, without writing.
func (m model) missingOutputs() ([]string, error) {
	if m.draft.SharedStorage {
		missing, err := m.validateOutput()
		if err != nil {
			return nil, err
		}
		if missing {
			return []string{m.draft.Output}, nil
		}
		return nil, nil
	}
	var paths []string
	for _, id := range m.enabledProviders() {
		provider := m.providerView(id)
		missing, err := provider.validateOutput()
		if err != nil {
			return nil, fmt.Errorf("%s storage: %w", providerName(id), err)
		}
		if missing && !containsPath(paths, provider.draft.Output) {
			paths = append(paths, provider.draft.Output)
		}
	}
	return paths, nil
}

func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}
