package validate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(filepath.Join(root, ".agents"), os.DirFS("../../.agents")); err != nil {
		t.Fatal(err)
	}
	return root
}

func put(t *testing.T, root, name, text string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSourceValidation(t *testing.T) {
	root := fixture(t)
	if err := Source(root); err != nil {
		t.Fatal(err)
	}
	// Provider adapters are optional; their presence must not become mandatory.
	if err := os.Remove(filepath.Join(root, ".agents/providers/codex/skills/review/plan-review/agents/openai.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := Source(root); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != ".agents" {
		t.Fatalf("validation wrote files: %v %v", entries, err)
	}
}

func TestInvalidSourceContracts(t *testing.T) {
	for _, kind := range []string{"name", "description", "frontmatter", "duplicate", "provider-field", "broken-link", "escaping-link", "source-adapter", "moved-adapter", "orphan-adapter", "missing-registry", "unknown-helper", "duplicate-helper", "output-registry", "symlink", "layout"} {
		t.Run(kind, func(t *testing.T) {
			root := fixture(t)
			manifest := ".agents/skills/productivity/commit-message/SKILL.md"
			switch kind {
			case "name":
				put(t, root, manifest, "---\nname: wrong\ndescription: Example\n---\n")
			case "description":
				put(t, root, manifest, "---\nname: commit-message\n---\n")
			case "frontmatter":
				put(t, root, manifest, "---\nname: commit-message\n")
			case "duplicate":
				put(t, root, manifest, "---\nname: commit-message\nname: commit-message\ndescription: Example\n---\n")
			case "provider-field":
				put(t, root, manifest, "---\nname: commit-message\ndescription: Example\ndisable-model-invocation: true\n---\n")
			case "broken-link":
				put(t, root, manifest, "---\nname: commit-message\ndescription: Example\n---\n[missing](missing.md)")
			case "escaping-link":
				put(t, root, manifest, "---\nname: commit-message\ndescription: Example\n---\n[escape](../../../../../outside.md)")
			case "source-adapter":
				put(t, root, ".agents/skills/productivity/commit-message/agents/openai.yaml", "policy: {}\n")
			case "moved-adapter":
				path := filepath.Join(root, ".agents/providers/codex/skills/review/plan-review/agents/openai.yaml")
				put(t, root, ".agents/providers/codex/skills/scoping/plan-review/agents/openai.yaml", "policy: {}\n")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "orphan-adapter":
				put(t, root, ".agents/providers/codex/skills/review/removed/agents/openai.yaml", "policy: {}\n")
			case "missing-registry":
				if err := os.Remove(filepath.Join(root, ".agents/providers/claude/explicit-only.txt")); err != nil {
					t.Fatal(err)
				}
			case "unknown-helper":
				put(t, root, ".agents/dependencies.txt", "ticket-to-plan unknown\n")
			case "duplicate-helper":
				put(t, root, ".agents/dependencies.txt", "ticket-to-plan plan-review\nticket-to-plan plan-review\n")
			case "output-registry":
				put(t, root, ".agents/markdown-output-skills.txt", "unknown\n")
			case "symlink":
				if err := os.Symlink("/etc/passwd", filepath.Join(root, ".agents/secret")); err != nil {
					t.Fatal(err)
				}
			case "layout":
				put(t, root, "config.yml", "obsolete\n")
			}
			if Source(root) == nil {
				t.Fatal("accepted invalid source")
			}
		})
	}
}

func TestIsolatedVerification(t *testing.T) {
	root := fixture(t)
	for _, key := range []string{"HOME", "AGENTS_HOME", "CODEX_HOME", "CLAUDE_HOME"} {
		t.Setenv(key, filepath.Join(t.TempDir(), "must-not-write"))
	}
	if err := Verify(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"HOME", "AGENTS_HOME", "CODEX_HOME", "CLAUDE_HOME"} {
		if _, err := os.Lstat(os.Getenv(key)); !os.IsNotExist(err) {
			t.Fatalf("verification touched %s", key)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "config")); !os.IsNotExist(err) {
		t.Fatal("verification wrote source settings")
	}
}
