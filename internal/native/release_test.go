package native

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func sourceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, text := range map[string]string{
		".agents/skills/workflow/task/SKILL.md":                           "---\nname: task\ndescription: Task workflow\n---\nUse [helper](../../shared/helper/SKILL.md) and $helper.\n",
		".agents/skills/shared/helper/SKILL.md":                           "---\nname: helper\ndescription: Shared guidance\n---\nRead [reference](references/guide.md).\n",
		".agents/skills/shared/helper/references/guide.md":                "Guidance\n",
		".agents/bundles.txt":                                             "starter task\n",
		".agents/dependencies.txt":                                        "task helper\n",
		".agents/markdown-output-skills.txt":                              "task\n",
		".agents/providers/claude/explicit-only.txt":                      "workflow/task\n",
		".agents/providers/codex/skills/workflow/task/agents/openai.yaml": "policy:\n  allow_implicit_invocation: false\n",
		".agents/global/instructions.md":                                  "## Markdown artifact outputs\nKeep artifacts together.\n",
		"LICENSE":                                                         "Fixture license\n",
	} {
		must(t, writeFile(filepath.Join(root, name), []byte(text), 0644))
	}
	must(t, writeFile(filepath.Join(root, ".agents/skills/shared/helper/tool"), []byte("executable resource"), 0755))
	return root
}

func copyTree(t *testing.T, source, target string) {
	t.Helper()
	must(t, os.CopyFS(target, os.DirFS(source)))
	// Normalize fixture modes independently of CopyFS and the caller's umask.
	must(t, filepath.WalkDir(source, func(path string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		return os.Chmod(filepath.Join(target, rel), info.Mode().Perm())
	}))
}

func TestReleaseRelocationAndReproducibility(t *testing.T) {
	source := sourceFixture(t)
	base := t.TempDir()
	first := filepath.Join(base, "first")
	second := filepath.Join(base, "second")
	must(t, Build(source, "1.2.3", first))
	must(t, Build(source, "1.2.3", second))
	a, _, err := inventory(first, false)
	must(t, err)
	b, _, err := inventory(second, false)
	must(t, err)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("release is not reproducible")
	}
	must(t, os.Rename(source, source+"-hidden"))
	t.Cleanup(func() { must(t, os.RemoveAll(source+"-hidden")) })
	must(t, Validate(first))
	if err := Build(source+"-hidden", "1.2.3", first); err == nil {
		t.Fatal("overwrote output")
	}
}

func TestReleaseCorruption(t *testing.T) {
	base := t.TempDir()
	original := filepath.Join(base, "original")
	must(t, Build(sourceFixture(t), "1.0.0", original))
	for _, kind := range []string{"resource", "archive", "marketplace", "closure", "manifest", "symlink", "traversal", "mode", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "release")
			copyTree(t, original, root)
			var r release
			must(t, readJSON(filepath.Join(root, "release.json"), &r))
			record := &r.Packages[0]
			pkg := filepath.Join(root, record.Provider, "plugins", record.Name)
			switch kind {
			case "resource":
				must(t, writeFile(filepath.Join(pkg, "LICENSE"), []byte("changed"), 0644))
			case "archive":
				must(t, writeFile(filepath.Join(root, record.Provider, "zips", record.Name+".zip"), []byte("broken"), 0644))
			case "marketplace":
				must(t, writeFile(filepath.Join(root, "claude", marketplacePath("claude")), []byte(`{"name":"agent-toolkit","plugins":[]}`), 0644))
			case "closure":
				record.Skills = []string{}
			case "manifest":
				must(t, writeFile(filepath.Join(pkg, "."+record.Provider+"-plugin/plugin.json"), []byte("{}"), 0644))
				record.Files, _, _ = inventory(pkg, false)
			case "symlink":
				must(t, os.Symlink("/missing", filepath.Join(root, "link")))
			case "malformed":
				must(t, writeFile(filepath.Join(root, "release.json"), []byte("null"), 0644))
				if Validate(root) == nil {
					t.Fatal("accepted null")
				}
				return
			case "mode":
				must(t, os.Chmod(filepath.Join(pkg, "LICENSE"), 0755))
			case "traversal":
				archive := filepath.Join(root, record.Provider, "zips", record.Name+".zip")
				f, err := os.Create(archive)
				must(t, err)
				w := zip.NewWriter(f)
				for name := range record.Files {
					h := &zip.FileHeader{Name: name}
					if name == "LICENSE" {
						h.Name = "../../escaped"
					}
					h.SetMode(0644)
					out, err := w.CreateHeader(h)
					must(t, err)
					data, err := os.ReadFile(filepath.Join(pkg, name))
					must(t, err)
					_, err = out.Write(data)
					must(t, err)
				}
				must(t, w.Close())
				must(t, f.Close())
				data, err := os.ReadFile(archive)
				must(t, err)
				record.Archive = digest(data)
			}
			must(t, writeJSON(filepath.Join(root, "release.json"), r))
			if err := Validate(root); err == nil {
				t.Fatal("accepted corruption")
			}
		})
	}
}

func TestRejectUnsafeBuildInputs(t *testing.T) {
	for _, kind := range []string{"version", "symlink", "escaping-link", "broken-link", "file-link"} {
		t.Run(kind, func(t *testing.T) {
			source := sourceFixture(t)
			out := filepath.Join(t.TempDir(), "output")
			version := "1.0.0"
			switch kind {
			case "file-link":
				must(t, writeFile(filepath.Join(source, ".agents/skills/shared/helper/references/guide.md"), []byte("[secret](file:///etc/passwd)"), 0644))
			case "version":
				version = "01.0.0"
			case "symlink":
				must(t, os.Symlink("/etc/passwd", filepath.Join(source, ".agents/skills/shared/helper/secret")))
			case "escaping-link":
				must(t, writeFile(filepath.Join(source, ".agents/skills/shared/helper/references/guide.md"), []byte("[escape](../../../../secret)"), 0644))
			case "broken-link":
				must(t, writeFile(filepath.Join(source, ".agents/skills/shared/helper/references/guide.md"), []byte("[missing](missing.md)"), 0644))
			}
			if Build(source, version, out) == nil {
				t.Fatal("accepted unsafe input")
			}
			if _, err := os.Lstat(out); !os.IsNotExist(err) {
				t.Fatal("failed build left output")
			}
		})
	}
}

func extractFixture(t *testing.T, name string) string {
	t.Helper()
	root := t.TempDir()
	r, err := zip.OpenReader(filepath.Join("testdata", name))
	must(t, err)
	defer r.Close()
	for _, file := range r.File {
		if !safeName(strings.TrimSuffix(file.Name, "/")) {
			t.Fatal(file.Name)
		}
		if file.FileInfo().IsDir() {
			must(t, os.MkdirAll(filepath.Join(root, file.Name), 0755))
			continue
		}
		in, err := file.Open()
		must(t, err)
		data, err := io.ReadAll(in)
		must(t, err)
		must(t, in.Close())
		must(t, writeFile(filepath.Join(root, file.Name), data, file.Mode().Perm()))
	}
	return root
}

func TestHistoricalRelease(t *testing.T) {
	must(t, Validate(extractFixture(t, "historical-release.zip")))
}
