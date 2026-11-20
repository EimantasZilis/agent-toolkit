package tui

import (
	"slices"
	"strings"
	"unicode"
)

var filterNames = []string{"All", "Installed", "Selected"}

func (m model) matchesSkill(provider, name string) bool {
	view := m.providerView(provider)
	if m.filters[provider] == 1 && !slices.Contains(m.status.Observed[provider], name) {
		return false
	}
	if m.filters[provider] == 2 && view.selected()[name] == "" {
		return false
	}
	query := strings.ToLower(strings.TrimSpace(m.searches[provider]))
	for _, item := range m.catalog.Skills {
		if item.Name == name {
			return strings.Contains(strings.ToLower(item.Name+" "+item.Description), query)
		}
	}
	return false
}

func (m *model) focusDiscovery(provider string) {
	for i, r := range m.skillRows() {
		if r.provider == provider {
			m.cursor = i
			break
		}
	}
	m.scroll, m.manualScroll = 0, false
}

func (m *model) startDiscovery(key string) {
	rows := m.skillRows()
	if len(rows) == 0 {
		return
	}
	id := rows[min(m.cursor, len(rows)-1)].provider
	if m.searches == nil {
		m.searches = map[string]string{}
	}
	if m.filters == nil {
		m.filters = map[string]int{}
	}
	if key == "f" {
		m.filters[id] = (m.filters[id] + 1) % len(filterNames)
	} else {
		m.editingSearch, m.searchProvider, m.originalSearch = true, id, m.searches[id]
	}
	m.focusDiscovery(id)
}

func (m *model) appendSearch(text string) {
	if strings.ContainsFunc(text, unicode.IsControl) {
		return
	}
	m.searches[m.searchProvider] += text
	m.focusDiscovery(m.searchProvider)
}

func (m model) searchKey(key string) model {
	switch key {
	case "enter":
		m.editingSearch = false
	case "esc", "ctrl+c":
		m.searches[m.searchProvider] = m.originalSearch
		m.editingSearch = false
	case "ctrl+u":
		m.searches[m.searchProvider] = ""
	case "backspace":
		text := []rune(m.searches[m.searchProvider])
		if len(text) > 0 {
			m.searches[m.searchProvider] = string(text[:len(text)-1])
		}
	default:
		if key == "space" {
			key = " "
		}
		if len([]rune(key)) == 1 {
			m.appendSearch(key)
		}
	}
	m.focusDiscovery(m.searchProvider)
	return m
}
