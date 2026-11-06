package native

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
	"github.com/EimantasZilis/agent-toolkit/internal/render"
)

var providers = []string{"codex", "claude"}

const publisher = "agent-toolkit contributors"

type skillRecord struct {
	Key            string   `json:"key"`
	Requires       []string `json:"requires"`
	ClaudeExplicit bool     `json:"claude_explicit"`
	CodexExplicit  bool     `json:"codex_explicit"`
}

type packageRecord struct {
	Provider  string            `json:"provider"`
	Selection string            `json:"selection"`
	Name      string            `json:"name"`
	Skills    []string          `json:"skills"`
	Files     map[string]string `json:"files"`
	Archive   string            `json:"archive_sha256,omitempty"`
}

type release struct {
	Version  string                 `json:"version"`
	Catalog  map[string]skillRecord `json:"catalog"`
	Bundles  map[string][]string    `json:"bundles"`
	Packages []packageRecord        `json:"packages"`
}

func packageName(selection string, bundles map[string][]string) string {
	if _, ok := bundles[selection]; ok {
		return "bundle-" + selection
	}
	return selection
}

// Build creates and validates a new archive release. Existing output is refused.
func Build(source, version, destination string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("version must be X.Y.Z without leading zeros")
	}
	c, err := catalog.Load(source)
	if err != nil {
		return err
	}
	if err := validateResources(c); err != nil {
		return err
	}
	return stage(destination, func(root string) error {
		if err := populate(c, version, root); err != nil {
			return err
		}
		if err := validateTree(root, false, c.Root); err != nil {
			return err
		}
		return Validate(root)
	})
}

func validateResources(c *catalog.Catalog) error {
	// Inspect ancestors as well as resource members, so a symlinked category or
	// adapter directory cannot cross the source boundary.
	for _, base := range []string{".agents", "LICENSE"} {
		err := filepath.WalkDir(filepath.Join(c.Root, base), func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.Type()&os.ModeSymlink != 0 || (!e.IsDir() && !e.Type().IsRegular()) {
				return fmt.Errorf("unsafe source resource: %s", path)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func populate(c *catalog.Catalog, version, root string) error {
	r := release{Version: version, Catalog: map[string]skillRecord{}, Bundles: c.Bundles}
	selections := c.BundleNames()
	for _, skill := range c.Skills {
		requires := append([]string{}, skill.Helpers...)
		explicit := false
		if skill.CodexAdapter != "" {
			data, err := os.ReadFile(skill.CodexAdapter)
			if err != nil {
				return err
			}
			explicit = strings.Contains(string(data), "allow_implicit_invocation: false")
		}
		r.Catalog[skill.Name] = skillRecord{skill.Key, requires, skill.Explicit, explicit}
		selections = append(selections, skill.Name)
	}
	sort.Strings(selections)
	for _, provider := range providers {
		for _, selection := range selections {
			selected, err := c.Select(selection, false)
			if err != nil {
				return err
			}
			sort.Strings(selected)
			name := packageName(selection, c.Bundles)
			pkg := filepath.Join(root, provider, "plugins", name)
			if err := renderPackage(c, provider, name, pkg, selected); err != nil {
				return err
			}
			if err := writeJSON(filepath.Join(pkg, "."+provider+"-plugin/plugin.json"), manifest(name, version, provider)); err != nil {
				return err
			}
			license, err := os.ReadFile(filepath.Join(c.Root, "LICENSE"))
			if err != nil {
				return err
			}
			if err := writeFile(filepath.Join(pkg, "LICENSE"), license, 0644); err != nil {
				return err
			}
			files, _, err := inventory(pkg, false)
			if err != nil {
				return err
			}
			archive := filepath.Join(root, provider, "zips", name+".zip")
			if err := writeArchive(pkg, archive); err != nil {
				return err
			}
			data, err := os.ReadFile(archive)
			if err != nil {
				return err
			}
			r.Packages = append(r.Packages, packageRecord{provider, selection, name, selected, files, digest(data)})
		}
		if err := writeMarketplace(root, provider, selections, c.Bundles, false); err != nil {
			return err
		}
	}
	return writeJSON(filepath.Join(root, "release.json"), r)
}

func renderPackage(c *catalog.Catalog, provider, name, pkg string, selected []string) error {
	for _, item := range selected {
		skill, err := c.Lookup(item)
		if err != nil {
			return err
		}
		err = filepath.WalkDir(skill.Source, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("unsafe resource: %s", path)
			}
			rel, err := filepath.Rel(skill.Source, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".md" {
				text, err := render.Markdown(c, item, rel, string(data), render.Options{Mode: render.Native, Provider: provider, Package: name, SkillsRoot: filepath.Join(pkg, "skills"), Selected: selected})
				if err != nil {
					return err
				}
				data = []byte(text)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return writeFile(filepath.Join(pkg, "skills", item, rel), data, mode(info))
		})
		if err != nil {
			return err
		}
		if provider == "codex" && skill.CodexAdapter != "" {
			data, err := os.ReadFile(skill.CodexAdapter)
			if err != nil {
				return err
			}
			if err := writeFile(filepath.Join(pkg, "skills", item, "agents/openai.yaml"), data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func manifest(name, version, provider string) map[string]any {
	description := "Portable agent-toolkit skills: " + name + "."
	m := map[string]any{"name": name, "version": version, "description": description, "author": map[string]string{"name": publisher}}
	if provider == "codex" {
		display := name
		if len(display) > 30 {
			display = display[:30]
		}
		m["skills"] = "./skills/"
		m["interface"] = map[string]any{"displayName": display, "shortDescription": "Portable engineering skills", "longDescription": description, "developerName": publisher, "category": "Developer Tools", "capabilities": []string{}}
	}
	return m
}

func marketplacePath(provider string) string {
	if provider == "claude" {
		return ".claude-plugin/marketplace.json"
	}
	return ".agents/plugins/marketplace.json"
}

func writeMarketplace(root, provider string, selections []string, bundles map[string][]string, combined bool) error {
	entries := []map[string]any{}
	for _, selection := range selections {
		name := packageName(selection, bundles)
		path := "./plugins/" + name
		if combined {
			path = "./" + provider + "/plugins/" + name
		}
		e := map[string]any{"name": name, "source": path}
		if provider == "codex" {
			e["source"] = map[string]string{"source": "local", "path": path}
			e["policy"] = map[string]string{"installation": "AVAILABLE", "authentication": "ON_INSTALL"}
			e["category"] = "Developer Tools"
		}
		entries = append(entries, e)
	}
	m := map[string]any{"name": "agent-toolkit", "plugins": entries}
	if provider == "claude" {
		m["owner"] = map[string]string{"name": publisher}
	} else {
		m["interface"] = map[string]string{"displayName": "agent-toolkit"}
	}
	if !combined {
		root = filepath.Join(root, provider)
	}
	return writeJSON(filepath.Join(root, marketplacePath(provider)), m)
}

func writeArchive(pkg, archive string) (err error) {
	if err = os.MkdirAll(filepath.Dir(archive), 0755); err != nil {
		return err
	}
	f, err := os.Create(archive)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()
	w := zip.NewWriter(f)
	err = filepath.WalkDir(pkg, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(pkg, path)
		if err != nil {
			return err
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		h := &zip.FileHeader{Name: filepath.ToSlash(rel), Method: zip.Deflate}
		h.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		h.SetMode(mode(info))
		out, err := w.CreateHeader(h)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = out.Write(data)
		return err
	})
	closeErr := w.Close()
	if err != nil {
		return err
	}
	return closeErr
}
