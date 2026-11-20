package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderStorageRenderingAndIsolation(t *testing.T) {
	s := testService(t)
	codexOutput := filepath.Join(s.Root, "codex output")
	claudeOutput := filepath.Join(s.Root, "claude output")
	settings := Settings{Providers: []string{"codex", "claude"}, ByProvider: map[string]ProviderSettings{
		"codex":  {Requested: []string{"ticket-to-plan"}, Output: codexOutput, FullTemplate: true},
		"claude": {Requested: []string{"ticket-to-plan"}, Output: claudeOutput, FullTemplate: true},
	}}
	plan := previewFor(t, s, Request{Desired: settings})
	if exists(codexOutput) || exists(claudeOutput) {
		t.Fatal("preview created output directories")
	}
	status, err := s.Apply(context.Background(), plan, Confirmations{CreateOutput: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range settings.Providers {
		own, other := codexOutput, claudeOutput
		if id == "claude" {
			own, other = other, own
		}
		root, guide := s.destinations(id)
		data, err := os.ReadFile(guide)
		if err != nil || !strings.Contains(string(data), own) || strings.Contains(string(data), other) {
			t.Fatalf("%s has wrong output: %v\n%s", guide, err, data)
		}
		data, err = os.ReadFile(filepath.Join(root, "ticket-to-plan/SKILL.md"))
		if err != nil || !strings.Contains(string(data), s.Root+"/.agents/global/markdown-output.md") || strings.Contains(string(data), own) || strings.Contains(string(data), other) {
			t.Fatalf("skill must route storage through global settings: %v\n%s", err, data)
		}
		if !exists(own) || status.Desired.ForProvider(id).Output != own {
			t.Fatal("storage not saved", id)
		}
	}
	artifact := filepath.Join(claudeOutput, "keep.md")
	write(t, artifact, "keep")
	settings.ByProvider["claude"] = ProviderSettings{Requested: []string{"ticket-to-plan"}, FullTemplate: true}
	plan = previewFor(t, s, Request{Desired: settings})
	for _, op := range plan.Operations {
		if op.Provider == "codex" && op.Action != "keep" {
			t.Fatalf("Claude storage changed Codex: %+v", op)
		}
	}
	status = applyFor(t, s, plan)
	if status.Desired.ForProvider("claude").Output != "" || !exists(artifact) {
		t.Fatal("repo-local change moved artifacts")
	}
	applyFor(t, s, previewFor(t, s, Request{Intent: Refresh}))
}

func TestSharedStorageUpgradePreservesLocations(t *testing.T) {
	for _, version := range []string{"1", "2"} {
		t.Run(version, func(t *testing.T) {
			s := testService(t)
			output := filepath.Join(s.Root, "shared artifacts")
			legacy := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"ticket-to-plan"}, Output: output}
			applyFor(t, s, previewFor(t, s, Request{Desired: legacy}))
			if version == "2" {
				legacy.ByProvider = map[string]ProviderSettings{}
				for _, id := range legacy.Providers {
					legacy.ByProvider[id] = legacy.ProviderPreferences(id)
				}
				data := strings.Replace(string(settingsBytes(legacy)), "STATE_VERSION=3", "STATE_VERSION=2", 1)
				write(t, filepath.Join(s.Root, "config/settings.env"), data)
				status, err := s.Status(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				inv, err := inventoryBytes([]byte(data), status.Applied)
				if err != nil {
					t.Fatal(err)
				}
				write(t, filepath.Join(s.Root, "config/inventory.json"), string(inv))
			}
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
				upgraded.ByProvider[id] = status.Desired.ProviderPreferences(id)
			}
			plan := previewFor(t, s, Request{Desired: upgraded})
			if len(plan.Changes) != 0 {
				t.Fatalf("migration changed installed contents: %+v", plan.Changes)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(before) != string(after) {
				t.Fatal("read or preview rewrote settings")
			}
			status = applyFor(t, s, plan)
			for _, id := range legacy.Providers {
				if status.Desired.ForProvider(id).Output != output {
					t.Fatal("lost shared output", id)
				}
			}
		})
	}
}

func TestProviderOutputDirectoryRollback(t *testing.T) {
	s := testService(t)
	initial := Settings{Providers: []string{"codex", "claude"}, ByProvider: map[string]ProviderSettings{"codex": {}, "claude": {}}}
	before := applyFor(t, s, previewFor(t, s, Request{Desired: initial}))
	for _, id := range initial.Providers {
		initial.ByProvider[id] = ProviderSettings{Output: filepath.Join(s.Root, "new outputs", id)}
	}
	plan := previewFor(t, s, Request{Desired: initial})
	injected := errors.New("output rollback")
	s.fail = func(point string) error {
		if point == "after-settings" {
			return injected
		}
		return nil
	}
	if _, err := s.Apply(context.Background(), plan, Confirmations{CreateOutput: true}); !errors.Is(err, injected) {
		t.Fatal(err)
	}
	s.fail = nil
	status, err := s.Status(context.Background())
	if err != nil || status.Revision != before.Revision || exists(filepath.Join(s.Root, "new outputs")) {
		t.Fatal("rollback retained output changes", err)
	}
}
