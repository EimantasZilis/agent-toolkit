package tui

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

func writeMigrationFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func linkedModel(t *testing.T, mixed bool) (model, string) {
	t.Helper()
	m, base := backendFixture(t)

	service := m.backend.(*install.Service)
	source := filepath.Join(service.Root, ".agents/skills/coding/python-standards/scripts/nested/run.sh")
	writeMigrationFile(t, source, []byte("#!/bin/sh\nexit 0\n"))
	if err := os.Chmod(source, 0755); err != nil {
		t.Fatal(err)
	}

	m.draft = settings{Providers: []string{"codex", "claude"}, Requested: []string{"python-standards", "ticket-to-plan"}, Retained: []string{"plan-writing-conventions"}, Concise: true, FullTemplate: true, Output: filepath.Join(base, "custom output")}
	m.globalOutput = true
	m = saveFixture(t, m)

	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c, err := service.Catalog()
	if err != nil {
		t.Fatal(err)
	}

	for provider, names := range status.Applied {
		root := filepath.Join(service.Homes.Agents, "skills/agent-toolkit")
		if provider == "claude" {
			root = filepath.Join(service.Homes.Claude, "skills")
		}
		for _, name := range names {
			if mixed && provider == "codex" && name == "plan-review" {
				continue
			}
			skill, err := c.Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, name)
			manifest, err := os.ReadFile(filepath.Join(path, "SKILL.md"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(path); err != nil {
				t.Fatal(err)
			}
			writeMigrationFile(t, filepath.Join(path, "SKILL.md"), manifest)
			writeMigrationFile(t, filepath.Join(path, ".agent-toolkit-owned"), []byte(fmt.Sprintf("%s\nSKILL.md %x\n", service.Root, sha256.Sum256(manifest))))
			err = filepath.WalkDir(skill.Source, func(file string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() || file == filepath.Join(skill.Source, "SKILL.md") {
					return nil
				}
				rel, err := filepath.Rel(skill.Source, file)
				if err != nil {
					return err
				}
				target := filepath.Join(path, rel)
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return err
				}
				return os.Symlink(file, target)
			})
			if err != nil {
				t.Fatal(err)
			}
			if provider == "codex" && skill.CodexAdapter != "" {
				if err := os.MkdirAll(filepath.Join(path, "agents"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(skill.CodexAdapter, filepath.Join(path, "agents/openai.yaml")); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	// Exercise the legacy inventory schema, preserving its settings checksum.
	invPath := filepath.Join(service.Root, "config/inventory.json")
	data, err := os.ReadFile(invPath)
	if err != nil {
		t.Fatal(err)
	}
	var inv map[string]any
	if err := json.Unmarshal(data, &inv); err != nil {
		t.Fatal(err)
	}
	inv["schema_version"] = 1
	delete(inv, "source")
	data, err = json.Marshal(inv)
	if err != nil {
		t.Fatal(err)
	}
	writeMigrationFile(t, invPath, data)

	m, _ = complete(m, m.Init())
	if m.err != "" || !m.status.Linked {
		t.Fatalf("legacy status: %+v %s", m.status, m.err)
	}

	return m, base
}

func migrationSnapshot(t *testing.T, root string) string {
	t.Helper()
	var out strings.Builder
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(&out, "%s %s\n", path, info.Mode())
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			fmt.Fprintln(&out, link)
		} else if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fmt.Fprintf(&out, "%x\n", sha256.Sum256(data))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func reviewMigration(t *testing.T, m model) model {
	t.Helper()
	m.cursor = 0
	for i := 0; i < 3; i++ {
		m, _ = press(m, "down")
	}
	m, cmd := press(m, "enter")
	if cmd == nil {
		t.Fatalf("migration did not start: %s", m.err)
	}
	m, _ = complete(m, cmd)
	return m
}

func TestMigrationReviewCancelAndApply(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(fmt.Sprint("mixed=", mixed), func(t *testing.T) {
			m, base := linkedModel(t, mixed)
			saved := clone(m.saved)
			before := migrationSnapshot(t, base)
			for _, cancel := range []string{"enter", "esc", "n"} {
				m = reviewMigration(t, m)
				review := strings.Join(m.viewLines(), "\n")
				for _, text := range []string{"convert", "Saved providers:", "Retained helpers:", "Required helper: plan-review", saved.Output, "Concise loading: true", "Full guidance: true", "[y/N]"} {
					if !strings.Contains(review, text) {
						t.Fatalf("review missing %q: %s", text, review)
					}
				}
				next, cmd := press(m, cancel)
				m = next
				if cmd != nil || len(m.dialogs) != 0 {
					t.Fatal("cancel started apply")
				}
				if migrationSnapshot(t, base) != before {
					t.Fatal("review/cancel wrote files")
				}
			}
			m = reviewMigration(t, m)
			m = finishDialogs(t, m)
			if m.status.Linked || !reflect.DeepEqual(m.saved, saved) || !strings.Contains(m.notice, "migrated to owned copies") {
				t.Fatalf("migration lost preferences or status: %+v", m)
			}
			if strings.Contains(strings.Join(m.managementActions(), "\n"), "Migrate") {
				t.Fatal("copied installation offers linked migration")
			}
			for _, root := range []string{filepath.Join(base, "agents/skills/agent-toolkit"), filepath.Join(base, "claude/skills")} {
				path := filepath.Join(root, "python-standards/scripts/nested/run.sh")
				info, err := os.Lstat(path)
				if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0755 {
					t.Fatalf("resource not copied: %v %v", info, err)
				}
			}
			installed := filepath.Join(base, "agents/skills/agent-toolkit/python-standards/scripts/nested/run.sh")
			old, err := os.ReadFile(installed)
			if err != nil {
				t.Fatal(err)
			}
			writeMigrationFile(t, filepath.Join(m.root, ".agents/skills/coding/python-standards/scripts/nested/run.sh"), []byte("edited checkout"))
			data, err := os.ReadFile(installed)
			if err != nil || string(data) != string(old) {
				t.Fatal("checkout edit changed installation")
			}
		})
	}
}

func TestMigrationRefusalAndStaleReview(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(fmt.Sprint("stale=", stale), func(t *testing.T) {
			m, base := linkedModel(t, false)
			if stale {
				m = reviewMigration(t, m)
				if m.err != "" || len(m.dialogs) == 0 {
					t.Fatalf("migration review failed: %s", m.err)
				}
			}
			path := filepath.Join(base, "claude/skills/python-standards/user.txt")
			writeMigrationFile(t, path, []byte("user file"))
			before := migrationSnapshot(t, base)
			if stale {
				next, apply := press(m, "y")
				if apply == nil {
					t.Fatal("migration acceptance did not start apply")
				}
				m, _ = complete(next, apply)
			} else {
				m = reviewMigration(t, m)
			}
			if m.err == "" || !strings.Contains(m.err, filepath.Dir(path)) {
				t.Fatalf("missing affected path: %s", m.err)
			}
			if !strings.Contains(m.notice, "preview again") || !m.status.Linked {
				t.Fatal("failure lost migration retry")
			}
			if migrationSnapshot(t, base) != before {
				t.Fatal("refusal/stale apply changed trees")
			}
		})
	}
}

func TestMigrationServiceFailureRemainsRetryable(t *testing.T) {
	m, backend := readyModel(t, false)
	m.page, m.status.Linked = managementPage, true
	backend.fail = "preview"
	m = reviewMigration(t, m)
	if m.err == "" || !strings.Contains(m.notice, "Migrate linked installation") {
		t.Fatal("preview service failure lost")
	}
	backend.fail = "apply"
	m = reviewMigration(t, m)
	next, cmd := press(m, "y")
	m, _ = complete(next, cmd)
	if m.err == "" || !strings.Contains(m.err, "recovery") {
		t.Fatal("apply service failure lost recovery diagnostic")
	}
}
