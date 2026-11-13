package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func legacyFixture(t *testing.T) (*Service, Settings, map[string][]string) {
	t.Helper()
	s := copiedService(t)
	settings := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"commit-message", "python-standards"}, Output: filepath.Join(s.Root, "custom output"), Concise: true, FullTemplate: true}
	// Include a nested executable resource and a directory that the historical
	// installer never materialized because it contained no files.
	c, err := s.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	skill, err := c.Lookup("python-standards")
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(skill.Source, "scripts/nested/run.sh"), "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(filepath.Join(skill.Source, "scripts/nested/run.sh"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(skill.Source, "empty"), 0755); err != nil {
		t.Fatal(err)
	}
	selected, err := c.Select(strings.Join(settings.Requested, " "), settings.Concise)
	if err != nil {
		t.Fatal(err)
	}
	applied := map[string][]string{}
	for _, provider := range settings.Providers {
		applied[provider] = selected
		for _, name := range selected {
			installedFixture(t, s, provider, name, settings, selected)
		}
	}
	saveFixture(t, s, settings, applied)
	return s, settings, applied
}

func installationSnapshot(t *testing.T, s *Service) string {
	t.Helper()
	var result strings.Builder
	for _, path := range []string{s.Root, s.Homes.Agents, s.Homes.Codex, s.Homes.Claude} {
		if err := fingerprint(path, &result); err != nil {
			t.Fatal(err)
		}
	}
	return result.String()
}

func TestLegacyRecognitionIsReadOnly(t *testing.T) {
	s, settings, applied := legacyFixture(t)
	c, err := s.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	before := installationSnapshot(t, s)
	for _, provider := range settings.Providers {
		for _, name := range applied[provider] {
			if err := s.validateLegacy(c, provider, name); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, intent := range []Intent{Configure, Refresh, Uninstall} {
		p, err := s.Preview(context.Background(), Request{Intent: intent, Desired: settings})
		if err != nil || len(p.Conflicts) == 0 {
			t.Fatalf("%s accepted legacy: %v %+v", intent, err, p)
		}
		if _, err := s.Apply(context.Background(), p, Confirmations{Removals: true}); err == nil {
			t.Fatalf("%s granted mutation", intent)
		}
	}
	if got := installationSnapshot(t, s); got != before {
		t.Fatal("recognition or ordinary management changed tree")
	}
}

func TestLegacyRejectsUnverifiedTrees(t *testing.T) {
	for _, change := range []string{"marker", "manifest", "missing", "extra-file", "extra-directory", "foreign", "dangling", "resource-file", "directory-link", "parent-link", "manifest-link", "missing-state", "forged-inventory", "foreign-marker", "adapter", "special"} {
		t.Run(change, func(t *testing.T) {
			s, _, _ := legacyFixture(t)
			root, _ := s.destinations("codex")
			path := filepath.Join(root, "python-standards")
			resource := filepath.Join(path, "scripts/nested/run.sh")
			replaceLink := func(path, target string) {
				t.Helper()
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			switch change {
			case "foreign-marker":
				write(t, filepath.Join(path, markerName), "/foreign\nSKILL.md bad\n")
			case "adapter":
				replaceLink(filepath.Join(path, "agents/openai.yaml"), filepath.Join(s.Root, "missing-adapter"))
			case "special":
				if err := unix.Mkfifo(filepath.Join(path, "pipe"), 0600); err != nil {
					t.Fatal(err)
				}
			case "marker":
				write(t, filepath.Join(path, markerName), s.Root+"\nSKILL.md bad\nextra\n")
			case "manifest":
				write(t, filepath.Join(path, "SKILL.md"), "changed")
			case "missing":
				if err := os.Remove(resource); err != nil {
					t.Fatal(err)
				}
			case "extra-file":
				write(t, filepath.Join(path, "user.txt"), "user")
			case "extra-directory":
				if err := os.Mkdir(filepath.Join(path, "unexpected"), 0755); err != nil {
					t.Fatal(err)
				}
			case "foreign":
				target := filepath.Join(t.TempDir(), "foreign")
				write(t, target, "user")
				replaceLink(resource, target)
			case "dangling":
				replaceLink(resource, filepath.Join(s.Root, "missing"))
			case "resource-file":
				if err := os.Remove(resource); err != nil {
					t.Fatal(err)
				}
				write(t, resource, "user")
			case "directory-link":
				replaceLink(filepath.Join(path, "scripts"), t.TempDir())
			case "parent-link":
				if err := os.Rename(root, root+".old"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(root+".old", root); err != nil {
					t.Fatal(err)
				}
			case "manifest-link":
				replaceLink(filepath.Join(path, "SKILL.md"), filepath.Join(s.Root, ".agents/skills/coding/python-standards/SKILL.md"))
			case "missing-state":
				if err := os.Remove(filepath.Join(s.Root, "config/settings.env")); err != nil {
					t.Fatal(err)
				}
			case "forged-inventory":
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
				write(t, filepath.Join(path, "user.txt"), "user")
			}
			before := installationSnapshot(t, s)
			c, err := s.Catalog()
			if err != nil {
				t.Fatal(err)
			}
			if err := s.validateLegacy(c, "codex", "python-standards"); err == nil {
				t.Fatal("accepted invalid legacy installation")
			} else if !strings.Contains(err.Error(), path) && change != "missing-state" {
				t.Fatalf("missing affected path: %v", err)
			}
			if installationSnapshot(t, s) != before {
				t.Fatal("validation mutated files")
			}
		})
	}
}
