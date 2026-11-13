package validate

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"bytes"
	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
	"github.com/EimantasZilis/agent-toolkit/internal/install"
	"github.com/EimantasZilis/agent-toolkit/internal/render"
)

// Verify validates source and exercises selective/full installs, refresh and
// uninstall through the Go service. All writes stay in a disposable tree.
func Verify(ctx context.Context, root string) error {
	if err := Source(root); err != nil {
		return err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	base, err := os.MkdirTemp("", "agent-toolkit-verify-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(base)

	for _, provider := range []string{"codex", "claude"} {
		for _, selection := range []string{"ticket-to-plan", "all"} {
			if err := ctx.Err(); err != nil {
				return err
			}
			fixture := filepath.Join(base, provider, selection)
			source := filepath.Join(fixture, "toolkit")
			if err := os.CopyFS(filepath.Join(source, ".agents"), os.DirFS(filepath.Join(abs, ".agents"))); err != nil {
				return err
			}
			homes := install.Homes{Agents: filepath.Join(fixture, "agents"), Codex: filepath.Join(fixture, "codex"), Claude: filepath.Join(fixture, "claude")}
			s, err := install.New(source, homes)
			if err != nil {
				return err
			}
			c, err := s.Catalog()
			if err != nil {
				return err
			}
			names, err := c.Select(selection, true)
			if err != nil {
				return err
			}
			guide := filepath.Join(homes.Codex, "AGENTS.md")
			skills := filepath.Join(homes.Agents, "skills/agent-toolkit")
			if provider == "claude" {
				guide = filepath.Join(homes.Claude, "CLAUDE.md")
				skills = filepath.Join(homes.Claude, "skills")
			}
			if err := os.MkdirAll(filepath.Dir(guide), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(guide, []byte("User guidance\n"), 0644); err != nil {
				return err
			}

			desired := install.Settings{Providers: []string{provider}, Requested: append([]string(nil), c.Bundles[selection]...), Concise: true, FullTemplate: true}
			if selection != "all" {
				desired.Requested = []string{selection}
			}
			p, err := s.Preview(ctx, install.Request{Intent: install.Configure, Desired: desired})
			if err != nil {
				return err
			}
			if len(p.Conflicts) > 0 {
				return fmt.Errorf("verification preview conflicts: %v", p.Conflicts)
			}
			status, err := s.Apply(ctx, p, install.Confirmations{Removals: true})
			if err != nil {
				return err
			}
			sort.Strings(names)
			observed := append([]string(nil), status.Observed[provider]...)
			sort.Strings(observed)
			if !reflect.DeepEqual(observed, names) {
				return fmt.Errorf("%s/%s membership mismatch", provider, selection)
			}
			if err := checkInstallation(c, provider, skills, names); err != nil {
				return err
			}
			text, err := os.ReadFile(guide)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(string(text), "User guidance\n") || !strings.Contains(string(text), "# agent-toolkit guidance") {
				return fmt.Errorf("guidance was not preserved and installed: %s", guide)
			}

			p, err = s.Preview(ctx, install.Request{Intent: install.Refresh})
			if err != nil {
				return err
			}
			for _, op := range p.Operations {
				if op.Action != "keep" {
					return fmt.Errorf("unchanged refresh is not stable: %v", op)
				}
			}
			if _, err = s.Apply(ctx, p, install.Confirmations{}); err != nil {
				return err
			}
			p, err = s.Preview(ctx, install.Request{Intent: install.Uninstall})
			if err != nil {
				return err
			}
			if _, err = s.Apply(ctx, p, install.Confirmations{Removals: true}); err != nil {
				return err
			}
			text, err = os.ReadFile(guide)
			if err != nil {
				return err
			}
			if strings.TrimSpace(string(text)) != "User guidance" {
				return fmt.Errorf("uninstall changed user guidance")
			}
			for _, name := range names {
				if _, err := os.Lstat(filepath.Join(skills, name)); !os.IsNotExist(err) {
					return fmt.Errorf("uninstall retained managed skill: %s", name)
				}
			}
		}
	}
	return nil
}

func checkInstallation(c *catalog.Catalog, provider, skills string, names []string) error {
	for _, name := range names {
		skill, err := c.Lookup(name)
		if err != nil {
			return err
		}
		path := filepath.Join(skills, name)
		data, err := os.ReadFile(filepath.Join(path, "SKILL.md"))
		if err != nil {
			return err
		}
		if provider == "claude" && skill.Explicit && !strings.Contains(string(data), "disable-model-invocation: true") {
			return fmt.Errorf("missing Claude policy: %s", name)
		}
		err = filepath.WalkDir(skill.Source, func(source string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() || source == filepath.Join(skill.Source, "SKILL.md") {
				return nil
			}
			rel, err := filepath.Rel(skill.Source, source)
			if err != nil {
				return err
			}
			return checkCopy(c, provider, skills, names, name, rel, filepath.Join(path, rel), source)
		})
		if err != nil {
			return err
		}
		if provider == "codex" && skill.CodexAdapter != "" {
			if err := checkCopy(c, provider, skills, names, name, "agents/openai.yaml", filepath.Join(path, "agents/openai.yaml"), skill.CodexAdapter); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkCopy(c *catalog.Catalog, provider, root string, selected []string, name, rel, destination, source string) error {
	info, err := os.Lstat(destination)
	if err != nil {
		return err
	}
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != sourceInfo.Mode().Perm() {
		return fmt.Errorf("resource is not a regular copy with source modes: %s", destination)
	}
	expected, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if strings.EqualFold(filepath.Ext(source), ".md") {
		text, err := render.Markdown(c, name, rel, string(expected), render.Options{Mode: render.Local, Provider: provider, SkillsRoot: root, Selected: selected})
		if err != nil {
			return err
		}
		expected = []byte(text)
	}
	actual, err := os.ReadFile(destination)
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("resource copy mismatch: %s", destination)
	}
	return nil
}
