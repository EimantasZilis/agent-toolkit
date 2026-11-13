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
	status := Status{Installed: installed, Desired: desired, Applied: applied, Observed: map[string][]string{}}
	for _, provider := range []string{"codex", "claude"} {
		names, err := s.observed(provider)
		if err != nil {
			return Status{}, err
		}
		status.Observed[provider] = names
	}
	status.Revision, err = s.revision()
	if err != nil {
		return Status{}, err
	}
	c, err := s.Catalog()
	if err != nil {
		return Status{}, err
	}
	selected := append(append([]string(nil), desired.Requested...), desired.Retained...)
	if desired.Concise {
		selected = append(selected, "concise-output")
	}
	if len(selected) > 0 {
		// Status still reports saved preferences if a source skill disappeared;
		// refresh/setup diagnoses that separately before applying anything.
		expanded, selectErr := c.Select(strings.Join(selected, " "), false)
		if selectErr == nil {
			selected = expanded
		}
	}
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
	case Uninstall:
		desired = Settings{}
		request.Removed = nil
	default:
		return Preview{}, problem("invalid_intent", "", string(request.Intent))
	}
	if err := validSettings(desired); err != nil {
		return Preview{}, err
	}
	desired.Providers = unique(desired.Providers)
	desired.Requested = unique(desired.Requested)
	desired.Retained = append([]string(nil), desired.Retained...)
	if request.Intent != Uninstall {
		for _, name := range append(append([]string(nil), desired.Requested...), desired.Retained...) {
			if _, err := c.Lookup(name); err != nil {
				return Preview{}, problem("setup_required", "", err.Error())
			}
		}
	}
	if request.Intent != Uninstall && request.Intent != Migrate {
		var required []string
		if len(desired.Requested) > 0 {
			required, err = c.Select(strings.Join(desired.Requested, " "), desired.Concise)
			if err != nil {
				return Preview{}, err
			}
		}
		installed := append(append([]string(nil), status.Applied["codex"]...), status.Applied["claude"]...)
		installed = append(installed, status.Observed["codex"]...)
		installed = append(installed, status.Observed["claude"]...)
		for _, skill := range c.Skills {
			for _, helper := range skill.Helpers {
				if contains(installed, helper) && !contains(request.Removed, helper) && !contains(required, helper) {
					desired.Retained = append(desired.Retained, helper)
				}
			}
		}
		if status.Desired.Concise && !desired.Concise && contains(installed, "concise-output") && !contains(request.Removed, "concise-output") {
			desired.Retained = append(desired.Retained, "concise-output")
		}
	}
	for _, name := range request.Removed {
		if !validName.MatchString(name) {
			return Preview{}, problem("invalid_settings", "", "invalid removed skill: "+name)
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
			return Preview{}, problem("setup_required", "", err.Error())
		}
	}
	for _, name := range request.Removed {
		if contains(selected, name) {
			return Preview{}, problem("required_helper", "", "remove parents requiring "+name+" first")
		}
	}
	var legacyCatalog *catalog.Catalog
	if request.Intent == Migrate {
		if err := s.migrationSelection(status, selected); err != nil {
			return Preview{}, err
		}
		legacyCatalog, err = s.Catalog()
		if err != nil {
			return Preview{}, err
		}
	}
	preview := Preview{Request: request, Desired: desired, Revision: status.Revision, Source: source, payloads: map[string]payload{}}
	preview.guidance, err = guidanceBlock(snapshot, desired)
	if err != nil {
		return Preview{}, err
	}
	for _, name := range selected {
		role := "required"
		if contains(desired.Requested, name) {
			role = "requested"
		} else if desired.Concise && name == "concise-output" {
			role = "automatic"
		} else if contains(desired.Retained, name) {
			role = "retained"
		}
		preview.Selection = append(preview.Selection, Selection{name, role})
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
				expected, err := skillPayload(c, name, provider, root, desired, selected)
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
		op, err := s.planGuidance(provider, desired, preview.guidance)
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
	preview.Native, err = s.nativeNotice(c, selected)
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
		Guidance string
	}{p, p.payloads, p.guidance})
	if err != nil {
		return ""
	}
	return digest(data)
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
	return render.Markdown(c, name, "SKILL.md", string(data), render.Options{Mode: render.Local, Provider: provider, SkillsRoot: root, OutputRoot: settings.Output, Selected: selected})
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
