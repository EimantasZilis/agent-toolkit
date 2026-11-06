package native

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func committedSource(t *testing.T) string {
	t.Helper()
	root := sourceFixture(t)
	for name, text := range map[string]string{
		"cmd/fixture/main.go":         "package main\n",
		"internal/fixture/fixture.go": "package fixture\n",
		"go.mod":                      "module fixture\n",
		"go.sum":                      "",
		".gitignore":                  "ignored\n",
	} {
		must(t, writeFile(filepath.Join(root, name), []byte(text), 0644))
	}
	_, err := gitQuery(root, "init", "-q")
	must(t, err)
	_, err = gitQuery(root, "add", ".")
	must(t, err)
	_, err = gitQuery(root, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "-qm", "Snapshot fixture")
	must(t, err)
	return root
}

func TestSnapshotPortableAndDeterministic(t *testing.T) {
	source := committedSource(t)
	base := t.TempDir()
	first := filepath.Join(base, "snapshot")
	second := filepath.Join(base, "second")
	must(t, Snapshot(source, "1.2.3", first))
	must(t, Snapshot(source, "1.2.3", second))
	a, m, err := inventory(first, true)
	must(t, err)
	b, n, err := inventory(second, true)
	must(t, err)
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(m, n) {
		t.Fatal("snapshot differs between builds")
	}
	if Snapshot(source, "1.2.3", first) == nil {
		t.Fatal("overwrote existing snapshot")
	}

	var record snapshotRecord
	must(t, readJSON(filepath.Join(first, "snapshot.json"), &record))
	commit, err := gitQuery(source, "rev-parse", "HEAD")
	must(t, err)
	if record.Commit != commit {
		t.Fatal("incorrect provenance")
	}
	must(t, os.Rename(source, source+"-hidden"))
	t.Cleanup(func() { must(t, os.RemoveAll(source+"-hidden")) })
	_, err = gitQuery(first, "init", "-q")
	must(t, err)
	must(t, ValidateSnapshot(first))
}

func TestSnapshotRequiresCommittedInputs(t *testing.T) {
	for _, kind := range []string{"dirty", "staged", "untracked", "ignored", "symlink", "mode", "committed-symlink", "assume-unchanged"} {
		t.Run(kind, func(t *testing.T) {
			source := committedSource(t)
			path := filepath.Join(source, "internal/fixture/fixture.go")
			switch kind {
			case "assume-unchanged":
				_, err := gitQuery(source, "update-index", "--assume-unchanged", "internal/fixture/fixture.go")
				must(t, err)
				must(t, writeFile(path, []byte("hidden edit\n"), 0644))
			case "dirty", "staged":
				must(t, writeFile(path, []byte("uncommitted\n"), 0644))
				if kind == "staged" {
					_, err := gitQuery(source, "add", "internal")
					must(t, err)
				}
			case "mode":
				must(t, os.Chmod(path, 0755))
			case "untracked":
				must(t, writeFile(filepath.Join(source, "cmd/new.go"), []byte("new"), 0644))
			case "ignored":
				must(t, writeFile(filepath.Join(source, ".agents/ignored"), []byte("ignored resource"), 0644))
			case "symlink", "committed-symlink":
				must(t, os.Symlink("/etc/passwd", filepath.Join(source, ".agents/link")))
				if kind == "committed-symlink" {
					_, err := gitQuery(source, "add", ".agents/link")
					must(t, err)
					_, err = gitQuery(source, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "-qm", "Symlink fixture")
					must(t, err)
				}
			}
			out := filepath.Join(t.TempDir(), "snapshot")
			if Snapshot(source, "1.0.0", out) == nil {
				t.Fatal("accepted uncommitted or unsafe source")
			}
			if _, err := os.Lstat(out); !os.IsNotExist(err) {
				t.Fatal("rejected snapshot left output")
			}
		})
	}
}

func resealSnapshot(t *testing.T, root string) {
	t.Helper()
	var s snapshotRecord
	must(t, readJSON(filepath.Join(root, "snapshot.json"), &s))
	var err error
	s.Files, s.Modes, err = inventory(root, true)
	must(t, err)
	must(t, writeJSON(filepath.Join(root, "snapshot.json"), s))
}

func TestSnapshotCorruption(t *testing.T) {
	original := filepath.Join(t.TempDir(), "snapshot")
	must(t, Snapshot(committedSource(t), "1.0.0", original))
	for _, kind := range []string{"bytes", "missing", "mode", "provenance", "schema", "symlink", "root-symlink", "git-symlink", "catalog-path", "cross-provider", "missing-catalog", "manifest"} {
		t.Run(kind, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "snapshot")
			copyTree(t, original, root)
			resource := filepath.Join(root, "codex/plugins/task/skills/task/SKILL.md")
			switch kind {
			case "bytes":
				must(t, writeFile(resource, []byte("changed"), 0644))
			case "missing":
				must(t, os.Remove(resource))
			case "mode":
				must(t, os.Chmod(resource, 0755))
			case "provenance", "schema":
				var s snapshotRecord
				must(t, readJSON(filepath.Join(root, "snapshot.json"), &s))
				if kind == "schema" {
					s.Schema = 2
				} else {
					s.Commit = "main"
				}
				must(t, writeJSON(filepath.Join(root, "snapshot.json"), s))
			case "symlink":
				must(t, os.Symlink("/etc/passwd", filepath.Join(root, "secret")))
			case "root-symlink":
				link := root + "-link"
				must(t, os.Symlink(root, link))
				root = link
			case "git-symlink":
				must(t, os.Symlink(t.TempDir(), filepath.Join(root, ".git")))
			case "missing-catalog":
				must(t, os.Remove(filepath.Join(root, marketplacePath("claude"))))
				resealSnapshot(t, root)
			case "manifest":
				path := "codex/plugins/task/.codex-plugin/plugin.json"
				must(t, writeFile(filepath.Join(root, path), []byte("{}"), 0644))
				var r release
				must(t, readJSON(filepath.Join(root, "release.json"), &r))
				for i := range r.Packages {
					if r.Packages[i].Provider == "codex" && r.Packages[i].Name == "task" {
						r.Packages[i].Files[".codex-plugin/plugin.json"] = digest([]byte("{}"))
					}
				}
				must(t, writeJSON(filepath.Join(root, "release.json"), r))
				resealSnapshot(t, root)
			case "catalog-path", "cross-provider":
				path := filepath.Join(root, marketplacePath("codex"))
				var m map[string]any
				must(t, readJSON(path, &m))
				entry := m["plugins"].([]any)[0].(map[string]any)
				source := entry["source"].(map[string]any)
				source["path"] = "../escape"
				if kind == "cross-provider" {
					source["path"] = "./claude/plugins/" + entry["name"].(string)
				}
				must(t, writeJSON(path, m))
				resealSnapshot(t, root)
			}
			if ValidateSnapshot(root) == nil {
				t.Fatal("accepted corrupt snapshot")
			}
		})
	}
}

func TestHistoricalSnapshot(t *testing.T) {
	must(t, ValidateSnapshot(extractFixture(t, "historical-snapshot.zip")))
}

func TestJSONWhitespaceRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	must(t, os.WriteFile(path, []byte(" \n\t"), 0644))
	var value json.RawMessage
	if readJSON(path, &value) == nil {
		t.Fatal("accepted empty JSON")
	}
}
