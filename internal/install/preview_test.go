package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testService(t *testing.T) *Service {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "toolkit")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(root, ".agents"), os.DirFS("../../.agents")); err != nil {
		t.Fatal(err)
	}
	s, err := New(root, Homes{filepath.Join(base, "agents home"), filepath.Join(base, "codex home"), filepath.Join(base, "claude home")})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func saveFixture(t *testing.T, s *Service, settings Settings, applied map[string][]string) {
	t.Helper()
	data := settingsBytes(settings)
	inv, err := inventoryBytes(data, applied)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(s.Root, "config/settings.env"), string(data))
	write(t, filepath.Join(s.Root, "config/inventory.json"), string(inv))
}

func installedFixture(t *testing.T, s *Service, provider, name string, settings Settings, selected []string) string {
	t.Helper()
	c, err := s.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	skill, err := c.Lookup(name)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := s.destinations(provider)
	path := filepath.Join(root, name)
	manifest, err := expectedManifest(c, name, provider, root, settings, selected)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(path, "SKILL.md"), manifest)
	write(t, filepath.Join(path, markerName), s.Root+"\nSKILL.md "+digest([]byte(manifest))+"\n")
	err = filepath.WalkDir(skill.Source, func(source string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || source == filepath.Join(skill.Source, "SKILL.md") {
			return nil
		}
		rel, err := filepath.Rel(skill.Source, source)
		if err != nil {
			return err
		}
		target := filepath.Join(path, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.Symlink(source, target)
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
	return path
}

func TestFreshPreviewIsReadOnly(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	status, err := s.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Installed {
		t.Fatal("fresh installation reported saved state")
	}
	p, err := s.Preview(ctx, Request{Intent: Configure, Desired: Settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Operations) != 1 || p.Operations[0].Action != "install" || p.Operations[0].Provider != "codex" {
		t.Fatal(p)
	}
	if !strings.Contains(p.Operations[0].Path, "agents home/skills/agent-toolkit/commit-message") {
		t.Fatal(p.Operations)
	}
	for _, path := range []string{filepath.Join(s.Root, "config"), s.Homes.Agents, s.Homes.Codex, s.Homes.Claude} {
		if exists(path) {
			t.Fatalf("read created %s", path)
		}
	}
}

func TestSavedStateRoundTripAndRetainedHelpers(t *testing.T) {
	s := testService(t)
	settings := Settings{Providers: []string{"claude", "codex"}, Requested: []string{"ticket-to-plan"}, Output: filepath.Join(t.TempDir(), `space "quoted" $(touch NEVER)`), FullTemplate: true}
	saveFixture(t, s, settings, map[string][]string{"codex": {"ticket-to-plan", "plan-review"}, "claude": {}})
	got, err := s.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(settingsBytes(got.Desired)) != string(settingsBytes(settings)) {
		t.Fatalf("state = %+v, want %+v", got.Desired, settings)
	}
	p, err := s.Preview(context.Background(), Request{Intent: Configure, Desired: Settings{Providers: []string{"codex"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(p.Desired.Retained, "plan-review") {
		t.Fatal(p.Desired)
	}
	if exists(filepath.Join(s.Root, "NEVER")) {
		t.Fatal("evaluated saved data")
	}
}

func TestManifestOwnershipAndResources(t *testing.T) {
	for _, change := range []string{"keep", "modified", "unrelated", "missing", "dangling", "symlink"} {
		t.Run(change, func(t *testing.T) {
			s := testService(t)
			settings := Settings{Providers: []string{"codex"}, Requested: []string{"python-standards"}}
			applyFor(t, s, previewFor(t, s, Request{Intent: Configure, Desired: settings}))
			root, _ := s.destinations("codex")
			path := filepath.Join(root, "python-standards")
			resource := filepath.Join(path, "scripts/check_function_size.py")
			switch change {
			case "modified":
				write(t, filepath.Join(path, "SKILL.md"), "user changed manifest\n")
			case "unrelated":
				write(t, filepath.Join(path, "user.txt"), "keep me")
			case "missing":
				if err := os.Remove(resource); err != nil {
					t.Fatal(err)
				}
			case "dangling":
				if err := os.Remove(resource); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("/missing", resource); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(path, path+"-original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+"-original", path); err != nil {
					t.Fatal(err)
				}
			}
			p, err := s.Preview(context.Background(), Request{Intent: Configure, Desired: settings})
			if err != nil {
				t.Fatal(err)
			}
			want := "conflict"
			if change == "keep" {
				want = "keep"
			}
			if p.Operations[0].Action != want {
				t.Fatalf("got %s want %s", p.Operations[0].Action, want)
			}
		})
	}
}

func TestGuidanceAndUninstall(t *testing.T) {
	s := testService(t)
	settings := Settings{Providers: []string{"codex"}, Concise: true}
	_, path := s.destinations("codex")
	block, err := guidanceBlock(s.Root, settings)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, "Hello user\n\n"+block)
	write(t, filepath.Join(filepath.Dir(path), markerName), s.Root+"\nGUIDANCE_BLOCK "+digest([]byte(block))+"\n")
	p, err := s.Preview(context.Background(), Request{Intent: Uninstall})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Confirmation || len(p.Operations) != 1 || p.Operations[0].Action != "strip-guidance" {
		t.Fatal(p)
	}
	write(t, path, "Hello user\n\n"+strings.Replace(block, "Concise output", "Modified output", 1))
	p, err = s.Preview(context.Background(), Request{Intent: Uninstall})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Conflicts) != 1 || p.Conflicts[0].Path != path {
		t.Fatal(p)
	}
}

func TestMalformedStateAndUnsafeRoots(t *testing.T) {
	for _, change := range []string{"pair", "version", "boolean", "duplicate", "inventory", "inventory-trailing", "state-link", "provider-link", "unknown-refresh"} {
		t.Run(change, func(t *testing.T) {
			s := testService(t)
			settings := Settings{Providers: []string{"codex"}, Requested: []string{"commit-message"}}
			saveFixture(t, s, settings, map[string][]string{"codex": {}, "claude": {}})
			path := filepath.Join(s.Root, "config/settings.env")
			switch change {
			case "pair":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "version":
				write(t, path, strings.Replace(string(settingsBytes(settings)), "STATE_VERSION=1", "STATE_VERSION=2", 1))
			case "boolean":
				write(t, path, strings.Replace(string(settingsBytes(settings)), "CONCISE_OUTPUT=false", "CONCISE_OUTPUT=wrong", 1))
			case "duplicate":
				write(t, path, string(settingsBytes(settings))+"PROVIDERS=claude\n")
			case "inventory":
				write(t, filepath.Join(s.Root, "config/inventory.json"), "{}")
			case "inventory-trailing":
				inventoryPath := filepath.Join(s.Root, "config/inventory.json")
				data, err := os.ReadFile(inventoryPath)
				if err != nil {
					t.Fatal(err)
				}
				write(t, inventoryPath, string(data)+"{}")
			case "state-link":
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".original", path); err != nil {
					t.Fatal(err)
				}
			case "provider-link":
				if err := os.MkdirAll(s.Homes.Agents, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(t.TempDir(), filepath.Join(s.Homes.Agents, "skills")); err != nil {
					t.Fatal(err)
				}
			case "unknown-refresh":
				settings.Requested = []string{"removed-skill"}
				saveFixture(t, s, settings, map[string][]string{"codex": {}, "claude": {}})
			}
			p, err := s.Preview(context.Background(), Request{Intent: Refresh})
			if change == "provider-link" {
				if err != nil || len(p.Conflicts) == 0 {
					t.Fatalf("unsafe root: %v %+v", err, p)
				}
				return
			}
			if err == nil {
				t.Fatal("accepted malformed state")
			}
		})
	}
}

func TestRevisionTracksSourceAndDestination(t *testing.T) {
	s := testService(t)
	before, err := s.revision()
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(s.Root, "internal/install/implementation.go"), "changed implementation")
	after, err := s.revision()
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("implementation change did not change revision")
	}
	root, _ := s.destinations("codex")
	write(t, filepath.Join(root, "user-file"), "changed destination")
	next, err := s.revision()
	if err != nil {
		t.Fatal(err)
	}
	if after == next {
		t.Fatal("destination change did not change revision")
	}
}
