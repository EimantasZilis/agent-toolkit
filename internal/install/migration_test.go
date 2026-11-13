package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func migrationFixture(t *testing.T) (*Service, Settings) {
	t.Helper()
	s, settings, _ := legacyFixture(t)
	block, err := guidanceBlock(s.Root, settings)
	if err != nil {
		t.Fatal(err)
	}
	for _, provider := range settings.Providers {
		_, guidance := s.destinations(provider)
		write(t, guidance, "User guidance before\n\n"+block+"\nUser guidance after\n")
		write(t, filepath.Join(filepath.Dir(guidance), markerName), s.Root+"\nGUIDANCE_BLOCK "+digest([]byte(block))+"\n")
	}
	return s, settings
}

func TestMigrationPreservesPreferencesAndCopies(t *testing.T) {
	s, settings := migrationFixture(t)
	before := installationSnapshot(t, s)
	p := previewFor(t, s, Request{Intent: Migrate, Desired: Settings{Providers: []string{"claude"}}, Removed: []string{"python-standards"}})
	if string(settingsBytes(p.Desired)) != string(settingsBytes(settings)) {
		t.Fatalf("preferences changed: %+v", p.Desired)
	}
	conversions := 0
	for _, op := range p.Operations {
		if op.Action == "convert" {
			conversions++
		}
	}
	if conversions != 2*len(p.Selection) {
		t.Fatalf("missing conversions: %+v", p.Operations)
	}
	if installationSnapshot(t, s) != before {
		t.Fatal("preview/cancel mutated installation")
	}
	sourceBefore := treeDigest(t, filepath.Join(s.Root, ".agents"))
	status := applyFor(t, s, p)
	if string(settingsBytes(status.Desired)) != string(settingsBytes(settings)) {
		t.Fatalf("saved preferences changed: %+v", status.Desired)
	}
	for _, provider := range settings.Providers {
		root, guidance := s.destinations(provider)
		for _, item := range p.Selection {
			path := filepath.Join(root, item.Name)
			if !s.safeCopied(path) {
				t.Fatalf("not copied: %s", path)
			}
			record, ok := s.copiedRecord(path)
			if !ok || record.Source != p.Source {
				t.Fatalf("bad provenance: %+v", record)
			}
		}
		files, err := readTree(filepath.Join(root, "python-standards"))
		if err != nil {
			t.Fatal(err)
		}
		if files["scripts/nested/run.sh"].Mode != 0755 {
			t.Fatal("executable mode lost")
		}
		if provider == "codex" && len(files["agents/openai.yaml"].Data) == 0 {
			t.Fatal("adapter missing")
		}
		data, err := os.ReadFile(guidance)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "User guidance before") || !strings.Contains(string(data), "User guidance after") {
			t.Fatal("user guidance lost")
		}
	}
	if treeDigest(t, filepath.Join(s.Root, ".agents")) != sourceBefore {
		t.Fatal("migration mutated checkout")
	}
	// Editing a checkout resource has no immediate effect on copied installs.
	root, _ := s.destinations("codex")
	installed := treeDigest(t, root)
	write(t, filepath.Join(s.Root, ".agents/skills/coding/python-standards/scripts/nested/run.sh"), "changed source")
	if treeDigest(t, root) != installed {
		t.Fatal("source edit changed installed copy")
	}
	again := previewFor(t, s, Request{Intent: Migrate})
	for _, op := range again.Operations {
		if op.Action != "keep" {
			t.Fatalf("repeat migration changed copies: %+v", op)
		}
	}
	applyFor(t, s, again)
	applyFor(t, s, previewFor(t, s, Request{Intent: Refresh}))
	applyFor(t, s, previewFor(t, s, Request{Intent: Uninstall}))
	if _, err := os.Stat(filepath.Join(root, "python-standards")); !os.IsNotExist(err) {
		t.Fatal("uninstall left skill")
	}
}

func TestMigrationRefusesWholeInstallation(t *testing.T) {
	for _, change := range []string{"manifest", "extra", "foreign", "missing", "settings", "dependency", "unavailable", "forged"} {
		t.Run(change, func(t *testing.T) {
			s, settings := migrationFixture(t)
			root, _ := s.destinations("claude")
			path := filepath.Join(root, "python-standards")
			switch change {
			case "manifest":
				write(t, filepath.Join(path, "SKILL.md"), "changed")
			case "extra":
				write(t, filepath.Join(path, "user.txt"), "user")
			case "foreign":
				remove(t, filepath.Join(path, "scripts/nested/run.sh"))
				if err := os.Symlink("/foreign", filepath.Join(path, "scripts/nested/run.sh")); err != nil {
					t.Fatal(err)
				}
			case "missing":
				remove(t, filepath.Join(path, "scripts/nested/run.sh"))
			case "settings":
				write(t, filepath.Join(s.Root, "config/settings.env"), string(settingsBytes(Settings{Providers: settings.Providers})))
			case "dependency":
				f := filepath.Join(s.Root, ".agents/dependencies.txt")
				data, err := os.ReadFile(f)
				if err != nil {
					t.Fatal(err)
				}
				write(t, f, string(data)+"\npython-standards ruby-standards\n")
			case "unavailable":
				settings.Requested = append(settings.Requested, "unavailable")
				saveFixture(t, s, settings, map[string][]string{"codex": {}, "claude": {}})
			case "forged":
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
				write(t, filepath.Join(path, "user.txt"), "unowned user content")
			}
			before := installationSnapshot(t, s)
			p, err := s.Preview(context.Background(), Request{Intent: Migrate})
			if err == nil && len(p.Conflicts) == 0 {
				t.Fatal("invalid installation accepted")
			}
			if err == nil {
				if _, err := s.Apply(context.Background(), p, Confirmations{}); err == nil {
					t.Fatal("conflicted migration applied")
				}
			}
			if installationSnapshot(t, s) != before {
				t.Fatal("refusal mutated either provider or state")
			}
		})
	}
}

func TestMigrationStaleInputs(t *testing.T) {
	for _, change := range []string{"settings", "inventory", "manifest", "link", "bytes", "mode", "adapter", "guidance", "source-manifest"} {
		t.Run(change, func(t *testing.T) {
			s, _ := migrationFixture(t)
			p := previewFor(t, s, Request{Intent: Migrate})
			root, guidance := s.destinations("codex")
			resource := filepath.Join(s.Root, ".agents/skills/coding/python-standards/scripts/nested/run.sh")
			switch change {
			case "settings":
				f := filepath.Join(s.Root, "config/settings.env")
				data, err := os.ReadFile(f)
				if err != nil {
					t.Fatal(err)
				}
				write(t, f, strings.Replace(string(data), "GLOBAL_GUIDANCE=true", "GLOBAL_GUIDANCE=false", 1))
			case "inventory":
				f := filepath.Join(s.Root, "config/inventory.json")
				data, err := os.ReadFile(f)
				if err != nil {
					t.Fatal(err)
				}
				write(t, f, string(data)+" ")
			case "manifest":
				write(t, filepath.Join(root, "python-standards/SKILL.md"), "changed manifest")
			case "link":
				f := filepath.Join(root, "python-standards/scripts/nested/run.sh")
				remove(t, f)
				if err := os.Symlink("/foreign", f); err != nil {
					t.Fatal(err)
				}
			case "bytes":
				write(t, resource, "changed bytes")
			case "mode":
				chmod(t, resource, 0700)
			case "adapter":
				write(t, filepath.Join(s.Root, ".agents/providers/codex/skills/coding/python-standards/agents/openai.yaml"), "changed adapter")
			case "guidance":
				write(t, guidance, "changed user guidance")
			case "source-manifest":
				f := filepath.Join(s.Root, ".agents/skills/coding/python-standards/SKILL.md")
				data, err := os.ReadFile(f)
				if err != nil {
					t.Fatal(err)
				}
				write(t, f, string(data)+"\nNew instructions\n")
			}
			before := installationSnapshot(t, s)
			if _, err := s.Apply(context.Background(), p, Confirmations{}); err == nil {
				t.Fatal("stale migration accepted")
			}
			if installationSnapshot(t, s) != before {
				t.Fatal("stale apply changed trees")
			}
		})
	}
}

func TestMigrationRollbackAndCancellation(t *testing.T) {
	for _, point := range []string{"after-provider:codex", "after-provider:claude", "after-settings", "after-inventory", "cancel-before", "cancel-during"} {
		t.Run(point, func(t *testing.T) {
			s, _ := migrationFixture(t)
			// Force a managed guidance refresh so rollback covers changed guidance too.
			f := filepath.Join(s.Root, ".agents/global/instructions.md")
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			write(t, f, string(data)+"\nNew source guidance.\n")
			p := previewFor(t, s, Request{Intent: Migrate})
			before := installationSnapshot(t, s)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sentinel := errors.New("injected migration failure")
			if point == "cancel-before" {
				cancel()
			}
			s.fail = func(at string) error {
				if at == point {
					return sentinel
				}
				if point == "cancel-during" && at == "after-provider:codex" {
					cancel()
				}
				return nil
			}
			_, err = s.Apply(ctx, p, Confirmations{})
			if !errors.Is(err, sentinel) && !errors.Is(err, context.Canceled) {
				t.Fatalf("wrong failure: %v", err)
			}
			if installationSnapshot(t, s) != before {
				t.Fatal("rollback failed to restore exact links, modes, guidance and state")
			}
		})
	}
}

func TestMigrationMixedCopiedAndLinked(t *testing.T) {
	s, settings := migrationFixture(t)
	// Materialize one independently verified copied skill in the mixed tree.
	c, err := s.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	root, _ := s.destinations("codex")
	path := filepath.Join(root, "commit-message")
	files, err := skillPayload(c, "commit-message", "codex", root, settings, settings.Requested)
	if err != nil {
		t.Fatal(err)
	}
	_, source, err := s.captureSource()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if err := writePayload(path, files); err != nil {
		t.Fatal(err)
	}
	marker, err := ownershipBytes(files, source)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(path, markerName), string(marker))
	before := treeDigest(t, path)
	unrelated := filepath.Join(root, "unrelated-user-skill/SKILL.md")
	write(t, unrelated, "User skill")
	p := previewFor(t, s, Request{Intent: Migrate})
	for _, op := range p.Operations {
		if op.Path == path && op.Action != "keep" {
			t.Fatal("migration changed copied skill")
		}
	}
	applyFor(t, s, p)
	if treeDigest(t, path) != before {
		t.Fatal("copied skill changed")
	}
	data, err := os.ReadFile(unrelated)
	if err != nil || string(data) != "User skill" {
		t.Fatal("unrelated skill changed")
	}
	// Copied conflicts still block a mixed migration.
	s, _ = migrationFixture(t)
	root, _ = s.destinations("codex")
	path = filepath.Join(root, "commit-message")
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if err := writePayload(path, files); err != nil {
		t.Fatal(err)
	}
	// A record from another checkout cannot authorize this directory.
	write(t, filepath.Join(path, markerName), string(marker))
	beforeTree := installationSnapshot(t, s)
	if _, err := s.Preview(context.Background(), Request{Intent: Migrate}); err == nil {
		t.Fatal("foreign copied record accepted")
	}
	if installationSnapshot(t, s) != beforeTree {
		t.Fatal("mixed conflict changed trees")
	}
}

func TestMigrationFailedRecoveryRetainsLinks(t *testing.T) {
	s, _ := migrationFixture(t)
	p := previewFor(t, s, Request{Intent: Migrate})
	root, _ := s.destinations("codex")
	path := filepath.Join(root, "python-standards")
	var before strings.Builder
	if err := fingerprint(path, &before); err != nil {
		t.Fatal(err)
	}
	s.fail = func(at string) error {
		if at == "after-inventory" || at == "rollback:"+path {
			return errors.New("injected recovery failure")
		}
		return nil
	}
	_, err := s.Apply(context.Background(), p, Confirmations{})
	recovery := recoveryRoot(t, err)
	entries, err := os.ReadDir(recovery)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		candidate := filepath.Join(recovery, entry.Name())
		link, err := os.Readlink(filepath.Join(candidate, "scripts/nested/run.sh"))
		if err == nil {
			if link != filepath.Join(s.Root, ".agents/skills/coding/python-standards/scripts/nested/run.sh") {
				t.Fatal("backup changed link identity")
			}
			if err := os.RemoveAll(path); err != nil {
				t.Fatal(err)
			}
			if err := copyPath(candidate, path); err != nil {
				t.Fatal(err)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("recovery lost legacy resource links")
	}
	var restored strings.Builder
	if err := fingerprint(path, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.String() != before.String() {
		t.Fatal("retained backup did not restore exact tree")
	}
}
