package tui

import (
	"fmt"
	"slices"

	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

func (m model) provider() string {
	if slices.Contains(m.draft.Providers, m.activeProvider) {
		return m.activeProvider
	}
	if len(m.draft.Providers) > 0 {
		return m.draft.Providers[0]
	}
	return "codex"
}

// storeProvider saves the editable projection without sharing slice storage.
func (m *model) storeProvider() {
	if m.draft.ByProvider == nil {
		m.draft.ByProvider = map[string]install.ProviderSettings{}
	}
	for _, id := range providerIDs {
		if _, ok := m.draft.ByProvider[id]; !ok {
			m.draft.ByProvider[id] = m.draft.ProviderPreferences(id)
		}
	}
	m.draft.ByProvider[m.provider()] = install.ProviderSettings{
		Requested: slices.Clone(m.draft.Requested), Retained: slices.Clone(m.draft.Retained),
		Concise: m.draft.Concise, FullTemplate: m.draft.FullTemplate, Documentation: m.draft.Documentation, Output: m.draft.Output,
	}
	if m.removedProviders == nil {
		m.removedProviders = map[string][]string{}
	}
	m.removedProviders[m.provider()] = slices.Clone(m.removed)
	if m.outputPaths == nil {
		m.outputPaths = map[string]string{}
	}
	m.outputPaths[m.outputKey()] = m.outputPath
	if m.outputModes == nil {
		m.outputModes = map[string]bool{}
	}
	m.outputModes[m.outputKey()] = m.globalOutput
}

func (m *model) loadProvider() {
	m.activeProvider = m.provider()
	p := m.draft.ForProvider(m.provider())
	m.draft.Requested, m.draft.Retained = p.Requested, p.Retained
	m.draft.Concise, m.draft.FullTemplate = p.Concise, p.FullTemplate
	m.draft.Documentation = p.Documentation
	if !m.draft.SharedStorage {
		m.draft.Output = p.Output
	}
	m.removed = slices.Clone(m.removedProviders[m.provider()])
	m.syncOutput()
}

// focusProvider follows the provider of the visible row being edited.
func (m *model) focusProvider(id string) {
	if id == m.provider() {
		return
	}
	m.storeProvider()
	m.activeProvider = id
	m.loadProvider()
}

// providerView projects a provider's draft without changing the current edit.
func (m model) providerView(id string) model {
	if id == m.provider() {
		return m
	}
	p := m.draft.ForProvider(id)
	m.activeProvider = id
	m.draft.Requested, m.draft.Retained = p.Requested, p.Retained
	m.draft.Concise, m.draft.FullTemplate = p.Concise, p.FullTemplate
	m.draft.Documentation = p.Documentation
	if !m.draft.SharedStorage {
		m.draft.Output = p.Output
	}
	m.removed = m.removedProviders[id]
	m.syncOutput()
	return m
}

func (m model) enabledProviders() []string {
	var ids []string
	for _, id := range providerIDs {
		if slices.Contains(m.draft.Providers, id) {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m model) providerSummary(s settings) []string {
	var lines []string
	if s.SharedStorage {
		lines = append(lines, "Markdown output: "+outputDescription(s.Output), "Layout: "+storageLayout(s))
	}
	for _, id := range s.Providers {
		p := s.ForProvider(id)
		lines = append(lines, fmt.Sprintf("%s: %d requested skills · coding guidance %t · documentation %t · concise replies %t", providerName(id), len(p.Requested), p.FullTemplate, p.DocumentationEnabled(), p.Concise))
		if !s.SharedStorage {
			lines = append(lines, "Markdown output: "+outputDescription(p.Output))
		}
	}
	return lines
}

// installedProviders uses observed content rather than saved or draft selections.
func (m model) installedProviders() []string {
	var ids []string
	for _, id := range providerIDs {
		state := m.status.Guidance[id].State
		if len(m.status.Observed[id]) > 0 || state == "Installed" || state == "Needs review" {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m model) outputKey() string {
	if m.draft.SharedStorage {
		return ""
	}
	return m.provider()
}

// shareStorage reuses an agreed legacy location without choosing between conflicts.
func (m *model) shareStorage() bool {
	if m.draft.SharedStorage {
		return true
	}
	path := m.draft.Output
	if len(m.draft.Providers) > 0 {
		path = m.draft.ForProvider(m.draft.Providers[0]).Output
	}
	for _, id := range m.draft.Providers {
		if m.draft.ForProvider(id).Output != path {
			return false
		}
	}
	m.draft.Output, m.draft.SharedStorage = path, true
	return true
}

func storageLayout(s settings) string {
	path := "<repo-root>/<skill>/<branch>/"
	if s.Output != "" {
		path = s.Output + "/<repo>/<skill>/<branch>/"
	}
	if s.SeparateByProvider {
		path += "<LLM>/"
	}
	return path
}
