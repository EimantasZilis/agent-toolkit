package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func reviewEntry(t *testing.T, p Preview, provider, name, rel string) ContentChange {
	t.Helper()
	for _, change := range p.Changes {
		if change.Provider == provider && change.Name == name && strings.HasSuffix(change.Path, rel) {
			return change
		}
	}
	t.Fatalf("missing %s/%s/%s in %+v", provider, name, rel, p.Changes)
	return ContentChange{}
}

func TestContentReviewFreshRefreshAndRemoval(t *testing.T) {
	s := copiedService(t)
	settings := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"python-standards"}, FullTemplate: true}
	source := filepath.Join(s.Root, ".agents/skills/coding/python-standards")
	write(t, filepath.Join(source, "scripts/mode.sh"), "#!/bin/sh\nexit 0\n")
	write(t, filepath.Join(source, "references/deleted.txt"), "remove me\n")
	write(t, filepath.Join(source, "references/replaced"), "file before directory\n")
	write(t, filepath.Join(source, "references/binary"), "\x00old")
	for _, provider := range settings.Providers {
		_, guidance := s.destinations(provider)
		write(t, guidance, "Private user guidance\n")
	}
	before := installationSnapshot(t, s)
	fresh := previewFor(t, s, Request{Desired: settings})
	if installationSnapshot(t, s) != before {
		t.Fatal("preview wrote installation")
	}
	count := 0
	for _, provider := range settings.Providers {
		files := fresh.payloads[provider+"/python-standards"]
		count += len(files) + 1 // managed guidance
		for rel, file := range files {
			change := reviewEntry(t, fresh, provider, "python-standards", rel)
			if change.Kind != "added" || change.Before != nil || !reflect.DeepEqual(change.After, contentState(file, true)) {
				t.Fatalf("incorrect addition: %+v", change)
			}
		}
	}
	if len(fresh.Changes) != count {
		t.Fatalf("fresh count = %d, want %d", len(fresh.Changes), count)
	}
	applyFor(t, s, fresh)
	if p := previewFor(t, s, Request{Intent: Refresh}); len(p.Changes) != 0 {
		t.Fatalf("no-op changes: %+v", p.Changes)
	}
	installed := installationSnapshot(t, s)
	manifest := filepath.Join(source, "SKILL.md")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	write(t, manifest, string(data)+"\nNew instructions\n")
	write(t, filepath.Join(source, "references/core.md"), "New core\n")
	chmod(t, filepath.Join(source, "references/core.md"), 0600)
	chmod(t, filepath.Join(source, "scripts/mode.sh"), 0755)
	chmod(t, filepath.Join(source, "references"), 0750)
	remove(t, filepath.Join(source, "references/deleted.txt"))
	remove(t, filepath.Join(source, "references/replaced"))
	write(t, filepath.Join(source, "references/replaced/child.txt"), "new child\n")
	write(t, filepath.Join(source, "references/added.txt"), "new resource\n")
	write(t, filepath.Join(source, "references/binary"), "\x00new binary")
	write(t, filepath.Join(source, "references/long.txt"), strings.Repeat("long text\n", 200))
	write(t, filepath.Join(s.Root, ".agents/providers/codex/skills/coding/python-standards/agents/openai.yaml"), "interface:\n  display_name: Updated\n")
	write(t, filepath.Join(s.Root, ".agents/global/instructions.md"), "New managed guidance\n")
	p := previewFor(t, s, Request{Intent: Refresh})
	repeat := previewFor(t, s, Request{Intent: Refresh})
	if !reflect.DeepEqual(p.Changes, repeat.Changes) || p.seal != repeat.seal {
		t.Fatal("non-deterministic review")
	}
	want := map[string]string{"SKILL.md": "modified", "references": "mode", "references/core.md": "modified", "scripts/mode.sh": "mode", "references/deleted.txt": "removed", "references/replaced": "replaced", "references/replaced/child.txt": "added", "references/added.txt": "added", "references/binary": "modified", "references/long.txt": "added"}
	if len(p.Changes) != 2*(len(want)+1)+1 {
		t.Fatalf("unexpected changed paths (%d): %+v", len(p.Changes), p.Changes)
	}
	for _, provider := range settings.Providers {
		for rel, kind := range want {
			change := reviewEntry(t, p, provider, "python-standards", rel)
			if change.Kind != kind {
				t.Fatalf("%s: %+v", rel, change)
			}
		}
		mode := reviewEntry(t, p, provider, "python-standards", "scripts/mode.sh")
		if mode.Before.Mode != 0644 || mode.After.Mode != 0755 || len(mode.Lines) != 0 {
			t.Fatalf("mode review: %+v", mode)
		}
		core := reviewEntry(t, p, provider, "python-standards", "references/core.md")
		if core.After.Mode != 0600 || !strings.Contains(strings.Join(core.Lines, "\n"), "+ New core") {
			t.Fatalf("combined review: %+v", core)
		}
		binary := reviewEntry(t, p, provider, "python-standards", "references/binary")
		if !binary.Binary || binary.Before.Size != 4 || binary.After.Size != 11 || binary.Before.SHA256 == binary.After.SHA256 {
			t.Fatalf("binary summary: %+v", binary)
		}
		long := reviewEntry(t, p, provider, "python-standards", "references/long.txt")
		if !long.Truncated || len(long.Lines) > 100 {
			t.Fatal("unbounded excerpt")
		}
		guidance := reviewEntry(t, p, provider, "guidance", ".md")
		if strings.Contains(strings.Join(guidance.Lines, "\n"), "Private user") {
			t.Fatal("leaked user guidance")
		}
	}
	// Source has changed, but both installed provider trees and saved state are
	// still those captured before preview (the full snapshot includes source).
	for _, home := range []string{s.Homes.Agents, s.Homes.Codex, s.Homes.Claude, filepath.Join(s.Root, "config")} {
		var current strings.Builder
		if err := fingerprint(home, &current); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(installed, current.String()) {
			t.Fatal("preview changed installed content")
		}
	}
	applyFor(t, s, p)
	for _, change := range p.Changes {
		if change.Name == "guidance" {
			continue
		}
		if change.After == nil {
			if exists(change.Path) {
				t.Fatalf("removal not applied: %s", change.Path)
			}
			continue
		}
		info, err := os.Lstat(change.Path)
		if err != nil || uint32(info.Mode().Perm()) != change.After.Mode {
			t.Fatalf("mode differs: %s %v", change.Path, err)
		}
		if !info.IsDir() {
			data, err := os.ReadFile(change.Path)
			if err != nil || digest(data) != change.After.SHA256 {
				t.Fatalf("bytes differ: %s", change.Path)
			}
		}
	}
	// Configure provider removal exposes every removed payload; Uninstall does too.
	settings.Providers = []string{"codex"}
	removed := previewFor(t, s, Request{Desired: settings})
	for _, change := range removed.Changes {
		if change.Provider != "claude" || change.Kind != "removed" {
			t.Fatalf("unexpected removal: %+v", change)
		}
	}
	if len(removed.Changes) == 0 {
		t.Fatal("missing Configure removals")
	}
	applyFor(t, s, removed)
	uninstall := previewFor(t, s, Request{Intent: Uninstall})
	for _, change := range uninstall.Changes {
		if change.Kind != "removed" || change.After != nil {
			t.Fatalf("unexpected uninstall: %+v", change)
		}
	}
}

func TestReviewDetailsAreSealed(t *testing.T) {
	for _, field := range []string{"path", "mode", "text", "binary", "truncation", "collection"} {
		t.Run(field, func(t *testing.T) {
			s := copiedService(t)
			p := previewFor(t, s, Request{Desired: Settings{Providers: []string{"codex"}, Requested: []string{"python-standards"}}})
			switch field {
			case "path":
				p.Changes[0].Path += "forged"
			case "mode":
				p.Changes[0].After.Mode = 0777
			case "text":
				p.Changes[0].Lines = []string{"forged"}
			case "binary":
				p.Changes[0].Binary = !p.Changes[0].Binary
			case "truncation":
				p.Changes[0].Truncated = !p.Changes[0].Truncated
			case "collection":
				p.Changes = nil
			}
			before := installationSnapshot(t, s)
			_, err := s.Apply(context.Background(), p, Confirmations{})
			var diagnostic *Error
			if !errors.As(err, &diagnostic) || diagnostic.Code != "invalid_preview" {
				t.Fatalf("tampered review: %v", err)
			}
			if installationSnapshot(t, s) != before {
				t.Fatal("tampered review caused writes")
			}
		})
	}
}

func TestLegacyContentReview(t *testing.T) {
	s, _ := migrationFixture(t)
	source := filepath.Join(s.Root, ".agents/skills/coding/python-standards/SKILL.md")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	write(t, source, string(data)+"\nChanged rendered manifest\n")
	before := installationSnapshot(t, s)
	p := previewFor(t, s, Request{Intent: Migrate})
	for _, provider := range []string{"codex", "claude"} {
		link := reviewEntry(t, p, provider, "python-standards", "scripts/nested/run.sh")
		if link.Kind != "converted" || link.Before.Type != "symlink" || link.After.Type != "file" || link.After.Mode != 0755 || link.Before.SHA256 != link.After.SHA256 {
			t.Fatalf("link conversion: %+v", link)
		}
		manifest := reviewEntry(t, p, provider, "python-standards", "SKILL.md")
		if !strings.Contains(strings.Join(manifest.Lines, "\n"), "+ Changed rendered manifest") {
			t.Fatalf("manifest review: %+v", manifest)
		}
		empty := reviewEntry(t, p, provider, "python-standards", "empty")
		if empty.After.Type != "directory" || empty.Kind != "added" {
			t.Fatal("missing new copied directory")
		}
	}
	adapter := reviewEntry(t, p, "codex", "python-standards", "agents/openai.yaml")
	if adapter.Kind != "converted" {
		t.Fatal("adapter conversion missing")
	}
	if installationSnapshot(t, s) != before {
		t.Fatal("legacy review wrote files")
	}
}

func TestTextReviewBounds(t *testing.T) {
	lines, truncated := textChanges("same\nold\nend\n", "same\nnew\nend\n")
	if truncated || !reflect.DeepEqual(lines, []string{"- old", "+ new"}) {
		t.Fatalf("excerpt: %q %t", lines, truncated)
	}
	lines, truncated = textChanges("", strings.Repeat("界", 9000))
	if !truncated || len(strings.Join(lines, "")) > 8192 {
		t.Fatal("long line was not bounded")
	}
}

func TestContentReviewTypeReplacementsAndOrdering(t *testing.T) {
	s := copiedService(t)
	source := filepath.Join(s.Root, ".agents/skills/coding/python-standards")
	write(t, filepath.Join(source, "replace/child"), "old\n")
	settings := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"python-standards"}}
	applyFor(t, s, previewFor(t, s, Request{Desired: settings}))
	remove(t, filepath.Join(source, "replace/child"))
	remove(t, filepath.Join(source, "replace"))
	write(t, filepath.Join(source, "replace"), "new\n")
	p := previewFor(t, s, Request{Intent: Refresh})
	if len(p.Changes) != 4 {
		t.Fatalf("replacement paths: %+v", p.Changes)
	}
	for _, provider := range settings.Providers {
		change := reviewEntry(t, p, provider, "python-standards", "replace")
		if change.Kind != "replaced" || change.Before.Type != "directory" || change.After.Type != "file" {
			t.Fatalf("replacement: %+v", change)
		}
		child := reviewEntry(t, p, provider, "python-standards", "replace/child")
		if child.Kind != "removed" {
			t.Fatal("missing child removal")
		}
	}
	for i := 1; i < len(p.Changes); i++ {
		a, b := p.Changes[i-1], p.Changes[i]
		if a.Provider > b.Provider || a.Provider == b.Provider && a.Path >= b.Path {
			t.Fatal("unsorted changes")
		}
	}
	applyFor(t, s, p)
	// A source-only change that does not affect rendering is absent from review.
	write(t, filepath.Join(s.Root, ".agents/global/instructions.md"), "unused template\n")
	if p := previewFor(t, s, Request{Intent: Refresh}); len(p.Changes) != 0 {
		t.Fatal("source-only changes appeared in review")
	}
}

func TestManagedGuidanceReviewPreferences(t *testing.T) {
	s := copiedService(t)
	initial := Settings{Providers: []string{"codex", "claude"}, Concise: true}
	applyFor(t, s, previewFor(t, s, Request{Desired: initial}))
	desired := initial
	desired.FullTemplate = true
	desired.Output = filepath.Join(s.Root, "markdown output")
	p := previewFor(t, s, Request{Desired: desired})
	for _, provider := range desired.Providers {
		change := reviewEntry(t, p, provider, "guidance", ".md")
		if change.Kind != "modified" || change.Before == nil || change.After == nil || len(change.Lines) == 0 {
			t.Fatalf("missing managed template/output diff: %+v", change)
		}
	}
}
