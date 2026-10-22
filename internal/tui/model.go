package tui

import (
	"fmt"
	"path/filepath"
	"reflect"

	tea "charm.land/bubbletea/v2"
)

const unsavedPrompt = "There are unsaved changes on the page. Would you like to save them? [S]ave / Discard"

var pageNames = []string{"LLM providers", "Skills", "Markdown output", "Global settings", "Review setup"}

type loadedMsg struct {
	catalog, status response
	installed       bool
	err             error
}
type resultMsg struct {
	action   string
	response response
	err      error
}
type dialog struct{ text string }

type model struct {
	backend                     backend
	root                        string
	loaded, wizard, busy        bool
	page, cursor, width, height int
	saved, draft                settings
	catalog                     response
	status                      response
	removed                     []string
	preview                     response
	applyArgs                   []string
	dialogs                     []dialog
	unsaved                     bool
	target                      int // -1 menu, -2 quit; other values are wizard pages.
	depart                      bool
	err                         string
	notice                      string
	expanded                    map[string]bool
	globalOutput                bool
	outputPath, originalPath    string
	editingPath                 bool
	pathCursor                  int
	createOutput                bool
	scroll                      int
	manualScroll                bool
}

// Run starts the TUI. root must contain the toolkit's scripts and source skills.
// Backend mutations finish before the UI accepts quit, allowing Bash rollback.
func Run(root string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve source root: %w", err)
	}
	m := newModel(shellBackend{abs}, abs)
	_, err = tea.NewProgram(m).Run()
	return err
}

func newModel(b backend, root string) model {
	return model{backend: b, root: root, page: -1, width: 80, height: 24, expanded: map[string]bool{}}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		catalog, err := m.backend.run("catalog", nil)
		if err != nil {
			return loadedMsg{err: err}
		}
		status, err := m.backend.run("status", nil)
		installed, statErr := savedInstallation(m.root)
		return loadedMsg{catalog, status, installed, firstError(err, statErr)}
	}
}

func clone(s settings) settings {
	s.Providers = append([]string(nil), s.Providers...)
	s.Requested = append([]string(nil), s.Requested...)
	s.Retained = append([]string(nil), s.Retained...)
	return s
}

func (m model) dirty() bool {
	return !reflect.DeepEqual(m.saved, m.draft) || len(m.removed) > 0 || m.globalOutput != (m.saved.Output != "")
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.loaded, m.catalog, m.status = true, msg.catalog, msg.status
		m.saved, m.draft = clone(msg.status.Desired), clone(msg.status.Desired)
		m.wizard = !msg.installed && len(m.saved.Providers) == 0
		if m.wizard {
			m.page = 0
			for _, b := range m.catalog.Bundles {
				if b.Name == "starter" {
					m.draft.Requested = append([]string(nil), b.Skills...)
				}
			}
		}
		m.syncOutput()
		if m.wizard {
			m.globalOutput = true
			m.draft.Output = m.outputPath
		}
		return m, nil
	case resultMsg:
		return m.result(msg)
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		if m.editingPath && !m.busy && msg.Text != "" {
			m.insertPath(msg.Text)
			return m, nil
		}
		return m.key(msg.String())
	case tea.PasteMsg:
		if m.editingPath && !m.busy {
			m.insertPath(msg.Content)
		}
	}
	return m, nil
}

func (m model) key(key string) (tea.Model, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	if m.editingPath {
		return m.editPath(key)
	}
	if key == "pgdown" || key == "pgup" {
		m.scroll = m.viewOffset()
		step := max(1, m.height-4)
		if key == "pgup" {
			step = -step
		}
		m.scroll = max(0, m.scroll+step)
		m.manualScroll = true
		return m, nil
	}
	m.manualScroll = false
	if !m.loaded {
		if key == "q" || key == "ctrl+c" {
			return m, tea.Quit
		}
		if key == "r" {
			m.err = ""
			m.busy = true
			return m, m.Init()
		}
		return m, nil
	}
	if m.unsaved {
		switch key {
		case "s", "S":
			m.unsaved = false
			if m.wizard {
				m.page, m.cursor, m.depart = 4, 0, false
				return m, nil
			}
			return m.startSave()
		case "d", "D":
			m.unsaved = false
			m.draft = clone(m.saved)
			m.removed = nil
			m.syncOutput()
			return m.leave()
		case "esc":
			m.unsaved = false
			m.depart = false
		}
		return m, nil
	}
	if len(m.dialogs) > 0 {
		if key == "y" || key == "Y" {
			m.dialogs = m.dialogs[1:]
			if len(m.dialogs) == 0 {
				return m.apply()
			}
		} else if key == "n" || key == "enter" || key == "esc" {
			m.dialogs = nil
			m.depart = false
			m.notice = "Save canceled; draft preserved."
		}
		return m, nil
	}
	if key == "ctrl+c" || key == "q" {
		return m.navigate(-2)
	}
	if key == "esc" {
		if m.page == -1 {
			return m.navigate(-2)
		}
		if m.wizard && m.page > 0 {
			return m.navigate(m.page - 1)
		}
		if m.wizard {
			return m.navigate(-2)
		}
		return m.navigate(-1)
	}
	if m.page == -1 {
		switch key {
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(3, m.cursor+1)
		case "enter":
			m.page, m.cursor = m.cursor, 0
		}
		return m, nil
	}
	if key == "s" {
		if m.wizard && m.page < 4 {
			return m.navigate(m.page + 1)
		}
		m.depart = false
		return m.startSave()
	}
	if key == "tab" || key == "enter" {
		if key == "enter" && m.page == 2 && m.globalOutput && m.cursor == 2 {
			m.editingPath = true
			m.originalPath = m.outputPath
			m.pathCursor = len([]rune(m.outputPath))
			return m, nil
		}
		if m.wizard && m.page < 4 {
			return m.navigate(m.page + 1)
		}
	}
	return m.edit(key)
}

func (m model) navigate(target int) (tea.Model, tea.Cmd) {
	if m.wizard && m.page == 0 && target > 0 && len(m.draft.Providers) == 0 {
		m.err = "Select at least one LLM provider before continuing."
		return m, nil
	}
	if m.wizard && m.page == 2 && target > 2 {
		if _, err := m.validateOutput(); err != nil {
			m.err = err.Error()
			return m, nil
		}
	}
	m.err = ""
	m.target, m.depart = target, true
	if m.dirty() && (!m.wizard || target < 0) {
		m.unsaved = true
		return m, nil
	}
	return m.leave()
}

func (m model) leave() (tea.Model, tea.Cmd) {
	m.depart = false
	if m.target == -2 {
		return m, tea.Quit
	}
	m.page, m.cursor = m.target, 0
	return m, nil
}

func (m model) startSave() (tea.Model, tea.Cmd) {
	missing, err := m.validate()
	if err != nil {
		m.err = err.Error()
		m.depart = false
		return m, nil
	}
	m.err, m.notice, m.busy = "", "Preparing preview…", true
	m.createOutput = missing
	m.applyArgs = m.draft.arguments(m.removed)
	if m.createOutput {
		m.applyArgs = append(m.applyArgs, "--create-output-directory")
	}
	args := append([]string(nil), m.applyArgs...)
	return m, func() tea.Msg { r, err := m.backend.run("preview", args); return resultMsg{"preview", r, err} }
}

func (m model) result(msg resultMsg) (tea.Model, tea.Cmd) {
	m.busy = false
	if msg.err != nil {
		m.err = msg.err.Error()
		m.depart = false
		m.notice = "Draft preserved. Press s to preview again and retry."
		return m, nil
	}
	if msg.action == "preview" {
		m.preview = msg.response
		if len(m.preview.Conflicts) > 0 {
			m.err = "Resolve ownership conflicts before saving."
			m.depart = false
			return m, nil
		}
		m.dialogs = m.confirmations()
		return m, nil
	}
	m.saved, m.draft = clone(msg.response.Desired), clone(msg.response.Desired)
	m.syncOutput()
	m.status = msg.response
	m.removed, m.preview, m.applyArgs = nil, response{}, nil
	m.notice = "Settings saved."
	if m.wizard {
		m.wizard = false
		m.page, m.cursor = -1, 0
	}
	if m.depart {
		return m.leave()
	}
	return m, nil
}

func (m model) apply() (tea.Model, tea.Cmd) {
	m.busy, m.notice = true, "Applying changes… Please wait for completion or rollback."
	args := append([]string(nil), m.applyArgs...)
	args = append(args, "--revision", m.preview.Revision)
	if m.preview.Confirmation {
		args = append(args, "--confirm-removals")
	}
	return m, func() tea.Msg {
		missing, err := m.validateOutput()
		if err != nil {
			return resultMsg{action: "apply", err: err}
		}
		if missing && !m.createOutput {
			return resultMsg{action: "apply", err: fmt.Errorf("output directory disappeared; preview again to confirm creation")}
		}
		r, err := m.backend.run("apply", args)
		return resultMsg{"apply", r, err}
	}
}

func (m model) confirmations() []dialog {
	dialogs := m.selectionConfirmations()
	if m.createOutput {
		dialogs = append([]dialog{{"Create the Markdown output directory?\n" + m.draft.Output + "\n[y/N]"}}, dialogs...)
	}
	return dialogs
}

func (m model) validate() (bool, error) {
	if m.wizard && len(m.draft.Providers) == 0 {
		return false, fmt.Errorf("select at least one LLM provider on Page 1")
	}
	return m.validateOutput()
}

func (m model) edit(key string) (tea.Model, tea.Cmd) {
	m.editSelection(key)
	m.editSettings(key)
	return m, nil
}

func (m model) viewLines() []string {
	lines := []string{}
	if !m.loaded {
		lines = append(lines, "Loading backend…", m.err, "r retry · q quit")
	} else {
		if m.page < 0 {
			for i, name := range pageNames[:4] {
				lines = append(lines, row(i == m.cursor, name))
			}
		} else {
			lines = append(lines, fmt.Sprintf("%d. %s", m.page+1, pageNames[m.page]), "")
			lines = append(lines, m.pageLines()...)
		}
		for _, message := range []string{m.status.Native.Message, m.notice, m.err} {
			if message != "" {
				lines = append(lines, "", message)
			}
		}
		if m.unsaved {
			lines = append(lines, "", unsavedPrompt, "Esc cancels")
		}
		if len(m.dialogs) > 0 {
			lines = append(lines, "")
			lines = append(lines, m.operationLines()...)
			lines = append(lines, m.dialogs[0].text, "Enter/No/Esc cancels; y confirms")
		}
	}
	return lines
}

func row(focused bool, text string) string {
	if focused {
		return "> " + text
	}
	return "  " + text
}

func (m model) operationLines() []string {
	lines := []string{"Proposed changes:"}
	for _, op := range m.preview.Operations {
		if op.Action != "keep" {
			lines = append(lines, op.Provider+": "+op.Action+" "+op.Name)
		}
	}
	return lines
}

func (m model) pageLines() []string {
	if m.page < 2 {
		return m.selectionLines()
	}
	if m.page == 4 {
		return []string{fmt.Sprintf("Providers: %v\nSkills: %v\nOutput: %s\nConcise loading: %t\nPress s to preview and apply setup.", m.draft.Providers, m.draft.Requested, m.outputSummary(), m.draft.Concise)}
	}
	return m.settingsLines()
}
