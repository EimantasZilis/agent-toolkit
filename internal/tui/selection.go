package tui

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

var providerIDs = []string{"codex", "claude"}

func providerName(id string) string {
	if id == "codex" {
		return "Codex"
	}
	if id == "claude" {
		return "Claude Code"
	}
	return id
}

func add(names []string, name string) []string {
	if !slices.Contains(names, name) {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func without(names []string, name string) []string {
	return slices.DeleteFunc(names, func(n string) bool { return n == name })
}

func (m model) helperNames() map[string]bool {
	helpers := map[string]bool{"concise-output": true}
	for _, s := range m.catalog.Skills {
		for _, h := range s.Helpers {
			helpers[h] = true
		}
	}
	return helpers
}

func (m model) selected() map[string]string {
	roles := map[string]string{}
	for _, name := range m.draft.Requested {
		roles[name] = "requested"
	}
	for _, name := range m.draft.Retained {
		if !slices.Contains(m.removed, name) && roles[name] == "" {
			roles[name] = "retained"
		}
	}
	helpers := m.helperNames()
	for _, inventory := range []map[string][]string{m.status.Observed, m.status.Applied} {
		for _, names := range [][]string{inventory[m.provider()]} {
			for _, name := range names {
				if helpers[name] && !slices.Contains(m.removed, name) && roles[name] == "" {
					roles[name] = "retained"
				}
			}
		}
	}
	if m.draft.Concise {
		roles["concise-output"] = "automatic"
	}

	// Resolve transitively, retaining a required role even when a helper was
	// also explicitly requested. A parent must be removed before its helper.
	for changed := true; changed; {
		changed = false
		for _, s := range m.catalog.Skills {
			if roles[s.Name] == "" {
				continue
			}
			for _, h := range s.Helpers {
				if roles[h] == "" {
					changed = true
				}
				if roles[h] != "automatic" {
					roles[h] = "required"
				}
			}
		}
	}
	return roles
}

func (m *model) toggleSkill(name string) {
	roles := m.selected()
	if roles[name] == "automatic" {
		m.notice = "Disable Concise replies in Global settings before removing this skill."
		return
	}
	if roles[name] == "required" {
		parents := []string{}
		for _, s := range m.catalog.Skills {
			if roles[s.Name] != "" && slices.Contains(s.Helpers, name) {
				parents = append(parents, s.Name)
			}
		}
		m.notice = "Required by: " + strings.Join(parents, ", ") + ". Remove those skills first."
		return
	}
	if roles[name] != "" {
		m.draft.Requested = without(m.draft.Requested, name)
		m.draft.Retained = without(m.draft.Retained, name)
		m.removed = add(m.removed, name)
	} else {
		m.removed = without(m.removed, name)
		m.draft.Requested = add(m.draft.Requested, name)
	}
	m.notice = "Selection staged; save to apply."
}

func (m *model) toggleGroup(names []string) {
	roles := m.selected()
	all := len(names) > 0
	for _, name := range names {
		all = all && roles[name] != ""
	}
	if !all {
		for _, name := range names {
			m.removed = without(m.removed, name)
			m.draft.Requested = add(m.draft.Requested, name)
		}
		return
	}

	// Batch parent removal first, so a required helper cannot become an
	// explicit removal while it is still needed by a selection outside this group.
	for _, name := range names {
		m.draft.Requested = without(m.draft.Requested, name)
	}
	remaining := m.selected()
	helpers := m.helperNames()
	for _, name := range names {
		if !helpers[name] && remaining[name] == "" {
			m.removed = add(m.removed, name)
		}
	}
	m.notice = "Selection staged; installed helpers are retained."
}

type skillRow struct {
	category, name string
	provider       string
	members        []string
	all            bool
}

func (m model) skillRows() []skillRow {
	var rows []skillRow
	for _, id := range m.enabledProviders() {
		for _, r := range m.providerSkillRows(id) {
			r.provider = id
			rows = append(rows, r)
		}
	}
	return rows
}

func (m model) categoryExpanded(provider, category string) bool {
	expanded, ok := m.expanded[provider+"/"+category]
	if ok {
		return expanded
	}
	return m.expanded[category]
}

func (m model) providerSkillRows(provider string) []skillRow {
	categories := []string{}
	for _, s := range m.catalog.Skills {
		if !slices.Contains(categories, s.Category) {
			categories = append(categories, s.Category)
		}
	}
	sort.Strings(categories)
	rows := []skillRow{}
	if len(m.catalog.Skills) > 0 {
		names := make([]string, 0, len(m.catalog.Skills))
		for _, s := range m.catalog.Skills {
			names = append(names, s.Name)
		}
		rows = append(rows, skillRow{all: true, members: names})
	}
	for _, category := range categories {
		members := []string{}
		for _, s := range m.catalog.Skills {
			if s.Category == category {
				members = append(members, s.Name)
			}
		}
		visible := []string{}
		for _, name := range members {
			if m.matchesSkill(provider, name) {
				visible = append(visible, name)
			}
		}
		if len(visible) == 0 {
			continue
		}
		rows = append(rows, skillRow{category: category, members: members})
		if m.categoryExpanded(provider, category) || m.searches[provider] != "" || m.filters[provider] != 0 {
			for _, name := range visible {
				rows = append(rows, skillRow{name: name})
			}
		}
	}
	return rows
}

func (m *model) editSelection(key string) {
	if m.page == 0 {
		switch key {
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(1, m.cursor+1)
		case "enter":
			id := providerIDs[m.cursor]
			m.storeProvider()
			if slices.Contains(m.draft.Providers, id) {
				m.draft.Providers = without(m.draft.Providers, id)
			} else {
				m.draft.Providers = add(m.draft.Providers, id)
			}
			m.loadProvider()
			m.err = ""
		}
		return
	}
	if m.page != 1 {
		return
	}
	rows := m.skillRows()
	if len(rows) == 0 {
		return
	}
	m.cursor = min(m.cursor, len(rows)-1)
	switch key {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(len(rows)-1, m.cursor+1)
	case "left", "right":
		if rows[m.cursor].category != "" {
			m.expanded[rows[m.cursor].provider+"/"+rows[m.cursor].category] = key == "right"
		}
	case "enter":
		r := rows[m.cursor]
		m.focusProvider(r.provider)
		if r.name != "" {
			m.toggleSkill(r.name)
		} else {
			m.toggleGroup(r.members)
		}
	case "a":
		m.focusProvider(rows[m.cursor].provider)
		for _, r := range rows {
			if r.provider == m.provider() && r.all {
				m.toggleGroup(r.members)
				break
			}
		}
	}
	updated := m.skillRows()
	m.cursor = min(m.cursor, len(updated)-1)
	m.focusProvider(updated[m.cursor].provider)
}

func check(selected bool) string {
	if selected {
		return "[x] "
	}
	return "[ ] "
}

func (m model) selectionLines() []string {
	if m.page == 0 {
		lines := []string{"Select which agents the skills should be installed for.", "Codex: skills and AGENTS.md. Claude Code: skills and CLAUDE.md.", ""}
		if m.wizard {
			lines = append(lines, "Starter selection: concise-output keeps replies focused; commit-message drafts messages for staged changes.", "Adjust the selection on Skills, or open Workflows to choose skills for a task.", "")
		}
		for i, id := range providerIDs {
			lines = append(lines, row(i == m.cursor, check(slices.Contains(m.draft.Providers, id))+providerName(id)))
		}
		return lines
	}
	lines := []string{"Each agent has its own skill selection. Required helpers are selected automatically.", "←/→ collapse/expand category · ↵ toggle bundle/skill · a toggles ALL skills for this agent, including hidden matches", ""}
	if len(m.enabledProviders()) == 0 {
		return append(lines, "Enable an agent on Agents to configure its skills.")
	}
	lastProvider := ""
	for i, r := range m.skillRows() {
		if r.provider != lastProvider {
			lines = append(lines, providerName(r.provider), fmt.Sprintf("Installed: %d skills · changes below are staged until saved.", len(m.status.Observed[r.provider])))
			lines = append(lines, fmt.Sprintf("Search: %s · Filter: %s · / search · f filter", m.searches[r.provider], filterNames[m.filters[r.provider]]))
			if len(m.providerSkillRows(r.provider)) == 1 {
				lines = append(lines, "No matching skills. Press / then Ctrl+u to clear search; f changes filter.")
			}
			lastProvider = r.provider
		}
		provider := m.providerView(r.provider)
		roles := provider.selected()
		text := ""
		if r.name != "" {
			text = "  " + check(roles[r.name] != "") + r.name
			if slices.Contains(m.status.Observed[r.provider], r.name) {
				text += " (installed)"
			}
			if roles[r.name] == "required" || roles[r.name] == "retained" {
				text += " (" + roles[r.name] + ")"
			}
		} else {
			all := true
			for _, name := range r.members {
				all = all && roles[name] != ""
			}
			if r.all {
				text = "Select all skills"
				if all {
					text = "Deselect all skills"
				}
				lines = append(lines, row(i == m.cursor, text))
				continue
			}
			arrow := "▸ "
			if m.categoryExpanded(r.provider, r.category) {
				arrow = "▾ "
			}
			mark := check(all)
			count := 0
			for _, name := range r.members {
				if roles[name] != "" {
					count++
				}
			}
			if count > 0 && !all {
				mark = "[-] "
			}
			text = fmt.Sprintf("%s%s%s  %d / %d", mark, arrow, r.category, count, len(r.members))
		}
		lines = append(lines, row(i == m.cursor, text))
	}
	return lines
}

func (m model) selectionConfirmations() []dialog {
	dialogs := []dialog{}
	providers := []string{}
	for _, id := range m.saved.Providers {
		if !slices.Contains(m.draft.Providers, id) {
			providers = append(providers, providerName(id))
		}
	}
	if len(providers) > 0 {
		dialogs = append(dialogs, dialog{"Are you sure you want to uninstall " + strings.Join(providers, ", ") + " skills and any global settings associated with it? [y/N]"})
	}
	names := []string{}
	for _, op := range m.preview.Operations {
		if op.Action == "remove" && slices.Contains(m.draft.Providers, op.Provider) {
			names = add(names, op.Name)
		}
	}
	if len(names) > 0 {
		dialogs = append(dialogs, dialog{"Are you sure you want to uninstall these skills?\n" + strings.Join(names, "\n") + "\n[y/N]"})
	}
	if m.preview.Confirmation && len(dialogs) == 0 {
		dialogs = append(dialogs, dialog{"Remove the global guidance listed in this preview? [y/N]"})
	}
	return append(dialogs, dialog{"Apply the reviewed changes? [y/N]"})
}
