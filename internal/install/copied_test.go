package install

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func copiedService(t *testing.T) *Service {
	t.Helper()
	s := testService(t)
	write(t, filepath.Join(s.Root, ".agents/providers/codex/skills/coding/python-standards/agents/openai.yaml"), "interface:\n  display_name: Python fixture\n")
	return s
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func remove(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

func treeDigest(t *testing.T, path string) string {
	t.Helper()
	files, err := readTree(path)
	if err != nil {
		t.Fatal(err)
	}
	return payloadDigest(files)
}

func TestCopiedDevelopmentWorkingTreeAndIndependence(t *testing.T) {
	for _, git := range []bool{false, true} {
		t.Run(map[bool]string{false: "without-git", true: "dirty-git"}[git], func(t *testing.T) {
			s := copiedService(t)
			source := filepath.Join(s.Root, ".agents/skills/coding/python-standards")
			helper := filepath.Join(source, "scripts/fixture.sh")
			write(t, helper, "#!/bin/sh\nprintf 'baseline helper\\n'\n")
			chmod(t, helper, 0751)
			if git {
				for _, args := range [][]string{{"init", "-q"}, {"add", ".agents"}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.com", "commit", "-qm", "baseline"}} {
					cmd := exec.Command("git", args...)
					cmd.Dir = s.Root
					if output, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("git %v: %s %v", args, output, err)
					}
				}
			}
			write(t, helper, "#!/bin/sh\nprintf 'installed helper works\\n'\n")
			manifest := filepath.Join(source, "SKILL.md")
			data, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatal(err)
			}
			write(t, manifest, string(data)+"\nDirty instructions\n[fixture](<references/nested dir/start.md#run>)\n")
			reference := filepath.Join(source, "references/nested dir/start.md")
			write(t, reference, "# Run\n[helper](../../scripts/fixture.sh) [selected](../../../../review/review-branch/SKILL.md#review) [optional](../../../../productivity/pr-comment/SKILL.md)\n")
			write(t, filepath.Join(source, "references/untracked.txt"), "untracked current bytes")
			settings := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"python-standards", "review-branch"}}
			p := previewFor(t, s, Request{Intent: Configure, Desired: settings})
			applyFor(t, s, p)
			before := map[string]string{}
			for _, provider := range settings.Providers {
				root, _ := s.destinations(provider)
				path := filepath.Join(root, "python-standards")
				before[provider] = treeDigest(t, root)
				files, err := readTree(path)
				if err != nil || files["scripts/fixture.sh"].Mode != 0751 || string(files["references/untracked.txt"].Data) != "untracked current bytes" {
					t.Fatalf("%s did not install current bytes/modes: %v", provider, err)
				}
				if !strings.Contains(string(files["SKILL.md"].Data), "Dirty instructions") || !strings.Contains(string(files["SKILL.md"].Data), "<"+filepath.Join(path, "references/nested dir/start.md")+"#run>") {
					t.Fatalf("%s manifest lost edits or spaced link", provider)
				}
				record, ok := s.copiedRecord(path)
				if !ok || record.Source != p.Source {
					t.Fatalf("%s ownership provenance = %+v", provider, record)
				}
			}
			var inv inventory
			invData, err := os.ReadFile(filepath.Join(s.Root, "config/inventory.json"))
			if err != nil || json.Unmarshal(invData, &inv) != nil || inv.Source == nil || *inv.Source != p.Source || inv.Version != 2 {
				t.Fatalf("missing persisted provenance: %s %v", invData, err)
			}
			write(t, manifest, string(data)+"\nFurther dirty instructions\n")
			write(t, reference, "Changed supporting reference\n")
			write(t, helper, "#!/bin/sh\nprintf 'changed source'\n")
			chmod(t, helper, 0644)
			adapter := filepath.Join(s.Root, ".agents/providers/codex/skills/coding/python-standards/agents/openai.yaml")
			write(t, adapter, "interface:\n  display_name: Changed adapter\n")
			for _, provider := range settings.Providers {
				root, _ := s.destinations(provider)
				if treeDigest(t, root) != before[provider] {
					t.Fatalf("%s source edit affected installation", provider)
				}
			}
			// Read installed links and execute only the harmless fixture, with source absent.
			if err := os.Rename(s.Root, s.Root+"-away"); err != nil {
				t.Fatal(err)
			}
			func() {
				defer func() {
					if err := os.Rename(s.Root+"-away", s.Root); err != nil {
						t.Fatal(err)
					}
				}()
				for _, provider := range settings.Providers {
					root, _ := s.destinations(provider)
					path := filepath.Join(root, "python-standards")
					data, err := os.ReadFile(filepath.Join(path, "references/nested dir/start.md"))
					text := string(data)
					installedHelper := filepath.Join(path, "scripts/fixture.sh")
					if err != nil || !strings.Contains(text, "[helper](<"+installedHelper+">)") || !strings.Contains(text, filepath.Join(root, "review-branch/SKILL.md")+"#review") || !strings.Contains(text, "`pr-comment` (optional integration)") || strings.Contains(text, s.Root) {
						t.Fatalf("%s nested links: %s %v", provider, text, err)
					}
					if _, err := os.ReadFile(filepath.Join(root, "review-branch/SKILL.md")); err != nil {
						t.Fatal(err)
					}
					cmd := exec.Command(installedHelper)
					cmd.Dir = t.TempDir()
					if output, err := cmd.CombinedOutput(); err != nil || string(output) != "installed helper works\n" {
						t.Fatalf("installed helper: %s %v", output, err)
					}
					core, err := os.ReadFile(filepath.Join(path, "references/core.md"))
					if err != nil || !strings.Contains(string(core), "python3 '"+filepath.Join(path, "scripts/check_function_size.py")+"'") {
						t.Fatalf("installed checker path: %s %v", core, err)
					}
				}
			}()
			remove(t, filepath.Join(source, "references/untracked.txt"))
			write(t, filepath.Join(source, "references/added.txt"), "added on refresh")
			refresh := previewFor(t, s, Request{Intent: Refresh})
			if refresh.Source.Digest == p.Source.Digest || refresh.Source.Checkout != s.Root || refresh.Source.Kind != "development" {
				t.Fatalf("source provenance did not track working tree: %+v", refresh.Source)
			}
			for _, op := range refresh.Operations {
				if op.Name == "python-standards" && op.Action != "update" {
					t.Fatal(op)
				}
			}
			applyFor(t, s, refresh)
			for _, provider := range settings.Providers {
				root, _ := s.destinations(provider)
				files, err := readTree(filepath.Join(root, "python-standards"))
				_, old := files["references/untracked.txt"]
				if err != nil || old || string(files["references/added.txt"].Data) != "added on refresh" || files["scripts/fixture.sh"].Mode != 0644 {
					t.Fatalf("incomplete %s refresh: %v", provider, err)
				}
				if provider == "codex" && string(files["agents/openai.yaml"].Data) != "interface:\n  display_name: Changed adapter\n" {
					t.Fatal("adapter not refreshed")
				}
			}
			for _, op := range previewFor(t, s, Request{Intent: Refresh}).Operations {
				if op.Action != "keep" {
					t.Fatalf("unchanged refresh: %+v", op)
				}
			}
		})
	}
}

func TestCopiedConflictsPreserveContent(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		for _, change := range []string{"manifest", "resource", "mode", "missing", "extra-file", "extra-directory", "symlink", "record", "record-escape", "adapter"} {
			if provider == "claude" && change == "adapter" {
				continue
			}
			t.Run(provider+"/"+change, func(t *testing.T) {
				s := copiedService(t)
				settings := Settings{Providers: []string{provider}, Requested: []string{"python-standards"}}
				applyFor(t, s, previewFor(t, s, Request{Desired: settings}))
				root, _ := s.destinations(provider)
				path := filepath.Join(root, "python-standards")
				resource := filepath.Join(path, "scripts/check_function_size.py")
				switch change {
				case "manifest":
					write(t, filepath.Join(path, "SKILL.md"), "edited")
				case "resource":
					write(t, resource, "edited")
				case "adapter":
					write(t, filepath.Join(path, "agents/openai.yaml"), "edited")
				case "mode":
					info, err := os.Stat(resource)
					if err != nil {
						t.Fatal(err)
					}
					chmod(t, resource, info.Mode().Perm()^0100)
				case "missing":
					remove(t, resource)
				case "extra-file":
					write(t, filepath.Join(path, "foreign"), "preserve me")
				case "extra-directory":
					if err := os.Mkdir(filepath.Join(path, "foreign"), 0755); err != nil {
						t.Fatal(err)
					}
				case "symlink":
					remove(t, resource)
					if err := os.Symlink("/missing", resource); err != nil {
						t.Fatal(err)
					}
				case "record":
					write(t, filepath.Join(path, markerName), `{"schema_version":99}`)
				case "record-escape":
					record, ok := s.copiedRecord(path)
					if !ok {
						t.Fatal("missing initial record")
					}
					record.Files[0].Path = "../escape"
					data, err := json.Marshal(record)
					if err != nil {
						t.Fatal(err)
					}
					write(t, filepath.Join(path, markerName), string(data))
				}
				before, err := s.revision()
				if err != nil {
					t.Fatal(err)
				}
				for _, intent := range []Intent{Configure, Uninstall} {
					p, err := s.Preview(context.Background(), Request{Intent: intent, Desired: settings})
					if err != nil || len(p.Conflicts) == 0 {
						t.Fatalf("%s failed to conflict: %v", intent, err)
					}
					if _, err := s.Apply(context.Background(), p, Confirmations{Removals: true}); err == nil {
						t.Fatal("conflicted content was accepted")
					}
				}
				after, err := s.revision()
				if err != nil || after != before {
					t.Fatalf("conflict changed content: %v", err)
				}
			})
		}
	}
}

func TestCopiedSourceSafetyAndStaleness(t *testing.T) {
	for _, change := range []string{"symlink", "fifo", "privileged-mode", "adapter-collision", "source-bytes", "source-mode", "destination-mode", "during-payload"} {
		t.Run(change, func(t *testing.T) {
			s := copiedService(t)
			settings := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"python-standards"}}
			applyFor(t, s, previewFor(t, s, Request{Desired: settings}))
			p := previewFor(t, s, Request{Intent: Refresh})
			source := filepath.Join(s.Root, ".agents/skills/coding/python-standards")
			switch change {
			case "symlink":
				if err := os.Symlink("/missing", filepath.Join(source, "link")); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := unix.Mkfifo(filepath.Join(source, "pipe"), 0600); err != nil {
					t.Fatal(err)
				}
			case "privileged-mode":
				chmod(t, filepath.Join(source, "scripts/check_function_size.py"), os.ModeSetuid|0755)
			case "adapter-collision":
				write(t, filepath.Join(source, "agents/openai.yaml"), "collision")
			case "source-bytes":
				write(t, filepath.Join(source, "references/added.txt"), "changed")
			case "source-mode":
				chmod(t, filepath.Join(source, "scripts/check_function_size.py"), 0700)
			case "destination-mode":
				root, _ := s.destinations("claude")
				chmod(t, filepath.Join(root, "python-standards/SKILL.md"), 0600)
			case "during-payload":
				s.fail = func(at string) error {
					if at == "after-payload" {
						return os.WriteFile(filepath.Join(source, "references/raced.txt"), []byte("changed during preparation"), 0644)
					}
					return nil
				}
			}
			if change == "symlink" || change == "fifo" || change == "privileged-mode" || change == "adapter-collision" || change == "during-payload" {
				if _, err := s.Preview(context.Background(), Request{Intent: Refresh}); err == nil {
					t.Fatal("unsafe or racing source accepted")
				}
			}
			before, err := s.revision()
			if err != nil {
				t.Fatal(err)
			}
			destinations := map[string]string{}
			for _, provider := range settings.Providers {
				root, _ := s.destinations(provider)
				destinations[root] = treeDigest(t, root)
			}
			state := treeDigest(t, filepath.Join(s.Root, "config"))
			if _, err := s.Apply(context.Background(), p, Confirmations{}); err == nil {
				t.Fatal("stale or unsafe apply accepted")
			}
			for root, want := range destinations {
				if treeDigest(t, root) != want {
					t.Fatal("rejected apply changed provider files")
				}
			}
			if treeDigest(t, filepath.Join(s.Root, "config")) != state {
				t.Fatal("rejected apply changed saved state")
			}
			// The racing hook may change source again, so compare its revision separately.
			if change != "during-payload" {
				after, err := s.revision()
				if err != nil || before != after {
					t.Fatalf("rejected apply changed data: %v", err)
				}
			}
		})
	}
}

func TestCopiedLegacyRecognitionWithoutConversion(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			s := copiedService(t)
			settings := Settings{Providers: []string{provider}, Requested: []string{"python-standards"}, FullTemplate: true}
			installedFixture(t, s, provider, "python-standards", settings, settings.Requested)
			saveFixture(t, s, settings, map[string][]string{"codex": {}, "claude": {}, provider: {"python-standards"}})
			status, err := s.Status(context.Background())
			if err != nil || !contains(status.Observed[provider], "python-standards") {
				t.Fatalf("legacy not recognized: %+v %v", status, err)
			}
			before, err := s.revision()
			if err != nil {
				t.Fatal(err)
			}
			for _, intent := range []Intent{Configure, Refresh, Uninstall} {
				p, err := s.Preview(context.Background(), Request{Intent: intent, Desired: settings})
				if err != nil || len(p.Conflicts) == 0 {
					t.Fatalf("legacy mutation allowed: %s %v", intent, err)
				}
				if _, err := s.Apply(context.Background(), p, Confirmations{Removals: true}); err == nil {
					t.Fatal("converted legacy content")
				}
			}
			after, err := s.revision()
			if err != nil || before != after {
				t.Fatalf("legacy state changed: %v", err)
			}
		})
	}
}

func TestCopiedFreshAndRefreshRollback(t *testing.T) {
	for _, refresh := range []bool{false, true} {
		for _, point := range []string{"after-provider:codex", "after-provider:claude", "after-settings", "after-inventory"} {
			name := "fresh/"
			if refresh {
				name = "refresh/"
			}
			t.Run(name+point, func(t *testing.T) {
				s := copiedService(t)
				settings := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"python-standards"}, Concise: true, FullTemplate: true}
				for _, provider := range settings.Providers {
					_, guide := s.destinations(provider)
					write(t, guide, "User guidance\n")
				}
				request := Request{Desired: settings}
				if refresh {
					applyFor(t, s, previewFor(t, s, request))
					write(t, filepath.Join(s.Root, ".agents/skills/coding/python-standards/references/new.txt"), "dirty resource")
					chmod(t, filepath.Join(s.Root, ".agents/skills/coding/python-standards/scripts/check_function_size.py"), 0700)
					request = Request{Intent: Refresh}
				}
				p := previewFor(t, s, request)
				before, err := s.revision()
				if err != nil {
					t.Fatal(err)
				}
				s.fail = func(at string) error {
					if at == point {
						return errors.New("injected copied transaction failure")
					}
					return nil
				}
				if _, err := s.Apply(context.Background(), p, Confirmations{}); err == nil {
					t.Fatal("expected failure")
				}
				after, err := s.revision()
				if err != nil || before != after {
					t.Fatalf("rollback lost bytes, modes, guidance, ownership or state: %v", err)
				}
			})
		}
	}
}

func TestCopiedModeOnlyRefreshAndSourceDigest(t *testing.T) {
	s := copiedService(t)
	settings := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"python-standards"}}
	initial := previewFor(t, s, Request{Desired: settings})
	applyFor(t, s, initial)
	previous := initial.Source.Digest
	changes := []struct {
		name string
		edit func()
	}{
		{"mode", func() {
			chmod(t, filepath.Join(s.Root, ".agents/skills/coding/python-standards/scripts/check_function_size.py"), 0700)
		}},
		{"resource", func() {
			write(t, filepath.Join(s.Root, ".agents/skills/coding/python-standards/references/new.txt"), "new")
		}},
		{"adapter", func() {
			write(t, filepath.Join(s.Root, ".agents/providers/codex/skills/coding/python-standards/agents/openai.yaml"), "interface:\n  display_name: New\n")
		}},
	}
	for _, change := range changes {
		t.Run(change.name, func(t *testing.T) {
			change.edit()
			p := previewFor(t, s, Request{Intent: Refresh})
			if previous == p.Source.Digest {
				t.Fatal("source digest ignored changed input")
			}
			for _, op := range p.Operations {
				if op.Name != "python-standards" {
					continue
				}
				want := "update"
				if change.name == "adapter" && op.Provider == "claude" {
					want = "keep"
				}
				if op.Action != want {
					t.Fatalf("%+v want %s", op, want)
				}
			}
			applyFor(t, s, p)
			previous = p.Source.Digest
		})
	}
	// State and Git metadata are outside the complete .agents input digest.
	write(t, filepath.Join(s.Root, "config/unrelated"), "state")
	write(t, filepath.Join(s.Root, ".git/unrelated"), "git metadata")
	_, source, err := s.captureSource()
	if err != nil || source.Digest != previous {
		t.Fatalf("provenance included non-source files: %+v %v", source, err)
	}
}

func TestCopiedDestinationBoundarySubstitution(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			s := copiedService(t)
			settings := Settings{Providers: []string{provider}, Requested: []string{"python-standards"}}
			applyFor(t, s, previewFor(t, s, Request{Desired: settings}))
			p := previewFor(t, s, Request{Intent: Refresh})
			root, _ := s.destinations(provider)
			if err := os.Rename(root, root+"-original"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(root+"-original", root); err != nil {
				t.Fatal(err)
			}
			before := treeDigest(t, root+"-original")
			current, err := s.Preview(context.Background(), Request{Intent: Refresh})
			if err != nil || len(current.Conflicts) == 0 {
				t.Fatalf("substituted provider boundary accepted: %v", err)
			}
			if _, err := s.Apply(context.Background(), p, Confirmations{}); err == nil {
				t.Fatal("stale boundary accepted")
			}
			if treeDigest(t, root+"-original") != before {
				t.Fatal("substituted boundary content changed")
			}
		})
	}
}
