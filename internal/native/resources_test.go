package native

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

func TestRepositoryReleaseResources(t *testing.T) {
	root := filepath.Join(t.TempDir(), "release")
	must(t, Build("../..", "1.0.0", root))
	c, err := catalog.Load("../..")
	must(t, err)
	for _, provider := range providers {
		pkg := filepath.Join(root, provider, "plugins/bundle-all")
		for _, skill := range c.Skills {
			must(t, filepath.WalkDir(skill.Source, func(path string, e os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if e.IsDir() || filepath.Ext(path) == ".md" {
					return nil
				}
				rel, err := filepath.Rel(skill.Source, path)
				if err != nil {
					return err
				}
				copied := filepath.Join(pkg, "skills", skill.Name, rel)
				original, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				data, err := os.ReadFile(copied)
				if err != nil {
					return err
				}
				info, err := os.Stat(path)
				if err != nil {
					return err
				}
				actual, err := os.Stat(copied)
				if err != nil {
					return err
				}
				if !bytes.Equal(data, original) || mode(info) != mode(actual) {
					t.Fatalf("resource bytes or modes changed: %s", path)
				}
				return nil
			}))
		}
		// Inspect the bundled checker and its references; never execute Python.
		checker := filepath.Join(pkg, "skills/python-standards/scripts/check_function_size.py")
		data, err := os.ReadFile(checker)
		must(t, err)
		if !bytes.Contains(data, []byte("def main(")) {
			t.Fatal("bundled checker missing implementation")
		}
		must(t, filepath.WalkDir(filepath.Join(pkg, "skills/python-standards"), func(path string, e os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(data), ".agents/skills/coding/python-standards/scripts/") {
				t.Fatal("clone-dependent checker reference")
			}
			return nil
		}))
		data, err = os.ReadFile(filepath.Join(root, provider, "plugins/ticket-to-plan/skills/ticket-to-plan/SKILL.md"))
		must(t, err)
		for _, text := range []string{"../plan-review/SKILL.md", "`quick-plan` (optional integration)"} {
			if !strings.Contains(string(data), text) {
				t.Fatalf("missing native contract %q", text)
			}
		}
		if !strings.Contains(string(data), "../../settings/markdown-output.md") || strings.Contains(string(data), "## Markdown artifact outputs") {
			t.Fatal("native skill must route to the shared module")
		}
		module := filepath.Join(root, provider, "plugins/ticket-to-plan/settings/markdown-output.md")
		policy, err := os.ReadFile(module)
		must(t, err)
		original, err := os.ReadFile(filepath.Join(c.Root, ".agents/global/markdown-output.md"))
		must(t, err)
		if !bytes.HasPrefix(policy, original) || !bytes.Contains(policy, []byte("Honor explicit user guidance")) {
			t.Fatal("native module missing shared instructions or standalone defaults")
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(module), "SKILL.md")); !os.IsNotExist(err) {
			t.Fatal("internal module is invokable")
		}
		data, err = os.ReadFile(filepath.Join(root, provider, "plugins/ruby-rails-standards/skills/ruby-rails-standards/SKILL.md"))
		must(t, err)
		if !strings.Contains(string(data), "../ruby-standards/SKILL.md") {
			t.Fatal("Rails helper is not package-local")
		}
	}
}
