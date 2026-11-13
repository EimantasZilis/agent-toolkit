package install

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

// validateLegacy recognizes only the tree materialized by the linked Go
// installer. Recognition grants no authority to ordinary management actions.
func (s *Service) validateLegacy(c *catalog.Catalog, provider, name string) error {
	root, _ := s.destinations(provider)
	destination := filepath.Join(root, name)
	reject := func(path string) error {
		return problem("legacy_conflict", path, "expected an intact linked installation from this checkout")
	}
	settings, applied, installed, err := s.state()
	if err != nil {
		return err
	}
	if !installed || !contains(settings.Providers, provider) || !contains(applied[provider], name) {
		return problem("migration_state", destination, "migration requires saved settings and inventory for this skill")
	}
	base := s.Homes.Agents
	if provider == "claude" {
		base = s.Homes.Claude
	}
	if !realDirectoryPath(destination, base) {
		return reject(destination)
	}
	skill, err := c.Lookup(name)
	if err != nil {
		return problem("migration_selection", destination, err.Error())
	}
	source, err := readTree(skill.Source)
	if err != nil {
		return err
	}
	// Historical installation created directories only as parents of resources.
	expected := map[string]string{"SKILL.md": "", markerName: ""}
	addLink := func(rel, target string) {
		expected[rel] = target
		for dir := filepath.Dir(rel); dir != "."; dir = filepath.Dir(dir) {
			expected[dir] = ""
		}
	}
	for rel, item := range source {
		if !item.Directory && rel != "SKILL.md" {
			addLink(filepath.FromSlash(rel), filepath.Join(skill.Source, filepath.FromSlash(rel)))
		}
	}
	if provider == "codex" && skill.CodexAdapter != "" {
		addLink(filepath.Join("agents", "openai.yaml"), skill.CodexAdapter)
	}
	for _, rel := range []string{"SKILL.md", markerName} {
		path := filepath.Join(destination, rel)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return reject(path)
		}
	}
	manifest, err := os.ReadFile(filepath.Join(destination, "SKILL.md"))
	if err != nil {
		return err
	}
	marker, err := os.ReadFile(filepath.Join(destination, markerName))
	if err != nil {
		return err
	}
	if string(marker) != s.Root+"\nSKILL.md "+digest(manifest)+"\n" {
		return reject(filepath.Join(destination, markerName))
	}
	err = filepath.WalkDir(destination, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return reject(path)
		}
		if path == destination {
			return nil
		}
		rel, err := filepath.Rel(destination, path)
		if err != nil {
			return err
		}
		target, ok := expected[rel]
		if !ok {
			return reject(path)
		}
		delete(expected, rel)
		if rel == "SKILL.md" || rel == markerName {
			return nil
		}
		if target == "" {
			if !entry.IsDir() {
				return reject(path)
			}
			return nil
		}
		// Check link text before resolving anything, so foreign targets are never
		// traversed to establish ownership. Source trees contain no symlinks.
		link, err := os.Readlink(path)
		if err != nil || link != target {
			return reject(path)
		}
		canonical, err := filepath.EvalSymlinks(target)
		if err != nil || canonical != target {
			return reject(path)
		}
		info, err := os.Lstat(target)
		if err != nil || !info.Mode().IsRegular() {
			return reject(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for rel := range expected {
		return reject(filepath.Join(destination, rel))
	}
	return nil
}
