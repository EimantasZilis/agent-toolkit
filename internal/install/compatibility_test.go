package install

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReportedNativeOverlapIncludesRequiredHelpers(t *testing.T) {
	s := testService(t)
	s.NativeInventory = "testdata/codex-inventory.json"
	p := previewFor(t, s, Request{Intent: Configure, Desired: Settings{Providers: []string{"codex"}, Requested: []string{"ticket-to-plan"}}})
	want := []string{"plan-review", "ticket-to-plan"}
	if !reflect.DeepEqual(p.Native.Overlaps, want) {
		t.Fatalf("preview overlap = %v", p.Native)
	}
	status := applyFor(t, s, p)
	if !reflect.DeepEqual(status.Native.Overlaps, want) {
		t.Fatalf("saved overlap = %v", status.Native)
	}
}

func TestRefreshPreservesPreferencesAndChangedSource(t *testing.T) {
	s := testService(t)
	settings := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"ticket-to-plan"}, Output: filepath.Join(t.TempDir(), `artifacts "quotes" $(touch NEVER); &`), Concise: true, FullTemplate: true}
	applyFor(t, s, previewFor(t, s, Request{Intent: Configure, Desired: settings}))
	path := filepath.Join(s.Root, ".agents/skills/scoping/ticket-to-plan/SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, string(data)+"\nRefreshed source\n")
	p := previewFor(t, s, Request{Intent: Refresh})
	status := applyFor(t, s, p)
	if string(settingsBytes(status.Desired)) != string(settingsBytes(settings)) {
		t.Fatalf("refresh lost preferences: %+v", status.Desired)
	}
	for _, provider := range []string{"codex", "claude"} {
		root, _ := s.destinations(provider)
		data, err := os.ReadFile(filepath.Join(root, "ticket-to-plan/SKILL.md"))
		if err != nil || !strings.Contains(string(data), "Refreshed source") {
			t.Fatalf("%s not refreshed: %v", provider, err)
		}
	}
	if exists(filepath.Join(s.Root, "NEVER")) {
		t.Fatal("evaluated literal output path")
	}
}

func TestRefreshWithoutStateDoesNotInferPreferences(t *testing.T) {
	s := testService(t)
	settings := Settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}}
	installedFixture(t, s, "codex", "commit-message", settings, settings.Requested)
	if _, err := s.Preview(context.Background(), Request{Intent: Refresh}); err == nil {
		t.Fatal("refresh inferred settings from observed files")
	}
	if exists(filepath.Join(s.Root, "config")) {
		t.Fatal("read-only refresh wrote state")
	}
}

func TestInventoryCannotClaimUserContent(t *testing.T) {
	s := testService(t)
	root, _ := s.destinations("codex")
	path := filepath.Join(root, "commit-message/SKILL.md")
	write(t, path, "User skill\n")
	saveFixture(t, s, Settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}}, map[string][]string{"codex": {"commit-message"}, "claude": {}})
	status, err := s.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Observed["codex"]) != 0 {
		t.Fatal("inventory granted ownership")
	}
	p, err := s.Preview(context.Background(), Request{Intent: Configure, Desired: status.Desired})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Conflicts) != 1 || p.Conflicts[0].Path != filepath.Dir(path) {
		t.Fatal(p)
	}
	if _, err := s.Apply(context.Background(), p, Confirmations{Removals: true}); err == nil {
		t.Fatal("overwrote user skill")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "User skill\n" {
		t.Fatal("user skill changed")
	}
}

func TestDuplicateGuidanceCollapsesAndEditsConflict(t *testing.T) {
	s := testService(t)
	settings := Settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}, Concise: true}
	_, path := s.destinations("codex")
	write(t, path, "User instructions\n")
	applyFor(t, s, previewFor(t, s, Request{Intent: Configure, Desired: settings}))
	block, err := guidanceBlock(s.Root, settings)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, string(data)+block)
	applyFor(t, s, previewFor(t, s, Request{Intent: Configure, Desired: settings}))
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), blockStart) != 1 {
		t.Fatal("duplicate managed block survived")
	}
	write(t, path, string(data)+"\n"+conciseStart+"\nUser-edited loading instructions\n"+conciseEnd+"\n")
	p, err := s.Preview(context.Background(), Request{Intent: Uninstall})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Conflicts) != 1 || p.Conflicts[0].Path != path {
		t.Fatal("edited guidance did not conflict")
	}
}
