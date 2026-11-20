package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndependentDocumentationSettings(t *testing.T) {
	for _, coding := range []bool{false, true} {
		for _, documentation := range []bool{false, true} {
			s := testService(t)
			desired := Settings{Providers: []string{"codex", "claude"}, FullTemplate: coding, Documentation: &documentation}
			status := applyFor(t, s, previewFor(t, s, Request{Desired: desired}))
			status = applyFor(t, s, previewFor(t, s, Request{Intent: Refresh}))
			for _, id := range desired.Providers {
				if status.Guidance[id].FullTemplate != coding || status.Guidance[id].Documentation != documentation || status.Desired.ForProvider(id).DocumentationEnabled() != documentation {
					t.Fatalf("independent settings lost: %+v", status)
				}
			}
		}
	}
}

func TestGuidanceReferencesPersistentSettings(t *testing.T) {
	s := testService(t)
	desired := Settings{Providers: []string{"codex", "claude"}, FullTemplate: true, Concise: true, SharedStorage: true}
	p := previewFor(t, s, Request{Desired: desired})
	applyFor(t, s, p)

	for _, id := range desired.Providers {
		_, path := s.destinations(id)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text, "load and apply") || !strings.Contains(text, "Before answering or working") {
			t.Fatal("missing mandatory settings load", text)
		}
		for _, name := range []string{"instructions.md", "documentation.md", "markdown-output.md"} {
			target := filepath.Join(s.Root, ".agents/global", name)
			if !strings.Contains(text, "(<"+target+">)") {
				t.Fatal("link does not resolve to persistent checkout", text)
			}
			if _, err := os.ReadFile(target); err != nil {
				t.Fatal("linked setting unavailable after preview", err)
			}
		}
		skills, _ := s.destinations(id)
		if guidanceTarget(text, "Concise output") != filepath.ToSlash(filepath.Join(skills, "concise-output/SKILL.md")) {
			t.Fatal("concise route does not target the installed skill", text)
		}
		if _, err := os.ReadFile(guidanceTarget(text, "Concise output")); err != nil {
			t.Fatal("installed concise route unavailable", err)
		}
		if strings.Contains(text, "## Documentation") || strings.Contains(text, "invoke the /concise-output") {
			t.Fatal("global file contains copied settings", text)
		}
	}

	write(t, filepath.Join(s.Root, ".agents/global/instructions.md"), "Updated standing guidance\n")
	refresh := previewFor(t, s, Request{Intent: Refresh})
	for _, op := range refresh.Operations {
		if op.Name == "guidance" {
			t.Fatal("editing linked guidance rewrote provider files", op)
		}
	}
	status, err := s.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range desired.Providers {
		guide := status.Guidance[id]
		if !guide.FullTemplate || !guide.Concise || !guide.Output || guide.State != "Installed" {
			t.Fatal("linked settings not detected", guide)
		}
	}
}

func TestGlobalSettingsWithoutPublicSkills(t *testing.T) {
	for _, external := range []bool{false, true} {
		for _, concise := range []bool{false, true} {
			for _, coding := range []bool{false, true} {
				s := testService(t)
				desired := Settings{Providers: []string{"codex", "claude"}, SharedStorage: true, SeparateByProvider: true, Concise: concise, FullTemplate: coding}
				if external {
					desired.Output = filepath.Join(s.Root, "outputs")
				}
				status := applyFor(t, s, previewFor(t, s, Request{Desired: desired}))
				for _, id := range desired.Providers {
					guide := status.Guidance[id]
					if !guide.Output || guide.Concise != concise || guide.FullTemplate != coding || !strings.HasSuffix(guide.Location, "/<skill-name>/<branch-slug>/"+id+"/") {
						t.Fatalf("wrong effective settings: %+v", guide)
					}
					wantSkills := 0
					if concise {
						wantSkills = 1
					}
					if len(status.Observed[id]) != wantSkills {
						t.Fatal("internal settings exposed as skills", status.Observed)
					}
				}
				applyFor(t, s, previewFor(t, s, Request{Intent: Refresh}))
			}
		}
	}
}

func TestMissingGlobalModuleFailsPreview(t *testing.T) {
	for _, name := range []string{"instructions.md", "documentation.md", "markdown-output.md"} {
		t.Run(name, func(t *testing.T) {
			s := testService(t)
			remove(t, filepath.Join(s.Root, ".agents/global", name))
			_, err := s.Preview(context.Background(), Request{Desired: Settings{Providers: []string{"codex"}, FullTemplate: true}})
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("missing global module did not fail preview clearly", err)
			}
			if exists(s.Homes.Codex) || exists(filepath.Join(s.Root, "config")) {
				t.Fatal("failed preview changed the installation")
			}
		})
	}
}

func TestGuidanceSelectedReferences(t *testing.T) {
	s := testService(t)
	for _, tc := range []struct {
		name     string
		settings Settings
		files    []string
	}{
		{"storage only", Settings{}, []string{"markdown-output.md"}},
		{"concise", Settings{Concise: true}, []string{"markdown-output.md", "concise-output/SKILL.md"}},
		{"storage", Settings{Output: "/tmp/artifacts"}, []string{"markdown-output.md"}},
		{"general", Settings{FullTemplate: true}, []string{"instructions.md", "documentation.md", "markdown-output.md"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			block, err := s.guidanceBlockFrom(s.Root, "codex", tc.settings)
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.files) == 0 && block != "" {
				t.Fatal("disabled settings wrote guidance")
			}
			if strings.Count(block, "](<") != len(tc.files) {
				t.Fatal("unselected settings linked", block)
			}
			for _, name := range tc.files {
				if !strings.Contains(block, "/"+name+">)") {
					t.Fatal("missing selected setting", name)
				}
			}
		})
	}
	remove(t, filepath.Join(s.Root, ".agents/skills/productivity/concise-output/SKILL.md"))
	if _, err := s.guidanceBlockFrom(s.Root, "codex", Settings{Concise: true}); err == nil {
		t.Fatal("missing selected setting accepted")
	}
}

func TestRefreshReplacesInlineGuidanceWithReferences(t *testing.T) {
	s := testService(t)
	desired := Settings{Providers: []string{"codex", "claude"}, FullTemplate: true, Concise: true}
	applyFor(t, s, previewFor(t, s, Request{Desired: desired}))
	legacy := blockStart + "\n" + notice + "\n# agent-toolkit guidance\n## Documentation\nOld inline policy\n" + conciseStart + "\nOld concise policy\n" + conciseEnd + "\n" + blockEnd + "\n"
	for _, id := range desired.Providers {
		_, path := s.destinations(id)
		write(t, path, "User guidance\n"+legacy)
		write(t, filepath.Join(filepath.Dir(path), markerName), s.Root+"\nGUIDANCE_BLOCK "+digest([]byte(legacy))+"\n")
	}
	p := previewFor(t, s, Request{Intent: Refresh})
	for _, id := range desired.Providers {
		change := reviewEntry(t, p, id, "guidance", ".md")
		if change.Kind != "modified" {
			t.Fatal("inline guidance upgrade missing", change)
		}
	}
	applyFor(t, s, p)
	for _, id := range desired.Providers {
		_, path := s.destinations(id)
		data, err := os.ReadFile(path)
		if err != nil || !strings.HasPrefix(string(data), "User guidance\n") || strings.Contains(string(data), "Old inline policy") || !strings.Contains(string(data), "[Coding guidance](") {
			t.Fatalf("upgrade lost user content or retained inline settings: %s %v", data, err)
		}
	}
}
