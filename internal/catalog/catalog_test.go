package catalog

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSelectionContracts(t *testing.T) {
	c, err := Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, provider := range []string{"codex", "claude"} {
		for name, selection := range map[string]string{"starter": "starter", "all": "all", "coding": "coding", "individual": "ticket-to-plan ruby-rails-standards ticket-to-plan"} {
			t.Run(provider+"/"+name, func(t *testing.T) {
				data, err := os.ReadFile("testdata/" + name + ".txt")
				if err != nil {
					t.Fatal(err)
				}
				got, err := c.Select(selection, false)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, strings.Fields(string(data))) {
					t.Fatalf("selection = %v, want %s", got, data)
				}
			})
		}
	}
	if _, err := c.Select("unknown", false); err == nil {
		t.Fatal("accepted unknown selection")
	}
	if _, err := c.Select("", false); err == nil {
		t.Fatal("accepted empty selection")
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, data := range map[string]string{
		"skills/a/one/SKILL.md":                           "---\nname: one\n---\n",
		"skills/b/two/SKILL.md":                           "---\nname: two\n---\n",
		"bundles.txt":                                     "starter one\n",
		"dependencies.txt":                                "one two\ntwo one\n",
		"markdown-output-skills.txt":                      "one\n",
		"providers/claude/explicit-only.txt":              "a/one\n",
		"providers/codex/skills/a/one/agents/openai.yaml": "display_name: One\n",
	} {
		writeFixture(t, root, name, data)
	}
	return root
}

func writeFixture(t *testing.T, root, name, data string) {
	t.Helper()
	path := filepath.Join(root, ".agents", name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCyclesAndMetadata(t *testing.T) {
	c, err := Load(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Select("one one", false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"one", "two"}) {
		t.Fatalf("cycle closure = %v", got)
	}
	one, err := c.Lookup("one")
	if err != nil {
		t.Fatal(err)
	}
	if !one.Output || !one.Explicit || one.CodexAdapter == "" {
		t.Fatalf("missing metadata: %+v", one)
	}
	if !reflect.DeepEqual(c.BundleNames(), []string{"a", "all", "b", "starter"}) {
		t.Fatal(c.BundleNames())
	}
}

func TestInvalidRegistries(t *testing.T) {
	for _, tc := range []struct{ name, data string }{
		{"bundles.txt", "starter unknown\n"},
		{"bundles.txt", "all one\nstarter one\n"},
		{"bundles.txt", "starter one\nstarter two\n"},
		{"dependencies.txt", "one unknown\n"},
		{"dependencies.txt", "one\n"},
		{"markdown-output-skills.txt", "one\none\n"},
		{"providers/claude/explicit-only.txt", "b/one\n"},
		{"skills/b/one/SKILL.md", "---\nname: one\n---\n"},
		{"providers/codex/skills/b/one/agents/openai.yaml", "name: one\n"},
	} {
		t.Run(tc.name+tc.data, func(t *testing.T) {
			root := fixture(t)
			writeFixture(t, root, tc.name, tc.data)
			if _, err := Load(root); err == nil {
				t.Fatal("accepted invalid catalog")
			}
		})
	}
}
