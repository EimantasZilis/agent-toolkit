// Package catalog reads provider-neutral skill identities and selection registries.
// Loading and selection are read-only and never execute source content.
package catalog

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var identity = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Skill identifies a source directory and its provider metadata.
type Skill struct {
	Name, Category, Key, Source, CodexAdapter string
	Helpers                                   []string
	Output, Explicit                          bool
}

// Catalog holds source order (category then name) and validated registries.
// Callers must not mutate its skills or maps while sharing it.
type Catalog struct {
	Root    string
	Skills  []Skill
	Bundles map[string][]string
	byName  map[string]int
}

// Load validates identities, bundles, dependencies and provider registries.
func Load(root string) (*Catalog, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	c := &Catalog{Root: root, Bundles: map[string][]string{}, byName: map[string]int{}}
	paths, err := filepath.Glob(filepath.Join(root, ".agents/skills/*/*/SKILL.md"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("empty source catalog")
	}
	for _, path := range paths {
		source := filepath.Dir(path)
		name, category := filepath.Base(source), filepath.Base(filepath.Dir(source))
		if !identity.MatchString(name) || !identity.MatchString(category) {
			return nil, fmt.Errorf("invalid identity: %s/%s", category, name)
		}
		if _, ok := c.byName[name]; ok {
			return nil, fmt.Errorf("ambiguous skill: %s", name)
		}
		key := category + "/" + name
		c.byName[name] = len(c.Skills)
		c.Skills = append(c.Skills, Skill{Name: name, Category: category, Key: key, Source: source})
		c.Bundles[category] = append(c.Bundles[category], name)
		c.Bundles["all"] = append(c.Bundles["all"], name)
	}
	if err := c.loadRegistries(); err != nil {
		return nil, err
	}
	return c, nil
}

func readRows(path string, comments bool) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows [][]string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if comments && (strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#")) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			return nil, fmt.Errorf("empty registry entry in %s", path)
		}
		rows = append(rows, fields)
	}
	return rows, s.Err()
}

func (c *Catalog) loadRegistries() error {
	for _, registry := range []string{"bundles.txt", "dependencies.txt", "markdown-output-skills.txt", "providers/claude/explicit-only.txt"} {
		rows, err := readRows(filepath.Join(c.Root, ".agents", registry), registry == "bundles.txt" || registry == "dependencies.txt")
		if err != nil {
			return fmt.Errorf("read %s: %w", registry, err)
		}
		seen := map[string]bool{}
		for _, row := range rows {
			name := row[0]
			if seen[name] {
				return fmt.Errorf("duplicate %s entry: %s", registry, name)
			}
			seen[name] = true
			switch registry {
			case "bundles.txt":
				if !identity.MatchString(name) || c.Bundles[name] != nil || len(row) < 2 {
					return fmt.Errorf("invalid bundle: %s", name)
				}
				for _, member := range row[1:] {
					if _, err := c.Lookup(member); err != nil {
						return err
					}
				}
				c.Bundles[name] = append([]string(nil), row[1:]...)
			case "dependencies.txt":
				if _, err := c.Lookup(name); err != nil {
					return err
				}
				if len(row) < 2 {
					return fmt.Errorf("empty dependencies: %s", name)
				}
				for _, member := range row[1:] {
					if _, err := c.Lookup(member); err != nil {
						return err
					}
				}
				c.Skills[c.byName[name]].Helpers = append([]string(nil), row[1:]...)
			case "markdown-output-skills.txt":
				if len(row) != 1 {
					return fmt.Errorf("invalid output entry: %v", row)
				}
				if _, err := c.Lookup(name); err != nil {
					return err
				}
				c.Skills[c.byName[name]].Output = true
			default:
				if len(row) != 1 {
					return fmt.Errorf("invalid explicit entry: %v", row)
				}
				parts := strings.Split(name, "/")
				if len(parts) != 2 {
					return fmt.Errorf("invalid explicit key: %s", name)
				}
				skill, err := c.Lookup(parts[1])
				if err != nil {
					return err
				}
				if skill.Key != name {
					return fmt.Errorf("unknown explicit key: %s", name)
				}
				c.Skills[c.byName[skill.Name]].Explicit = true
			}
		}
	}
	if c.Bundles["starter"] == nil {
		return fmt.Errorf("missing starter bundle")
	}
	adapters, err := filepath.Glob(filepath.Join(c.Root, ".agents/providers/codex/skills/*/*/agents/openai.yaml"))
	if err != nil {
		return err
	}
	for _, adapter := range adapters {
		directory := filepath.Dir(filepath.Dir(adapter))
		name := filepath.Base(directory)
		skill, err := c.Lookup(name)
		if err != nil {
			return err
		}
		if skill.Key != filepath.Base(filepath.Dir(directory))+"/"+name {
			return fmt.Errorf("unknown adapter: %s", adapter)
		}
		c.Skills[c.byName[name]].CodexAdapter = adapter
	}
	return nil
}

// Lookup returns the unique skill with name, or an error for unknown names.
func (c *Catalog) Lookup(name string) (Skill, error) {
	i, ok := c.byName[name]
	if !ok {
		return Skill{}, fmt.Errorf("unknown skill: %s", name)
	}
	return c.Skills[i], nil
}

// BundleNames returns all named selections in lexical order.
func (c *Catalog) BundleNames() []string {
	names := make([]string, 0, len(c.Bundles))
	for name := range c.Bundles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Select expands a bundle or individual names in dependency traversal order.
// Dependencies are marked before descent, so cycles terminate and duplicates vanish.
// concise adds the guidance helper after the requested selections.
func (c *Catalog) Select(selection string, concise bool) ([]string, error) {
	names, ok := c.Bundles[selection]
	if !ok {
		names = strings.Fields(selection)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("select at least one skill")
	}
	var selected []string
	seen := map[string]bool{}
	var include func(string) error
	include = func(name string) error {
		if seen[name] {
			return nil
		}
		skill, err := c.Lookup(name)
		if err != nil {
			return err
		}
		seen[name] = true
		selected = append(selected, name)
		for _, helper := range skill.Helpers {
			if err := include(helper); err != nil {
				return err
			}
		}
		return nil
	}
	for _, name := range names {
		if err := include(name); err != nil {
			return nil, err
		}
	}
	if concise {
		if err := include("concise-output"); err != nil {
			return nil, err
		}
	}
	return selected, nil
}
