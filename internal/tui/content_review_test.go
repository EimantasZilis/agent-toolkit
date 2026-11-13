package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

func previewAction(t *testing.T, m model, intent install.Intent) model {
	t.Helper()
	next, cmd := m.startAction(intent)
	m = next.(model)
	if cmd == nil {
		t.Fatalf("preview did not start: %s", m.err)
	}
	m, _ = complete(m, cmd)
	if m.err != "" || len(m.dialogs) == 0 {
		t.Fatalf("preview failed: %s", m.err)
	}
	if !reflect.DeepEqual(m.preview.Changes, m.preview.Plan.Changes) {
		t.Fatal("projection lost typed review")
	}
	return m
}

func TestDevelopmentReviewRepeatedWorkingTreeLoop(t *testing.T) {
	for _, providers := range [][]string{{"codex"}, {"claude"}, {"codex", "claude"}} {
		t.Run(strings.Join(providers, "+"), func(t *testing.T) {
			m, base := backendFixture(t)
			m.draft = settings{Providers: providers, Requested: []string{"python-standards", "ticket-to-plan"}, Concise: true, FullTemplate: true, Output: filepath.Join(base, "custom output")}
			m.syncOutput()
			source := filepath.Join(m.root, ".agents/skills/coding/python-standards")
			manifest := filepath.Join(source, "SKILL.md")
			original, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatal(err)
			}
			helper := filepath.Join(source, "scripts/safe.sh")
			writeMigrationFile(t, helper, []byte("#!/bin/sh\nexit 0\n"))
			writeMigrationFile(t, filepath.Join(source, "references/deleted.txt"), []byte("remove later\n"))
			for generation := 0; generation < 3; generation++ {
				before := migrationSnapshot(t, base)
				marker := fmt.Sprintf("Working tree generation %d", generation)
				writeMigrationFile(t, manifest, append(append([]byte(nil), original...), []byte("\n"+marker+"\n")...))
				resource := filepath.Join(source, "references/current.txt")
				writeMigrationFile(t, resource, []byte(marker+" resource\n"))
				if generation == 1 {
					if err := os.Chmod(helper, 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.Remove(filepath.Join(source, "references/deleted.txt")); err != nil {
						t.Fatal(err)
					}
					writeMigrationFile(t, filepath.Join(source, "references/added.txt"), []byte("new resource\n"))
					writeMigrationFile(t, filepath.Join(m.root, ".agents/providers/codex/skills/coding/python-standards/agents/openai.yaml"), []byte("interface:\n  display_name: Changed\n"))
					guidance := filepath.Join(m.root, ".agents/global/instructions.md")
					data, err := os.ReadFile(guidance)
					if err != nil {
						t.Fatal(err)
					}
					writeMigrationFile(t, guidance, append(data, []byte("\nChanged managed guidance\n")...))
				}
				if generation > 0 {
					for _, home := range []string{filepath.Join(base, "agents"), filepath.Join(base, "codex"), filepath.Join(base, "claude"), filepath.Join(m.root, "config")} {
						if _, err := os.Stat(home); os.IsNotExist(err) {
							continue
						} else if err != nil {
							t.Fatal(err)
						}
						// Source changes must not alter any installed home or saved state.
						assertSnapshotContains(t, before, migrationSnapshot(t, home))
					}
				}
				intent := install.Refresh
				if generation == 0 {
					intent = install.Configure
				}
				saved := clone(m.saved)
				m = previewAction(t, m, intent)
				review := strings.Join(m.operationLines(), "\n")
				for _, text := range []string{marker, "SKILL.md", "references/current.txt", "Content changes"} {
					if !strings.Contains(review, text) {
						t.Fatalf("review missing %q", text)
					}
				}
				if generation == 1 {
					for _, text := range []string{"scripts/safe.sh", "file mode 0644 -> file mode 0755", "removed", "references/deleted.txt", "references/added.txt", "Changed managed guidance"} {
						if !strings.Contains(review, text) {
							t.Fatalf("refresh missing %q", text)
						}
					}
					if strings.Contains(strings.Join(providers, ","), "codex") && !strings.Contains(review, "agents/openai.yaml") {
						t.Fatal("adapter missing from review")
					}
					for _, cancel := range []string{"enter", "n", "esc"} {
						snapshot := migrationSnapshot(t, base)
						next, cmd := press(m, cancel)
						m = next
						if cmd != nil || len(m.dialogs) != 0 || migrationSnapshot(t, base) != snapshot {
							t.Fatal("cancel mutated installation")
						}
						m = previewAction(t, m, intent)
					}
				}
				m = finishDialogs(t, m)
				if generation > 0 && !reflect.DeepEqual(m.saved, saved) {
					t.Fatal("refresh changed saved preferences")
				}
				for _, provider := range providers {
					root := filepath.Join(base, "agents/skills/agent-toolkit/python-standards")
					if provider == "claude" {
						root = filepath.Join(base, "claude/skills/python-standards")
					}
					for _, rel := range []string{"SKILL.md", "references/current.txt"} {
						data, err := os.ReadFile(filepath.Join(root, rel))
						if err != nil || !strings.Contains(string(data), marker) {
							t.Fatalf("installed marker missing: %s %v", rel, err)
						}
					}
					info, err := os.Stat(filepath.Join(root, "scripts/safe.sh"))
					mode := os.FileMode(0644)
					if generation > 0 {
						mode = 0755
					}
					if err != nil || info.Mode().Perm() != mode {
						t.Fatal("reviewed helper mode not applied")
					}
				}
			}
			m = previewAction(t, m, install.Refresh)
			if !strings.Contains(strings.Join(m.operationLines(), "\n"), "Installed content: unchanged.") {
				t.Fatal("no-op review not labelled")
			}
		})
	}
}

func assertSnapshotContains(t *testing.T, whole, part string) {
	t.Helper()
	if !strings.Contains(whole, part) {
		t.Fatal("source edit changed installed content or saved state")
	}
}

func TestStaleReviewRequiresCorrectAction(t *testing.T) {
	for _, change := range []string{"source", "destination", "settings", "inventory"} {
		t.Run(change, func(t *testing.T) {
			m, base := backendFixture(t)
			m.draft = settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}}
			m = saveFixture(t, m)
			m = previewAction(t, m, install.Refresh)
			switch change {
			case "source":
				writeMigrationFile(t, filepath.Join(m.root, ".agents/skills/productivity/commit-message/new.txt"), []byte("new\n"))
			case "destination":
				writeMigrationFile(t, filepath.Join(base, "agents/skills/agent-toolkit/commit-message/user.txt"), []byte("user\n"))
			case "settings", "inventory":
				path := filepath.Join(m.root, "config/settings.env")
				if change == "inventory" {
					path = filepath.Join(m.root, "config/inventory.json")
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				writeMigrationFile(t, path, append(data, '\n'))
			}
			before := migrationSnapshot(t, base)
			m, cmd := press(m, "y")
			if cmd == nil {
				t.Fatal("no apply command")
			}
			m, followup := complete(m, cmd)
			if followup != nil || len(m.dialogs) != 0 || m.err == "" || !strings.Contains(m.notice, "Select Refresh installation") {
				t.Fatalf("stale recovery: %s %s", m.err, m.notice)
			}
			if migrationSnapshot(t, base) != before {
				t.Fatal("stale apply wrote files")
			}
			if change == "source" || change == "inventory" {
				m.cursor = 1
				m, cmd = press(m, "enter")
				m, _ = complete(m, cmd)
				if m.err != "" || m.preview.Plan.Request.Intent != install.Refresh || len(m.dialogs) == 0 {
					t.Fatal("refresh retry did not require review")
				}
			}
		})
	}
	for _, intent := range []install.Intent{install.Configure, install.Refresh, install.Migrate} {
		m, b := readyModel(t, false)
		m = previewAction(t, m, intent)
		b.fail = "apply"
		m, cmd := press(m, "y")
		m, followup := complete(m, cmd)
		if followup != nil {
			t.Fatal("failure automatically retried")
		}
		expected := map[install.Intent]string{install.Configure: "Press s", install.Refresh: "Select Refresh installation", install.Migrate: "Select Migrate linked installation"}[intent]
		if !strings.Contains(m.notice, expected) {
			t.Fatalf("wrong recovery for %s: %s", intent, m.notice)
		}
	}
}

func TestContentReviewControlsAndScrolling(t *testing.T) {
	m, _ := readyModel(t, false)
	m.width, m.height = 30, 8
	m.dialogs = []dialog{{"Apply? [y/N]"}}
	for i := 0; i < 150; i++ {
		m.preview.Changes = append(m.preview.Changes, install.ContentChange{Provider: "codex", Name: "fixture", Path: fmt.Sprintf("resource-%03d", i), Kind: "modified", Before: &install.ContentState{Type: "file", Mode: 0644}, After: &install.ContentState{Type: "file", Mode: 0755}, Lines: []string{"- old", "+ new"}})
	}
	m.preview.Changes[0].Path += "\x1b[2J\r\u202e"
	m.preview.Changes[0].Lines = []string{"+ malicious\x1b]52;c;data\a\b\t"}
	m.preview.Changes[1].Binary, m.preview.Changes[1].Truncated = true, true
	m.preview.Changes[1].After.Size, m.preview.Changes[1].After.SHA256 = 42, "binary-digest"
	rendered := strings.Join(m.operationLines(), "\n")
	for _, raw := range []string{"\x1b", "\r", "\a", "\b", "\t", "\u202e"} {
		if strings.Contains(rendered, raw) {
			t.Fatalf("unescaped terminal control %q", raw)
		}
	}
	for _, text := range []string{"\\x1b", "\\r", "\\u202e", "Binary content", "42 bytes, sha256 binary-digest", "Text excerpt truncated", "0644 -> file mode 0755"} {
		if !strings.Contains(rendered, text) {
			t.Fatalf("missing %q", text)
		}
	}
	all := m.wrappedLines()
	covered := make([]bool, len(all))
	m.manualScroll, m.scroll = true, 0
	for {
		offset := m.viewOffset()
		_, height := m.viewportSize()
		for i := offset; i < min(offset+height, len(all)); i++ {
			covered[i] = true
		}
		next, cmd := press(m, "pgdown")
		if cmd != nil {
			t.Fatal("scroll triggered an action")
		}
		if next.viewOffset() == offset {
			break
		}
		m = next
	}
	for i, seen := range covered {
		if !seen {
			t.Fatalf("review row %d unreachable", i)
		}
	}
	for m.viewOffset() > 0 {
		m, _ = press(m, "pgup")
	}
	if !strings.Contains(m.View().Content, "pgup/pgdown") {
		t.Fatal("missing scroll controls")
	}
	// The original review collection and plan keep the source bytes unchanged.
	if !strings.Contains(m.preview.Changes[0].Lines[0], "\x1b") {
		t.Fatal("display escaping altered review data")
	}
}

func TestMigrationReviewKeepsCopiedMembersAbsent(t *testing.T) {
	m, base := linkedModel(t, true)
	m = reviewMigration(t, m)
	for _, change := range m.preview.Changes {
		if change.Provider == "codex" && change.Name == "plan-review" {
			t.Fatal("unchanged copied member appeared")
		}
	}
	review := strings.Join(m.operationLines(), "\n")
	for _, text := range []string{"converted", "symlink mode", "scripts/nested/run.sh", "file mode 0755"} {
		if !strings.Contains(review, text) {
			t.Fatalf("conversion detail missing %q", text)
		}
	}
	before, err := m.backend.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := migrationSnapshot(t, base)
	m, cmd := press(m, "esc")
	after, err := m.backend.Status(context.Background())
	if err != nil || cmd != nil || before.Revision != after.Revision || migrationSnapshot(t, base) != snapshot {
		t.Fatal("migration review cancellation changed installation")
	}
}
