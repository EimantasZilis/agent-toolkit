package tui

import (
	"fmt"
	"slices"
)

type workflowStep struct{ skill, prompt, output string }
type workflow struct {
	name, purpose, input string
	steps                []workflowStep
}

// Keep these curated entry points aligned with docs/usage/workflows.md.
var workflows = []workflow{
	{"Plan a change", "Turn a ticket into a reviewed implementation plan.", "A ticket path or link; repository context if available.", []workflowStep{
		{"ticket-context", "Use ticket-context for <ticket>.", "Repository-backed context; optional when the domain is already clear."},
		{"ticket-to-plan", "Use ticket-to-plan for <ticket>, using <context> as supporting context.", "Planning README and phase files, with the plan-review checkpoint included."},
	}},
	{"Implement a reviewed plan", "Implement bounded work from a reviewed plan on a feature branch.", "A reviewed planning README and clean feature branch.", []workflowStep{
		{"next-commit", "Use next-commit with <planning-readme>.", "One validated unit and updated progress; no commit."},
		{"commit-message", "Use commit-message for my staged changes.", "A commit message after you inspect and stage the unit. Commit separately and repeat."},
		{"next-phase", "Use next-phase with <planning-readme>.", "Alternative to the preceding two steps: validate and locally commit an entire active phase. Never push."},
	}},
	{"Prepare a pull request", "Review completed work and prepare evidence for a pull request.", "A committed feature branch, base branch, and ticket or plan.", []workflowStep{
		{"review-branch", "Use review-branch against main, using <ticket> and <planning-readme>.", "Actionable findings. Resolve, validate, commit, and repeat review before continuing."},
		{"test-cases", "Use test-cases for this branch against main, using <ticket> and <planning-readme>.", "Manual cases. Run them separately and record results; fix and re-review failures."},
		{"pr-description", "Use pr-description for this change, using the ticket, diff, and recorded validation results.", "A copy-ready description. Publish separately."},
	}},
}

func (m model) workflowSkills() []string {
	names := []string{}
	for _, step := range workflows[m.workflowIndex].steps {
		names = add(names, step.skill)
	}
	for changed := true; changed; {
		changed = false
		for _, item := range m.catalog.Skills {
			if !slices.Contains(names, item.Name) {
				continue
			}
			for _, helper := range item.Helpers {
				if !slices.Contains(names, helper) {
					names = add(names, helper)
					changed = true
				}
			}
		}
	}
	return names
}

func (m *model) editWorkflow(key string) {
	ids := m.enabledProviders()
	count := len(workflows) + max(1, len(ids))
	switch key {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(count-1, m.cursor+1)
	case "enter":
		if m.cursor < len(workflows) {
			m.workflowIndex = m.cursor
			m.cursor = len(workflows)
			return
		}
		if m.cursor >= len(workflows) {
			if len(ids) == 0 {
				m.page, m.cursor = 0, 0
				return
			}
			id := ids[m.cursor-len(workflows)]
			names := m.workflowSkills()
			for _, name := range names {
				if !slices.ContainsFunc(m.catalog.Skills, func(s skill) bool { return s.Name == name }) {
					m.err = "Workflow skill unavailable in this checkout: " + name
					return
				}
			}
			m.focusProvider(id)
			for _, step := range workflows[m.workflowIndex].steps {
				name := step.skill
				if m.selected()[name] == "" {
					m.removed = without(m.removed, name)
					m.draft.Requested = add(m.draft.Requested, name)
				}
			}
			m.storeProvider()
			m.notice = "Missing skills staged for " + providerName(id) + ". Press s to review installation changes."
		}
	}
	if m.cursor < len(workflows) {
		m.workflowIndex = m.cursor
	}
}

func (m model) workflowLines() []string {
	lines := []string{"Choose a task and press Enter for its agent actions. Use prompts in your agent.", "This page stages skills only. Press s to review and confirm installation.", ""}
	for i, w := range workflows {
		lines = append(lines, row(m.cursor == i, w.name))
	}
	w := workflows[m.workflowIndex]
	lines = append(lines, "", w.name, w.purpose, "Input: "+w.input, "", "Installation coverage (includes required helpers)")
	names := m.workflowSkills()
	for _, id := range m.enabledProviders() {
		installed, selected := 0, 0
		view := m.providerView(id)
		roles := view.selected()
		for _, name := range names {
			if slices.Contains(m.status.Observed[id], name) {
				installed++
			}
			if roles[name] != "" {
				selected++
			}
		}
		lines = append(lines, fmt.Sprintf("%s: %d/%d installed · %d/%d in draft", providerName(id), installed, len(names), selected, len(names)))
	}
	for i, step := range w.steps {
		lines = append(lines, "", fmt.Sprintf("%d. %s", i+1, step.skill), "Prompt: "+step.prompt, "Output: "+step.output)
	}
	lines = append(lines, "", "Actions")
	ids := m.enabledProviders()
	if len(ids) == 0 {
		lines = append(lines, row(m.cursor == len(workflows), "Enable an agent on Agents"))
	}
	for i, id := range ids {
		lines = append(lines, row(m.cursor == len(workflows)+i, "Add missing skills for "+providerName(id)))
	}
	return lines
}
