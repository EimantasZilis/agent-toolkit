package tui

import (
	"context"
	"fmt"
	"github.com/EimantasZilis/agent-toolkit/internal/install"
	"github.com/charmbracelet/x/term"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	tea "charm.land/bubbletea/v2"
)

const managementPage = -3

const unsavedPrompt = "There are unsaved changes in the configuration. Would you like to save them? [S]ave / Discard"

var pageNames = []string{"LLMs", "Skills", "Global settings", "Review setup"}

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
	initial                     *settings
	catalog                     response
	status                      response
	removed                     []string
	activeProvider              string
	removedProviders            map[string][]string
	preview                     response
	applyRequest                install.Request
	quitAfter                   bool
	dialogs                     []dialog
	unsaved                     bool
	refreshPending              bool
	target                      int // -1 menu, -2 quit; other values are wizard pages.
	depart                      bool
	err                         string
	notice                      string
	expanded                    map[string]bool
	globalOutput                bool
	outputPath, originalPath    string
	outputPaths                 map[string]string
	outputModes                 map[string]bool
	outputPathsToCreate         []string
	editingPath                 bool
	pathCursor                  int
	createOutput                bool
	scroll                      int
	manualScroll                bool
	help                        bool
	positions                   [4]int
	reviewDetails               bool
	conflictReview              bool
	errorReview                 bool
	skillDetails                bool
	browseScroll                int
	browseManual                bool
}

type uiTheme struct {
	accent, heading, selectedFG, selectedBG string
	error, notice, warning, muted           string
}

var defaultTheme = uiTheme{
	accent: "#89DDFF", heading: "#89DDFF",
	selectedFG: "#F8F8F2", selectedBG: "#34495E", error: "#FF6B6B",
	notice: "#A6E3A1", warning: "#F9C74F", muted: "#7F849C",
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
	if s.ByProvider != nil {
		profiles := map[string]install.ProviderSettings{}
		for id := range s.ByProvider {
			profiles[id] = s.ProviderPreferences(id)
		}
		s.ByProvider = profiles
	}
	s.Providers = append([]string(nil), s.Providers...)
	s.Requested = append([]string(nil), s.Requested...)
	s.Retained = append([]string(nil), s.Retained...)
	return s
}

func (m model) dirty() bool {
	for id, removed := range m.removedProviders {
		if id == m.provider() {
			removed = m.removed
		}
		for _, name := range removed {
			if slices.Contains(m.status.Applied[id], name) || slices.Contains(m.status.Observed[id], name) {
				return true
			}
		}
	}
	baseline := m.originalSettings()
	if m.draft.SharedStorage {
		for _, id := range m.draft.Providers {
			if baseline.ForProvider(id).Output != m.draft.Output {
				return true
			}
		}
		if m.globalOutput != (m.draft.Output != "") {
			return true
		}
	}
	if baseline.SeparateByProvider != m.draft.SeparateByProvider {
		return true
	}
	for id, mode := range m.outputModes {
		if m.draft.SharedStorage {
			continue
		}
		if id == m.provider() || id == "" {
			mode = m.globalOutput
		}
		if mode != (baseline.ForProvider(id).Output != "") {
			return true
		}
	}
	if m.draft.ByProvider != nil {
		current := clone(m.draft)
		current.ByProvider[m.provider()] = install.ProviderSettings{Requested: current.Requested, Retained: current.Retained, Concise: current.Concise, FullTemplate: current.FullTemplate, Documentation: current.Documentation, Output: current.Output}
		for _, id := range providerIDs {
			a, b := baseline.ForProvider(id), current.ForProvider(id)
			if !sameNames(a.Requested, b.Requested) || !sameNames(a.Retained, b.Retained) || a.Concise != b.Concise || a.FullTemplate != b.FullTemplate || a.DocumentationEnabled() != b.DocumentationEnabled() || (!m.draft.SharedStorage && a.Output != b.Output) {
				return true
			}
		}
		baseline.Requested, baseline.Retained = m.draft.Requested, m.draft.Retained
		baseline.Concise, baseline.FullTemplate = m.draft.Concise, m.draft.FullTemplate
		baseline.Output = m.draft.Output
		baseline.Documentation = m.draft.Documentation
	}
	if !sameNames(baseline.Providers, m.draft.Providers) ||
		!sameNames(baseline.Requested, m.draft.Requested) ||
		!sameNames(baseline.Retained, m.draft.Retained) ||
		baseline.Output != m.draft.Output || baseline.Concise != m.draft.Concise ||
		baseline.FullTemplate != m.draft.FullTemplate || baseline.DocumentationEnabled() != m.draft.DocumentationEnabled() || m.globalOutput != (baseline.Output != "") {
		return true
	}
	for _, inventory := range []map[string][]string{m.status.Observed, m.status.Applied} {
		for _, names := range inventory {
			for _, name := range m.removed {
				if slices.Contains(names, name) {
					return true
				}
			}
		}
	}
	return false
}

func (m model) originalSettings() settings {
	if m.wizard && m.initial != nil {
		return *m.initial
	}
	return m.saved
}

func sameNames(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(slices.Compact(a), slices.Compact(b))
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
		m.loadProvider()
		m.shareStorage()
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
			initial := clone(m.draft)
			m.initial = &initial
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
	browseManual := m.manualScroll
	if m.busy {
		return m, nil
	}
	if m.editingPath {
		return m.editPath(key)
	}
	if key == "esc" {
		key = "backspace"
	}
	if m.skillDetails {
		switch key {
		case "enter", "backspace", "v":
			m.skillDetails = false
			m.scroll, m.manualScroll = m.browseScroll, m.browseManual
		case "down", "j", "up", "k", "pgdown", "pgup":
			step := 1
			if key == "pgdown" || key == "pgup" {
				_, height := m.viewportSize()
				step = max(1, height-1)
			}
			if key == "up" || key == "k" || key == "pgup" {
				step = -step
			}
			m.scroll, m.manualScroll = max(0, m.viewOffset()+step), true
		case "q", "ctrl+c":
			m.skillDetails = false
			return m.navigate(-2)
		}
		return m, nil
	}
	if m.conflictReview || m.errorReview {
		switch key {
		case "up", "k", "down", "j":
			step := 1
			if key == "up" || key == "k" {
				step = -1
			}
			m.scroll = max(0, m.viewOffset()+step)
			m.manualScroll = true
			return m, nil
		case "backspace", "enter", "q", "ctrl+c":
			m.conflictReview, m.errorReview = false, false
			m.err, m.notice = "", "Nothing applied; draft preserved."
			m.scroll, m.manualScroll = 0, false
			if key == "q" || key == "ctrl+c" {
				return m.navigate(-2)
			}
			return m, nil
		default:
			if key != "pgup" && key != "pgdown" {
				return m, nil
			}
		}
	}
	if key == "pgdown" || key == "pgup" {
		if m.unsaved {
			return m, nil
		}
		m.scroll = m.viewOffset()
		_, height := m.viewportSize()
		step := max(1, height-1)
		if key == "pgup" {
			step = -step
		}
		m.scroll = max(0, m.scroll+step)
		m.manualScroll = true
		return m, nil
	}
	if len(m.dialogs) > 0 && m.reviewDetails && (key == "up" || key == "down" || key == "k" || key == "j") {
		step := 1
		if key == "up" || key == "k" {
			step = -1
		}
		m.scroll = max(0, m.viewOffset()+step)
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
	if m.refreshPending {
		if key == "d" || key == "D" {
			m.refreshPending, m.unsaved = false, false
			m.draft = clone(m.saved)
			m.removed, m.removedProviders = nil, nil
			m.outputPaths, m.outputModes = nil, nil
			m.loadProvider()
			return m.startAction(install.Refresh)
		}
		if key == "enter" || key == "backspace" || key == "n" {
			m.refreshPending, m.unsaved = false, false
		}
		return m, nil
	}
	if m.unsaved {
		switch key {
		case "s", "S":
			m.unsaved = false
			if m.wizard {
				m.page, m.cursor, m.depart = 3, 0, false
				return m, nil
			}
			return m.startSave()
		case "d", "D":
			m.unsaved = false
			m.draft = clone(m.originalSettings())
			m.removed = nil
			m.removedProviders = nil
			m.outputPaths = nil
			m.outputModes = nil
			m.loadProvider()
			m.syncOutput()
			return m.leave()
		case "backspace":
			m.unsaved = false
			m.depart = false
		}
		return m, nil
	}
	if len(m.dialogs) > 0 {
		if key == "v" {
			m.reviewDetails = !m.reviewDetails
			m.scroll, m.manualScroll = 0, m.reviewDetails
			return m, nil
		}
		if key == "y" || key == "Y" {
			m.dialogs = m.dialogs[1:]
			m.reviewDetails = false
			if len(m.dialogs) == 0 {
				return m.apply()
			}
		} else if key == "n" || key == "enter" || key == "backspace" {
			m.dialogs = nil
			m.depart = false
			m.notice = "Save canceled; draft preserved."
			if m.applyRequest.Intent == install.Migrate {
				m.notice = "Migration canceled; linked installations unchanged."
			}
		}
		return m, nil
	}
	if key == "?" {
		m.help = !m.help
		return m, nil
	}
	if m.help {
		if key == "backspace" {
			m.help = false
		}
		return m, nil
	}
	if key == "v" && m.page == 1 && m.focusedSkillName() != "" {
		m.skillDetails = true
		m.browseScroll, m.browseManual = m.scroll, browseManual
		m.scroll, m.manualScroll = 0, true
		return m, nil
	}
	if key == "e" && m.err != "" {
		m.errorReview = true
		m.scroll, m.manualScroll = 0, true
		return m, nil
	}
	if key == "s" {
		m.depart = false
		return m.startSave()
	}
	if key == "tab" || key == "shift+tab" || (key >= "1" && key <= "4") {
		pages := []int{managementPage, 0, 1, 2}
		index := 0
		for i, page := range pages {
			if m.page == page {
				index = i
			}
		}
		next := (index + 1) % len(pages)
		if key == "shift+tab" {
			next = (index + len(pages) - 1) % len(pages)
		}
		if len(key) == 1 {
			next = int(key[0] - '1')
		}
		if m.wizard && m.page == 0 && next > 1 && len(m.draft.Providers) == 0 {
			m.err = "Select at least one LLM before continuing."
			return m, nil
		}
		if m.wizard && m.page == 2 && next != 3 {
			if _, err := m.missingOutputs(); err != nil {
				m.err = err.Error()
				return m, nil
			}
		}
		m.positions[index] = m.cursor
		m.page, m.cursor = pages[next], m.positions[next]
		m.scroll, m.manualScroll, m.err = 0, false, ""
		return m, nil
	}
	if key == "space" || key == " " {
		key = "enter"
	}
	if key == "ctrl+c" || key == "q" {
		return m.navigate(-2)
	}
	if key == "backspace" {
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
				m.page, m.cursor = 0, 0
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
			m.cursor = min(2, m.cursor+1)
		case "enter":
			m.page, m.cursor = m.cursor, 0
		}
		return m, nil
	}
	return m.edit(key)
}

func (m model) navigate(target int) (tea.Model, tea.Cmd) {
	if m.wizard && m.page == 0 && target > 0 && len(m.draft.Providers) == 0 {
		m.err = "Select at least one LLM before continuing."
		return m, nil
	}
	if m.wizard && m.page == 2 && target > 2 {
		if _, err := m.missingOutputs(); err != nil {
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
	if intent == install.Refresh && m.dirty() {
		m.refreshPending, m.unsaved = true, true
		return m, nil
	}
	missing := false
	var err error
	checked := m
	if intent == install.Refresh {
		checked.draft = clone(m.saved)
		checked.outputModes, checked.outputPaths = nil, nil
		checked.loadProvider()
	}
	if intent != install.Uninstall && intent != install.Migrate {
		missing, err = checked.validate()
	}
	if err != nil {
		m.err = err.Error()
		m.depart = false
		return m, nil
	}
	if intent == install.Configure {
		if !m.shareStorage() {
			m.err = "Choose one shared storage location on Global settings before saving."
			return m, nil
		}
		m.storeProvider()
	}
	m.err, m.notice, m.busy = "", "Preparing preview…", true
	m.preview = response{}
	m.reviewDetails = false
	m.createOutput = missing
	m.outputPathsToCreate = nil
	if missing {
		m.outputPathsToCreate, err = checked.missingOutputs()
		if err != nil {
			m.busy = false
			m.err = err.Error()
			return m, nil
		}
	}
	m.applyRequest = install.Request{Intent: intent, Desired: clone(m.draft), RemovedByProvider: m.removedProviders}
	if intent == install.Refresh {
		m.applyRequest.Desired = clone(m.saved)
	}
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
		if m.applyRequest.Intent == install.Refresh {
			m.notice = "Refresh did not complete. Select Refresh installation to preview again."
		}
		if m.applyRequest.Intent == install.Migrate {
			m.notice = "Migration did not complete. Select Migrate linked installation to preview again."
		}
		return m, nil
	}
	if msg.action == "preview" {
		m.preview = msg.response
		m.reviewDetails = false
		m.notice = "Review the proposed changes before accepting."
		if len(m.preview.Conflicts) > 0 {
			m.err = "Resolve ownership conflicts before applying:"
			m.dialogs = nil
			m.conflictReview = true
			m.scroll, m.manualScroll = 0, true
			for _, conflict := range m.preview.Conflicts {
				m.err += "\n\n" + terminalText(providerName(conflict.Provider)+" / "+conflict.Name) + "\n" + terminalText(conflict.Path) + "\nCause: " + terminalText(conflict.Reason) + "\nRecovery: " + terminalText(conflict.Recovery)
			}
			m.depart = false
			return m, nil
		}
		m.dialogs = m.confirmations()
		return m, nil
	}
	intent := m.applyRequest.Intent
	m.saved, m.draft = clone(msg.response.Desired), clone(msg.response.Desired)
	m.initial = nil
	m.outputPaths = nil
	m.outputModes = nil
	m.removedProviders = nil
	m.loadProvider()
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
		initial := clone(m.draft)
		m.initial = &initial
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
		if plan.Request.Intent != install.Uninstall && plan.Request.Intent != install.Migrate {
			checked := m
			checked.draft = clone(plan.Desired)
			checked.outputModes, checked.outputPaths = nil, nil
			checked.loadProvider()
			paths, err := checked.missingOutputs()
			if err != nil {
				return resultMsg{action: "apply", err: err}
			}
			for _, path := range paths {
				if !slices.Contains(m.outputPathsToCreate, path) {
					return resultMsg{action: "apply", err: fmt.Errorf("output directory disappeared; preview again to confirm creation")}
				}
			}
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
		dialogs = append([]dialog{{"Create the Markdown output directories?\n" + strings.Join(m.outputPathsToCreate, "\n") + "\n[y/N]"}}, dialogs...)
	}
	return dialogs
}

func (m model) validate() (bool, error) {
	if m.wizard && len(m.draft.Providers) == 0 {
		return false, fmt.Errorf("select at least one LLM on Page 1")
	}
	paths, err := m.missingOutputs()
	return len(paths) > 0, err
}

func (m model) edit(key string) (tea.Model, tea.Cmd) {
	m.editSelection(key)
	m.editSettings(key)
	return m, nil
}

func (m model) viewLines() []string {
	if m.skillDetails {
		return []string{m.focusedSkillName(), "Enter/Esc or v returns; arrows and Page Up/Page Down scroll.", "", m.focusedSkillDescription()}
	}
	if m.errorReview {
		return []string{"Operation error", "Enter/Esc returns; Up/Down or Page Up/Page Down scroll.", "", m.err, "", m.notice}
	}
	if m.conflictReview {
		lines := []string{"Installation blocked. Nothing applied.", "Esc, ⌫ or ↵ returns; Up/Down or Page Up/Page Down scroll.", ""}
		if m.status.Linked && m.applyRequest.Intent != install.Migrate {
			lines = append(lines, "Legacy linked skills require migration before configure, refresh, or uninstall.", "Return to the management menu and choose Migrate linked installation.", "")
		}
		return append(lines, m.err)
	}
	if len(m.dialogs) > 0 {
		lines := m.reviewSummaryLines()
		if m.reviewDetails {
			lines = m.operationLines()
		}
		context := strings.Split(m.dialogs[0].text, "\n")[1:]
		for _, line := range context {
			if line != "[y/N]" {
				lines = append(lines, line)
			}
		}
		return lines
	}
	lines := []string{}
	if !m.loaded {
		lines = append(lines, "Loading backend…", m.err, "r retry · q quit")
	} else {
		if m.page == managementPage {
			return m.overviewLines()
		} else if m.page < 0 {
			for i, name := range pageNames[:3] {
				lines = append(lines, row(i == m.cursor, name))
			}
		} else {
			lines = append(lines, pageNames[m.page], "")
			lines = append(lines, m.pageLines()...)
		}
		for _, message := range []string{m.status.Native.Message, m.notice, m.err} {
			if message != "" {
				lines = append(lines, "", message)
			}
		}
	}
	return lines
}

func guidanceStatus(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
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
			fmt.Sprintf("Saved LLMs: %v", saved.Providers),
			fmt.Sprintf("Requested skills: %v", saved.Requested),
			fmt.Sprintf("Retained helpers: %v", saved.Retained),
			"Markdown output: "+outputDescription(saved.Output),
			fmt.Sprintf("Concise loading: %t", saved.Concise),
			fmt.Sprintf("Full guidance: %t", saved.FullTemplate))
		if saved.ByProvider != nil {
			lines = append(lines, m.providerSummary(saved)...)
		}
		for _, item := range m.preview.Selection {
			if item.Role == "required" {
				lines = append(lines, "Required helper: "+item.Name)
			}
		}
	}
	for _, op := range m.preview.Operations {
		if op.Action != "keep" {
			lines = append(lines, terminalText(op.Provider+": "+op.Action+" "+op.Name), "  "+terminalText(op.Path))
		}
	}
	// Saved preferences can contain untrusted paths as well as resource text.
	for i := range lines {
		lines[i] = terminalText(lines[i])
	}
	lines = append(lines, contentReviewLines(m.preview.Changes)...)
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
	if m.page == 3 {
		draft := clone(m.draft)
		if draft.ByProvider != nil {
			draft.ByProvider[m.provider()] = install.ProviderSettings{Requested: draft.Requested, Retained: draft.Retained, Concise: draft.Concise, FullTemplate: draft.FullTemplate, Documentation: draft.Documentation, Output: draft.Output}
		}
		return append(m.providerSummary(draft), "Press s to preview and apply setup.")
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
	var names []string
	for _, id := range providerIDs {
		p := m.saved.ForProvider(id)
		names = append(names, p.Requested...)
		names = append(names, p.Retained...)
	}
	for _, name := range names {
		if !known[name] {
			m.catalog.Skills = append(m.catalog.Skills, skill{Name: name, Category: "unavailable"})
			known[name] = true
		}
	}
}
