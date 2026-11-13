package tui

import (
	"context"
	"fmt"
	"github.com/EimantasZilis/agent-toolkit/internal/install"
	"github.com/charmbracelet/x/term"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"syscall"

	tea "charm.land/bubbletea/v2"
)

const managementPage = -3

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
	applyRequest                install.Request
	quitAfter                   bool
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
	themeIndex                  int
}

type uiTheme struct {
	name                                    string
	accent, heading, selectedFG, selectedBG string
	error, notice, warning, muted           string
}

var uiThemes = []uiTheme{
	{
		name: "Midnight", accent: "#89DDFF", heading: "#C792EA",
		selectedFG: "#F8F8F2", selectedBG: "#34495E", error: "#FF6B6B",
		notice: "#A6E3A1", warning: "#F9C74F", muted: "#7F849C",
	},
	{
		name: "Ocean", accent: "#80CBC4", heading: "#82AAFF",
		selectedFG: "#102A43", selectedBG: "#B2DFDB", error: "#C62828",
		notice: "#2E7D32", warning: "#8D6E00", muted: "#52606D",
	},
	{
		name: "Daylight", accent: "#005F87", heading: "#7F3FBF",
		selectedFG: "#17212B", selectedBG: "#D9E9F2", error: "#B42318",
		notice: "#207547", warning: "#8B5E00", muted: "#5D6873",
	},
}

// Run starts the source TUI with real terminal descriptors. Mutations finish
// or roll back before shutdown, including handled signals and terminal loss.
func Run(root string) error {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return fmt.Errorf("the TUI requires an interactive terminal; run make run from a terminal")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve source root: %w", err)
	}
	homes, err := install.EnvironmentHomes()
	if err != nil {
		return err
	}
	service, err := install.New(abs, homes)
	if err != nil {
		return err
	}
	tracked := &trackedBackend{backend: service}
	defer tracked.close()
	program := tea.NewProgram(newModel(tracked, abs), tea.WithoutSignalHandler())
	signals, done := make(chan os.Signal, 1), make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	defer close(done)
	go func() {
		for {
			select {
			case <-signals:
				program.Send(shutdownMsg{})
			case <-done:
				return
			}
		}
	}()
	final, err := program.Run()
	if err == nil {
		if m, ok := final.(model); ok && m.quitAfter && m.err != "" {
			return fmt.Errorf("%s", m.err)
		}
	}
	return err
}

type shutdownMsg struct{}

func newModel(b backend, root string) model {
	return model{backend: b, root: root, page: -1, width: 80, height: 24, expanded: map[string]bool{}}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		catalog, err := m.backend.Catalog()
		if err != nil {
			return loadedMsg{err: err}
		}
		status, err := m.backend.Status(context.Background())
		return loadedMsg{catalogResponse(catalog), statusResponse(status), status.Installed, err}
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
	case shutdownMsg:
		if m.busy {
			m.quitAfter = true
			return m, nil
		}
		return m, tea.Quit
	case loadedMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.loaded, m.catalog, m.status = true, msg.catalog, msg.status
		m.saved, m.draft = clone(msg.status.Desired), clone(msg.status.Desired)
		m.includeUnavailableSkills()
		if msg.installed {
			m.page = managementPage
		}
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
	if key == "t" || key == "T" {
		step := 1
		if key == "T" {
			step = -1
		}
		m.themeIndex = (m.themeIndex + step + len(uiThemes)) % len(uiThemes)
		m.notice = "Theme: " + uiThemes[m.themeIndex].name
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
			if m.applyRequest.Intent == install.Migrate {
				m.notice = "Migration canceled; linked installations unchanged."
			}
		}
		return m, nil
	}
	if key == "ctrl+c" || key == "q" {
		return m.navigate(-2)
	}
	if key == "esc" {
		if m.page == managementPage {
			return m.navigate(-2)
		}
		if m.page == -1 {
			if m.status.Installed {
				return m.navigate(managementPage)
			}
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
	if m.page == managementPage {
		switch key {
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(len(m.managementActions())-1, m.cursor+1)
		case "enter":
			switch m.cursor {
			case 0:
				m.page, m.cursor = -1, 0
			case 1:
				return m.startAction(install.Refresh)
			case 2:
				return m.startAction(install.Uninstall)
			case 3:
				return m.startAction(install.Migrate)
			}
		}
		return m, nil
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

func (m model) startSave() (tea.Model, tea.Cmd) { return m.startAction(install.Configure) }

func (m model) startAction(intent install.Intent) (tea.Model, tea.Cmd) {
	missing := false
	var err error
	if intent != install.Uninstall && intent != install.Migrate {
		missing, err = m.validate()
	}
	if err != nil {
		m.err = err.Error()
		m.depart = false
		return m, nil
	}
	m.err, m.notice, m.busy = "", "Preparing preview…", true
	m.preview = response{}
	m.createOutput = missing
	m.applyRequest = install.Request{Intent: intent, Desired: clone(m.draft), Removed: append([]string(nil), m.removed...)}
	if intent == install.Migrate {
		m.applyRequest = install.Request{Intent: intent}
	}
	request := m.applyRequest
	return m, func() tea.Msg {
		p, err := m.backend.Preview(context.Background(), request)
		return resultMsg{"preview", previewResponse(p), err}
	}
}

func (m model) result(msg resultMsg) (tea.Model, tea.Cmd) {
	m.busy = false
	if m.quitAfter {
		if msg.err != nil {
			m.err = msg.err.Error()
		}
		return m, tea.Quit
	}
	if msg.err != nil {
		m.err = msg.err.Error()
		m.depart = false
		m.notice = "Draft preserved. Press s to preview again and retry."
		if m.applyRequest.Intent == install.Migrate {
			m.notice = "Migration did not complete. Select Migrate linked installation to preview again."
		}
		return m, nil
	}
	if msg.action == "preview" {
		m.preview = msg.response
		m.notice = "Review the proposed changes before accepting."
		if len(m.preview.Conflicts) > 0 {
			m.err = "Resolve ownership conflicts before applying:"
			for _, conflict := range m.preview.Conflicts {
				m.err += "\n" + conflict.Path
			}
			m.depart = false
			return m, nil
		}
		m.dialogs = m.confirmations()
		return m, nil
	}
	intent := m.applyRequest.Intent
	m.saved, m.draft = clone(msg.response.Desired), clone(msg.response.Desired)
	m.syncOutput()
	m.status = msg.response
	m.removed, m.preview, m.applyRequest = nil, response{}, install.Request{}
	m.notice = "Settings saved. Start a new agent session to load the changes."
	if intent == install.Refresh {
		m.notice = "Installation refreshed. Start a new agent session to load the changes."
	}
	if intent == install.Migrate {
		m.page, m.cursor = managementPage, 0
		m.notice = "Linked installation migrated to owned copies. Start a new agent session to load the changes."
	}
	if intent == install.Uninstall {
		m.wizard, m.page, m.cursor = true, 0, 0
		for _, b := range m.catalog.Bundles {
			if b.Name == "starter" {
				m.draft.Requested = append([]string(nil), b.Skills...)
			}
		}
		m.globalOutput = true
		m.draft.Output = m.outputPath
		m.notice = "Managed installation removed. Start a new agent session. Setup is ready for a new installation."
		return m, nil
	}
	if m.wizard {
		m.wizard = false
		m.page, m.cursor = managementPage, 0
	}
	if m.depart {
		return m.leave()
	}
	return m, nil
}

func (m model) apply() (tea.Model, tea.Cmd) {
	m.busy, m.notice = true, "Applying changes… Please wait for completion or rollback."
	plan := m.preview.Plan
	confirmed := install.Confirmations{Removals: m.preview.Confirmation, CreateOutput: m.createOutput}
	return m, func() tea.Msg {
		missing := false
		var err error
		if plan.Request.Intent != install.Uninstall && plan.Request.Intent != install.Migrate {
			missing, err = m.validateOutput()
		}
		if err != nil {
			return resultMsg{action: "apply", err: err}
		}
		if missing && !m.createOutput {
			return resultMsg{action: "apply", err: fmt.Errorf("output directory disappeared; preview again to confirm creation")}
		}
		status, err := m.backend.Apply(context.Background(), plan, confirmed)
		return resultMsg{"apply", statusResponse(status), err}
	}
}

func (m model) confirmations() []dialog {
	if m.applyRequest.Intent == install.Migrate {
		return []dialog{{"Convert the reviewed links to owned copies with these saved preferences? [y/N]"}}
	}
	if m.applyRequest.Intent == install.Uninstall {
		return []dialog{{"Uninstall all verified managed skills and guidance, and forget saved settings? [y/N]"}}
	}
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
		if m.page == managementPage {
			for i, name := range m.managementActions() {
				lines = append(lines, row(i == m.cursor, name))
			}
			lines = append(lines, "", "Run git pull --ff-only separately before refreshing source changes.")
		} else if m.page < 0 {
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

func (m model) managementActions() []string {
	actions := []string{"Configure", "Refresh installation", "Uninstall"}
	if m.status.Linked {
		actions = append(actions, "Migrate linked installation")
	}
	return actions
}

func (m model) operationLines() []string {
	lines := []string{"Proposed changes:"}
	if m.applyRequest.Intent == install.Migrate {
		saved := m.preview.Desired
		lines = append(lines, "Migration uses the current Development checkout.",
			fmt.Sprintf("Saved providers: %v", saved.Providers),
			fmt.Sprintf("Requested skills: %v", saved.Requested),
			fmt.Sprintf("Retained helpers: %v", saved.Retained),
			"Markdown output: "+outputDescription(saved.Output),
			fmt.Sprintf("Concise loading: %t", saved.Concise),
			fmt.Sprintf("Full guidance: %t", saved.FullTemplate))
		for _, item := range m.preview.Selection {
			if item.Role == "required" {
				lines = append(lines, "Required helper: "+item.Name)
			}
		}
	}
	for _, op := range m.preview.Operations {
		if op.Action != "keep" {
			lines = append(lines, op.Provider+": "+op.Action+" "+op.Name+"\n  "+op.Path)
		}
	}
	return lines
}

func outputDescription(path string) string {
	if path == "" {
		return "Repository output"
	}
	return path
}

func (m model) pageLines() []string {
	if m.page < 2 {
		return m.selectionLines()
	}
	if m.page == 4 {
		return []string{fmt.Sprintf("Providers: %v\nSkills: %v\nOutput: %s\nConcise loading: %t\nFull guidance: %t\nPress s to preview and apply setup.", m.draft.Providers, m.draft.Requested, m.outputSummary(), m.draft.Concise, m.draft.FullTemplate)}
	}
	return m.settingsLines()
}

// includeUnavailableSkills lets setup explicitly forget names removed from source.
// Existing owned directories still require independent ownership verification.
func (m *model) includeUnavailableSkills() {
	known := map[string]bool{}
	for _, s := range m.catalog.Skills {
		known[s.Name] = true
	}
	for _, name := range append(append([]string(nil), m.saved.Requested...), m.saved.Retained...) {
		if !known[name] {
			m.catalog.Skills = append(m.catalog.Skills, skill{Name: name, Category: "unavailable"})
			known[name] = true
		}
	}
}
