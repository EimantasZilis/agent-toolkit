package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderSettingsIsolationAndRefresh(t *testing.T) {
	s := testService(t)
	settings := Settings{Providers: []string{"codex", "claude"}, ByProvider: map[string]ProviderSettings{
		"codex":  {Requested: []string{"ticket-to-plan"}, Concise: true, FullTemplate: true},
		"claude": {Requested: []string{"commit-message"}},
	}}
	status := applyFor(t, s, previewFor(t, s, Request{Desired: settings}))
	if !contains(status.Observed["codex"], "plan-review") || contains(status.Observed["claude"], "plan-review") {
		t.Fatal("dependencies crossed providers", status.Observed)
	}
	_, codexGuide := s.destinations("codex")
	_, claudeGuide := s.destinations("claude")
	if !exists(codexGuide) || !exists(claudeGuide) || status.Guidance["claude"].Concise || status.Guidance["claude"].FullTemplate || !status.Guidance["claude"].Output {
		t.Fatal("guidance crossed providers")
	}
	before, err := os.ReadFile(codexGuide)
	if err != nil {
		t.Fatal(err)
	}

	settings.ByProvider["claude"] = ProviderSettings{Requested: []string{"commit-message"}, Concise: true, FullTemplate: true}
	applyFor(t, s, previewFor(t, s, Request{Desired: settings}))
	settings.ByProvider["claude"] = ProviderSettings{}
	p := previewFor(t, s, Request{Desired: settings, RemovedByProvider: map[string][]string{"claude": {"concise-output"}}})
	for _, op := range p.Operations {
		if op.Provider == "codex" && op.Action != "keep" {
			t.Fatalf("Claude edit changes Codex: %+v", op)
		}
	}
	status = applyFor(t, s, p)
	if len(status.Observed["claude"]) != 0 || !contains(status.Observed["codex"], "concise-output") {
		t.Fatal(status.Observed)
	}
	after, err := os.ReadFile(codexGuide)
	if err != nil || string(before) != string(after) {
		t.Fatal("Codex guidance changed", err)
	}
	applyFor(t, s, previewFor(t, s, Request{Intent: Refresh}))
}

func TestLegacyPreferencesUpgradeOnApply(t *testing.T) {
	s := testService(t)
	legacy := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"commit-message"}, Concise: true, FullTemplate: true}
	applyFor(t, s, previewFor(t, s, Request{Desired: legacy}))
	path := filepath.Join(s.Root, "config/settings.env")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	status, err := s.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	upgraded := status.Desired
	upgraded.ByProvider = map[string]ProviderSettings{}
	for _, id := range legacy.Providers {
		upgraded.ByProvider[id] = legacy.ProviderPreferences(id)
	}
	p := previewFor(t, s, Request{Desired: upgraded})
	for _, op := range p.Operations {
		if op.Action != "keep" {
			t.Fatalf("upgrade changed behavior: %+v", op)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(before) {
		t.Fatal("preview wrote migration")
	}
	status = applyFor(t, s, p)
	data, err = os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "STATE_VERSION=3") {
		t.Fatal("settings not upgraded", err)
	}
	for _, id := range legacy.Providers {
		if !status.Desired.ForProvider(id).Concise {
			t.Fatal("lost guidance", id)
		}
	}
}

// A failed multi-provider apply restores both installations and saved profiles.
func TestProviderSettingsRollbackAndRemoval(t *testing.T) {
	s := testService(t)
	settings := Settings{Providers: []string{"codex", "claude"}, ByProvider: map[string]ProviderSettings{
		"codex":  {Requested: []string{"commit-message"}, Concise: true},
		"claude": {Requested: []string{"ticket-to-plan"}, FullTemplate: true},
	}}
	status := applyFor(t, s, previewFor(t, s, Request{Desired: settings}))
	settings.ByProvider["codex"] = ProviderSettings{Requested: []string{"python-standards"}}
	settings.ByProvider["claude"] = ProviderSettings{Requested: []string{"commit-message"}}
	plan := previewFor(t, s, Request{Desired: settings})
	injected := errors.New("injected provider failure")
	s.fail = func(point string) error {
		if point == "after-provider:claude" {
			return injected
		}
		return nil
	}
	if _, err := s.Apply(context.Background(), plan, Confirmations{Removals: true}); !errors.Is(err, injected) {
		t.Fatalf("apply = %v", err)
	}
	s.fail = nil
	restored, err := s.Status(context.Background())
	if err != nil || restored.Revision != status.Revision {
		t.Fatal("rollback changed state", err)
	}
	desired := restored.Desired
	desired.Providers = []string{"codex"}
	plan = previewFor(t, s, Request{Desired: desired})
	for _, op := range plan.Operations {
		if op.Provider == "codex" && op.Action != "keep" {
			t.Fatalf("provider removal changed Codex: %+v", op)
		}
	}
	status = applyFor(t, s, plan)
	if len(status.Observed["claude"]) != 0 || !contains(status.Observed["codex"], "commit-message") {
		t.Fatal(status.Observed)
	}
}
