package tui

import (
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
		for _, names := range inventory {
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
		m.notice = "Disable concise-output automatic loading on Page 4 before removing this skill."
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
	members        []string
}

func (m model) skillRows() []skillRow {
	categories := []string{}
	for _, s := range m.catalog.Skills {
		if !slices.Contains(categories, s.Category) {
			categories = append(categories, s.Category)
		}
	}
	sort.Strings(categories)
	rows := []skillRow{}
	for _, category := range categories {
		members := []string{}
		for _, s := range m.catalog.Skills {
			if s.Category == category {
				members = append(members, s.Name)
			}
		}
		rows = append(rows, skillRow{category: category, members: members})
		if m.expanded[category] {
			for _, name := range members {
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
		case "space":
			id := providerIDs[m.cursor]
			if slices.Contains(m.draft.Providers, id) {
				m.draft.Providers = without(m.draft.Providers, id)
			} else {
				m.draft.Providers = add(m.draft.Providers, id)
			}
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
			m.expanded[rows[m.cursor].category] = key == "right"
		}
	case "space":
		r := rows[m.cursor]
		if r.name != "" {
			m.toggleSkill(r.name)
		} else {
			m.toggleGroup(r.members)
		}
	case "a":
		names := []string{}
		for _, s := range m.catalog.Skills {
			names = append(names, s.Name)
		}
		m.toggleGroup(names)
	}
}

func check(selected bool) string {
	if selected {
		return "[x] "
	}
	return "[ ] "
}

func (m model) selectionLines() []string {
	if m.page == 0 {
		lines := []string{"Select which LLM providers the skills should be installed for.", "Codex: skills and AGENTS.md. Claude Code: skills and CLAUDE.md.", ""}
		for i, id := range providerIDs {
			lines = append(lines, row(i == m.cursor, check(slices.Contains(m.draft.Providers, id))+providerName(id)))
		}
		return lines
	}
	lines := []string{"Select skills to install. Required helpers are selected automatically.", "←/→ collapse/expand category · space toggle bundle/skill · a toggle all", ""}
	roles := m.selected()
	for i, r := range m.skillRows() {
		text := ""
		if r.name != "" {
			text = "  " + check(roles[r.name] != "") + r.name
			if roles[r.name] != "" {
				text += " (" + roles[r.name] + ")"
			}
		} else {
			all := true
			for _, name := range r.members {
				all = all && roles[name] != ""
			}
			arrow := "▸ "
			if m.expanded[r.category] {
				arrow = "▾ "
			}
			text = check(all) + arrow + r.category
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
