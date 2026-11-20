package install

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
	"github.com/EimantasZilis/agent-toolkit/internal/render"
)

// Status reads saved preferences and observed ownership without writing files.
func (s *Service) Status(ctx context.Context) (Status, error) {
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	desired, applied, installed, err := s.state()
	if err != nil {
		return Status{}, err
	}
	status := Status{Installed: installed, Desired: desired, Applied: applied, Observed: map[string][]string{}, Guidance: map[string]GuidanceStatus{}}
	for _, provider := range []string{"codex", "claude"} {
		names, err := s.observed(provider)
		if err != nil {
			return Status{}, err
		}
		status.Observed[provider] = names
		_, path := s.destinations(provider)
		_, block, count, guideErr := s.guidanceParts(path)
		guide := GuidanceStatus{Path: path, State: "Not installed"}
		if guideErr != nil {
			guide.State = "Needs review"
		} else if count > 0 {
			guide.State = "Installed"
			guide.FullTemplate = strings.Contains(block, "# agent-toolkit guidance") || strings.Contains(block, "[Global guidance](") || strings.Contains(block, "[Coding guidance](")
			guide.Concise = strings.Contains(block, conciseStart) || strings.Contains(block, "[Concise output](")
			guide.Output = strings.Contains(block, "[Markdown output](") || strings.Contains(block, "## Markdown output") || strings.Contains(block, "## Installed Markdown output location")
			guide.Documentation = strings.Contains(block, "[Documentation guidance](") || strings.Contains(block, "## Documentation") || strings.Contains(block, "[Global guidance](")
			guide.DocumentationPath = guidanceTarget(block, "Documentation guidance")
			guide.CodingPath = guidanceTarget(block, "Coding guidance")
			if guide.CodingPath == "" {
				guide.CodingPath = guidanceTarget(block, "Global guidance")
			}
			guide.ConcisePath = guidanceTarget(block, "Concise output")
			guide.OutputPath = guidanceTarget(block, "Markdown output")
			if _, tail, ok := strings.Cut(block, "Markdown artifact location: `"); ok {
				guide.Location, _, _ = strings.Cut(tail, "`")
			}
		}
		status.Guidance[provider] = guide
		root, _ := s.destinations(provider)
		for _, name := range names {
			data, err := os.ReadFile(filepath.Join(root, name, markerName))
			if err != nil {
				return Status{}, err
			}
			if installed && strings.HasPrefix(string(data), s.Root+"\n") {
				status.Linked = true
			}
		}
	}
	status.Revision, err = s.revision()
	if err != nil {
		return Status{}, err
	}
	c, err := s.Catalog()
	if err != nil {
		return Status{}, err
	}
	var selected []string
	for _, provider := range desired.Providers {
		p := desired.ForProvider(provider)
		names := append(p.Requested, p.Retained...)
		if p.Concise {
			names = append(names, "concise-output")
		}
		if len(names) > 0 {
			expanded, selectErr := c.Select(strings.Join(names, " "), false)
			if selectErr == nil {
				names = expanded
			}
		}
		selected = append(selected, names...)
	}
	selected = unique(selected)
	status.Native, err = s.nativeNotice(c, selected)
	return status, err
}

func (s *Service) nativeNotice(c *catalog.Catalog, selected []string) (NativeOverlap, error) {
	notice := NativeOverlap{Status: "unknown", Message: "Native plugins are managed separately; overlapping skills may appear more than once."}
	if s.NativeInventory == "" {
		return notice, nil
	}
	if err := regularOrAbsent(s.NativeInventory); err != nil {
		return notice, err
	}
	data, err := os.ReadFile(s.NativeInventory)
	if err != nil {
		return notice, err
	}
	var exported struct {
		Installed []struct {
			PluginID string `json:"pluginId"`
		} `json:"installed"`
	}
	if err := json.Unmarshal(data, &exported); err != nil {
		return notice, err
	}
	if exported.Installed == nil {
		return notice, nil
	}
	notice.Status = "reported"
	for _, plugin := range exported.Installed {
		name, ok := strings.CutSuffix(plugin.PluginID, "@agent-toolkit")
		if !ok {
			continue
		}
		name = strings.TrimPrefix(name, "bundle-")
		names, err := c.Select(name, false)
		if err != nil {
			continue
		}
		notice.Skills = append(notice.Skills, names...)
	}
	notice.Skills = unique(notice.Skills)
	for _, name := range notice.Skills {
		if contains(selected, name) {
			notice.Overlaps = append(notice.Overlaps, name)
		}
	}
	if len(notice.Overlaps) > 0 {
		notice.Message = "Reported native packages overlap with this selection; manage them in the provider."
	}
	return notice, nil
}

// Preview resolves configure, refresh, migration or uninstall without mutations.
func (s *Service) Preview(ctx context.Context, request Request) (Preview, error) {
	status, err := s.Status(ctx)
	if err != nil {
		return Preview{}, err
	}
	inputs, source, err := s.captureSource()
	if err != nil {
		return Preview{}, err
	}
	snapshot, err := os.MkdirTemp("", "agent-toolkit-source-")
	if err != nil {
		return Preview{}, err
	}
	defer os.RemoveAll(snapshot)
	if err := writePayload(filepath.Join(snapshot, ".agents"), inputs); err != nil {
		return Preview{}, err
	}
	c, err := catalog.Load(snapshot)
	if err != nil {
		return Preview{}, err
	}
	if request.Intent == "" {
		request.Intent = Configure
	}
	desired := request.Desired
	switch request.Intent {
	case Configure:
	case Refresh, Migrate:
		if !status.Installed {
			return Preview{}, problem("setup_required", "", "no saved installation")
		}
		desired = status.Desired
		request.Removed = nil
		request.RemovedByProvider = nil
	case Uninstall:
		desired = Settings{}
		request.Removed = nil
		request.RemovedByProvider = nil
	default:
		return Preview{}, problem("invalid_intent", "", string(request.Intent))
	}
	if err := validSettings(desired); err != nil {
		return Preview{}, err
	}
	desired.Providers = unique(desired.Providers)
	selections := map[string][]string{}
	scoped := map[string]Settings{}
	var selected []string
	if desired.ByProvider != nil {
		copy := map[string]ProviderSettings{}
		for id, preferences := range desired.ByProvider {
			copy[id] = preferences
		}
		desired.ByProvider = copy
	}
	input := desired
	for _, provider := range desired.Providers {
		providerRequest := request
		if request.RemovedByProvider != nil {
			providerRequest.Removed = request.RemovedByProvider[provider]
		}
		preferences, names, err := resolveSelection(c, input.ForProvider(provider), status, providerRequest, provider)
		if err != nil {
			return Preview{}, err
		}
		scoped[provider], selections[provider] = preferences, names
		selected = append(selected, names...)
		if desired.ByProvider != nil {
			desired.ByProvider[provider] = preferences.ProviderPreferences(provider)
		} else {
			desired.Requested, desired.Retained = preferences.Requested, preferences.Retained
		}
	}
	selected = unique(selected)
	var legacyCatalog *catalog.Catalog
	if request.Intent == Migrate {
		if err := s.migrationSelection(status, selections); err != nil {
			return Preview{}, err
		}
		legacyCatalog, err = s.Catalog()
		if err != nil {
			return Preview{}, err
		}
	}
	preview := Preview{Request: request, Desired: desired, Revision: status.Revision, Source: source, payloads: map[string]payload{}}
	preview.guidance = map[string]string{}
	for _, provider := range desired.Providers {
		block, err := s.guidanceBlockFrom(snapshot, provider, scoped[provider])
		if err != nil {
			return Preview{}, err
		}
		preview.guidance[provider] = block
	}
	preview.SelectionByProvider = map[string][]Selection{}
	roles := map[string]string{}
	for _, provider := range desired.Providers {
		preferences := scoped[provider]
		for _, name := range selections[provider] {
			role := "required"
			if contains(preferences.Requested, name) {
				role = "requested"
			} else if preferences.Concise && name == "concise-output" {
				role = "automatic"
			} else if contains(preferences.Retained, name) {
				role = "retained"
			}
			preview.SelectionByProvider[provider] = append(preview.SelectionByProvider[provider], Selection{name, role})
			if roles[name] == "" || role == "requested" {
				roles[name] = role
			}
		}
	}
	for _, name := range selected {
		preview.Selection = append(preview.Selection, Selection{name, roles[name]})
	}
	add := func(op Operation) {
		preview.Operations = append(preview.Operations, op)
		if op.Action == "conflict" {
			preview.Conflicts = append(preview.Conflicts, op)
		}
		if strings.HasPrefix(op.Action, "remove") || op.Action == "strip-guidance" {
			preview.Confirmation = true
		}
	}
	for _, provider := range []string{"codex", "claude"} {
		if err := ctx.Err(); err != nil {
			return Preview{}, err
		}
		selected := selections[provider]
		root, guidance := s.destinations(provider)
		base := s.Homes.Agents
		if provider == "claude" {
			base = s.Homes.Claude
		}
		guidanceBase := s.Homes.Codex
		if provider == "claude" {
			guidanceBase = s.Homes.Claude
		}
		if !realDirectoryPath(root, base) || !realDirectoryPath(filepath.Dir(guidance), guidanceBase) {
			add(Operation{provider, "conflict", "skills", root})
			continue
		}
		if contains(desired.Providers, provider) {
			for _, name := range selected {
				expected, err := skillPayload(c, name, provider, root, s.Root, scoped[provider], selected)
				if err != nil {
					return Preview{}, err
				}
				preview.payloads[provider+"/"+name] = expected
				path := filepath.Join(root, name)
				action := "install"
				if exists(path) {
					action = "conflict"
					if s.safeCopied(path) {
						action = "update"
						actual, err := readTree(path)
						if err != nil {
							return Preview{}, err
						}
						delete(actual, markerName)
						if request.Intent == Migrate || payloadDigest(expected) == payloadDigest(actual) {
							action = "keep"
						}
					} else if request.Intent == Migrate {
						if err := s.validateLegacy(legacyCatalog, provider, name); err != nil {
							return Preview{}, err
						}
						action = "convert"
					}
				}
				add(Operation{provider, action, name, path})
			}
		}
		for _, name := range unique(append(append([]string(nil), status.Observed[provider]...), status.Applied[provider]...)) {
			if contains(desired.Providers, provider) && contains(selected, name) {
				continue
			}
			path := filepath.Join(root, name)
			action := "conflict"
			if s.safeCopied(path) {
				action = "remove"
			}
			if request.Intent == Refresh {
				return Preview{}, problem("setup_required", path, "refresh would remove a skill; review setup")
			}
			add(Operation{provider, action, name, path})
		}
		op, err := s.planGuidance(provider, desired, preview.guidance[provider])
		if err != nil {
			return Preview{}, err
		}
		if op != nil {
			add(*op)
		}
	}
	if request.Intent == Uninstall {
		preview.Confirmation = true
	}
	if request.Intent == Refresh && preview.Confirmation {
		return Preview{}, problem("setup_required", "", "refresh would remove managed content; review setup")
	}
	if request.Intent == Migrate && len(preview.Conflicts) == 0 {
		converts := false
		for _, op := range preview.Operations {
			converts = converts || op.Action == "convert"
		}
		if !converts {
			return Preview{}, problem("already_copied", "", "no linked skills remain; use reviewed refresh for source changes")
		}
	}
	preview.Native, err = s.nativeNotice(c, selected)
	if err != nil {
		return Preview{}, err
	}
	preview.Changes, err = s.contentChanges(preview, legacyCatalog)
	if err != nil {
		return Preview{}, err
	}
	if err := s.checkFailure("after-payload"); err != nil {
		return Preview{}, err
	}
	revision, err := s.revision()
	if err != nil {
		return Preview{}, err
	}
	if revision != status.Revision {
		return Preview{}, problem("stale_revision", "", "source or destination changed during payload preparation")
	}
	preview.seal = previewSignature(preview)
	return preview, nil
}

func previewSignature(p Preview) string {
	data, err := json.Marshal(struct {
		Preview  Preview
		Payloads map[string]payload
		Guidance map[string]string
	}{p, p.payloads, p.guidance})
	if err != nil {
		return ""
	}
	return digest(data)
}

func resolveSelection(c *catalog.Catalog, desired Settings, status Status, request Request, provider string) (Settings, []string, error) {
	var err error
	desired.Requested = unique(desired.Requested)
	desired.Retained = append([]string(nil), desired.Retained...)
	if request.Intent != Uninstall {
		for _, name := range append(append([]string(nil), desired.Requested...), desired.Retained...) {
			if _, err := c.Lookup(name); err != nil {
				return Settings{}, nil, problem("setup_required", "", err.Error())
			}
		}
	}
	if request.Intent != Uninstall && request.Intent != Migrate {
		var required []string
		if len(desired.Requested) > 0 {
			required, err = c.Select(strings.Join(desired.Requested, " "), desired.Concise)
			if err != nil {
				return Settings{}, nil, err
			}
		}
		installed := append([]string(nil), status.Applied[provider]...)
		installed = append(installed, status.Observed[provider]...)
		for _, skill := range c.Skills {
			for _, helper := range skill.Helpers {
				if contains(installed, helper) && !contains(request.Removed, helper) && !contains(required, helper) {
					desired.Retained = append(desired.Retained, helper)
				}
			}
		}
		if status.Desired.ForProvider(provider).Concise && !desired.Concise && contains(installed, "concise-output") && !contains(request.Removed, "concise-output") {
			desired.Retained = append(desired.Retained, "concise-output")
		}
	}
	for _, name := range request.Removed {
		if !validName.MatchString(name) {
			return Settings{}, nil, problem("invalid_settings", "", "invalid removed skill: "+name)
		}
		desired.Requested = without(desired.Requested, name)
		desired.Retained = without(desired.Retained, name)
	}
	desired.Retained = unique(desired.Retained)
	names := append(append([]string(nil), desired.Requested...), desired.Retained...)
	var selected []string
	if len(names) > 0 || desired.Concise {
		if len(names) == 0 {
			names = []string{"concise-output"}
		}
		selected, err = c.Select(strings.Join(names, " "), desired.Concise)
		if err != nil {
			return Settings{}, nil, problem("setup_required", "", err.Error())
		}
	}
	for _, name := range request.Removed {
		if contains(selected, name) {
			return Settings{}, nil, problem("required_helper", "", "remove parents requiring "+name+" first")
		}
	}

	return desired, selected, nil
}

func expectedManifest(c *catalog.Catalog, name, provider, root string, settings Settings, selected []string) (string, error) {
	skill, err := c.Lookup(name)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(skill.Source, "SKILL.md"))
	if err != nil {
		return "", err
	}
	return render.Markdown(c, name, "SKILL.md", string(data), render.Options{Mode: render.Local, Provider: provider, SkillsRoot: root, Selected: selected})
}

func without(names []string, removed string) []string {
	var result []string
	for _, name := range names {
		if name != removed {
			result = append(result, name)
		}
	}
	return result
}

// String describes the action and number of operations for diagnostics.
func (p Preview) String() string {
	return fmt.Sprintf("%s: %d operations (%d conflicts)", p.Request.Intent, len(p.Operations), len(p.Conflicts))
}
