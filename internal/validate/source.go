// Package validate checks source contracts and exercises installations in
// disposable homes. It does not invoke Bash, Python, or packaged resources.
package validate

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

var links = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
var remote = regexp.MustCompile(`^(https?://|mailto:)`)

// Source validates registries, skill metadata, links, layout and provider boundaries.
func Source(root string) error {
	c, err := catalog.Load(root)
	if err != nil {
		return err
	}
	for _, name := range []string{".cursor", "config.yml", "requirements.txt"} {
		if _, err := os.Lstat(filepath.Join(c.Root, name)); !os.IsNotExist(err) {
			return fmt.Errorf("unsupported source layout: %s", name)
		}
	}
	guide, err := os.ReadFile(filepath.Join(c.Root, ".agents/global/instructions.md"))
	if err != nil {
		return err
	}
	if !strings.Contains(string(guide), "## Markdown artifact outputs") {
		return fmt.Errorf("missing Markdown artifact output policy")
	}

	err = filepath.WalkDir(filepath.Join(c.Root, ".agents"), func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.Type()&os.ModeSymlink != 0 || (!e.IsDir() && !e.Type().IsRegular()) {
			return fmt.Errorf("unsafe source member: %s", path)
		}
		if e.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(filepath.Join(c.Root, ".agents/skills"), path)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && strings.HasSuffix(filepath.ToSlash(rel), "/agents/openai.yaml") {
			return fmt.Errorf("provider metadata belongs outside skills: %s", path)
		}
		if filepath.Ext(path) == ".md" {
			return Markdown(path, c.Root)
		}
		if filepath.Base(path) == "openai.yaml" {
			parts := strings.Split(filepath.ToSlash(path), "/providers/codex/skills/")
			if len(parts) != 2 {
				return fmt.Errorf("unsupported provider metadata: %s", path)
			}
			key := strings.TrimSuffix(parts[1], "/agents/openai.yaml")
			found := false
			for _, skill := range c.Skills {
				if skill.Key == key && skill.CodexAdapter == path {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("orphan provider adapter: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, skill := range c.Skills {
		data, err := os.ReadFile(filepath.Join(skill.Source, "SKILL.md"))
		if err != nil {
			return err
		}
		if err := metadata(string(data), skill.Name); err != nil {
			return fmt.Errorf("%s: %w", skill.Key, err)
		}
	}
	return nil
}

func metadata(text, name string) error {
	if !strings.HasPrefix(text, "---\n") {
		return fmt.Errorf("missing YAML front matter")
	}
	front, _, ok := strings.Cut(strings.TrimPrefix(text, "---\n"), "\n---\n")
	if !ok {
		return fmt.Errorf("unterminated YAML front matter")
	}
	values := map[string]string{}
	for _, line := range strings.Split(front, "\n") {
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return fmt.Errorf("malformed metadata row")
		}
		if _, exists := values[key]; exists {
			return fmt.Errorf("duplicate metadata: %s", key)
		}
		values[key] = strings.TrimSpace(value)
		if key != "name" && key != "description" {
			return fmt.Errorf("provider-specific or unsupported source metadata: %s", key)
		}
	}
	if values["name"] != name {
		return fmt.Errorf("metadata name must match %s", name)
	}
	if strings.Trim(values["description"], " \"'") == "" {
		return fmt.Errorf("missing metadata description")
	}
	return nil
}

// Markdown checks local links against a filesystem boundary without fetching URLs.
func Markdown(path, boundary string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, match := range links.FindAllStringSubmatch(string(data), -1) {
		ref := strings.Trim(match[1], "<>")
		if strings.HasPrefix(ref, "#") || remote.MatchString(ref) {
			continue
		}
		ref, err = url.PathUnescape(strings.SplitN(ref, "#", 2)[0])
		if err != nil {
			return err
		}
		target := filepath.Join(filepath.Dir(path), ref)
		rel, err := filepath.Rel(boundary, target)
		if err != nil || filepath.IsAbs(ref) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("escaping link in %s: %s", path, ref)
		}
		if _, err := os.Stat(target); err != nil {
			return fmt.Errorf("broken link in %s: %s: %w", path, ref, err)
		}
	}
	return nil
}
