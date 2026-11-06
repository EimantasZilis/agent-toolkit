package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

func TestRenderingContracts(t *testing.T) {
	c, err := catalog.Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	skill, err := c.Lookup("next-phase")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(skill.Source, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []Mode{Local, Native} {
		t.Run(string(mode), func(t *testing.T) {
			got, err := Markdown(c, "next-phase", "SKILL.md", string(source), Options{Mode: mode, Provider: "claude", Package: "starter", SkillsRoot: "/installed/skills", Selected: []string{"next-phase"}})
			if err != nil {
				t.Fatal(err)
			}
			got = strings.ReplaceAll(got, c.Root, "<checkout>")
			want, err := os.ReadFile("testdata/" + string(mode) + "-next-phase.md")
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("render differs from contract fixture\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestLinksAndInvocations(t *testing.T) {
	c, err := catalog.Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	text := "[review](../../review/review-branch/SKILL.md) [commit](../../productivity/commit-message/SKILL.md) $review-branch $commit-message $unknown [site](https://example.com) [anchor](#anchor)"
	for _, tc := range []struct {
		mode     Mode
		provider string
		wants    []string
	}{
		{Local, "claude", []string{"[review](</installed/skills/review-branch/SKILL.md>)", "/review-branch", "/commit-message", "$unknown"}},
		{Native, "claude", []string{"[review](../review-branch/SKILL.md)", "`commit-message` (optional integration)", "/starter:review-branch", "`commit-message`", "$unknown"}},
		{Native, "codex", []string{"$review-branch", "$commit-message", "[review](../review-branch/SKILL.md)"}},
	} {
		got, err := Markdown(c, "next-phase", "SKILL.md", text, Options{Mode: tc.mode, Provider: tc.provider, Package: "starter", SkillsRoot: "/installed/skills", Selected: []string{"next-phase", "review-branch"}})
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range append(tc.wants, "[site](https://example.com)", "[anchor](#anchor)") {
			if !strings.Contains(got, want) {
				t.Fatalf("%s/%s missing %q: %s", tc.mode, tc.provider, want, got)
			}
		}
		if tc.mode == Native && strings.Contains(got, c.Root) {
			t.Fatal("checkout path in native output")
		}
	}
}

func TestSupportingMarkdownAndOutputPolicy(t *testing.T) {
	c, err := catalog.Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	text := ".agents/skills/coding/python-standards/scripts/check_function_size.py"
	for _, mode := range []Mode{Local, Native} {
		got, err := Markdown(c, "python-standards", "references/code.md", text, Options{Mode: mode, Provider: "codex", SkillsRoot: "/package/skills"})
		if err != nil {
			t.Fatal(err)
		}
		if mode == Local && got != text {
			t.Fatal("changed linked supporting Markdown")
		}
		if mode == Native && !strings.Contains(got, `"<loaded-reference-directory>/../scripts/check_function_size.py"`) {
			t.Fatal(got)
		}
	}
	got, err := Markdown(c, "next-phase", "SKILL.md", "---\nname: next-phase\n---\n", Options{Mode: Local, Provider: "claude", OutputRoot: "/shared output"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "disable-model-invocation: true") || !strings.Contains(got, "`/shared output/<repo-slug>/<branch-slug>/<skill-name>/...`") {
		t.Fatal(got)
	}
	for _, opts := range []Options{{Mode: "bad", Provider: "codex"}, {Mode: Local, Provider: "bad"}, {Mode: Native, Provider: "codex", Selected: []string{"unknown"}}} {
		if _, err := Markdown(c, "next-phase", "SKILL.md", "", opts); err == nil {
			t.Fatal("accepted invalid options")
		}
	}
}
