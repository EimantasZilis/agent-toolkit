package native

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// Validate checks an archive release independently of its source checkout.
// It never extracts archives or executes packaged resources.
func Validate(root string) error {
	if err := validateTree(root, false, ""); err != nil {
		return err
	}
	_, err := validateRelease(root, false)
	return err
}

func closure(requested []string, catalog map[string]skillRecord) ([]string, error) {
	seen := map[string]bool{}
	var visit func(string) error
	visit = func(name string) error {
		info, ok := catalog[name]
		if !ok {
			return fmt.Errorf("unknown catalog skill: %s", name)
		}
		if seen[name] {
			return nil
		}
		seen[name] = true
		for _, helper := range info.Requires {
			if err := visit(helper); err != nil {
				return err
			}
		}
		return nil
	}
	for _, name := range requested {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	names := []string{}
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func validateRelease(root string, snapshot bool) (release, error) {
	var r release
	if err := readJSON(filepath.Join(root, "release.json"), &r); err != nil {
		return r, err
	}
	if !versionPattern.MatchString(r.Version) || len(r.Catalog) == 0 || len(r.Bundles) == 0 {
		return r, fmt.Errorf("invalid release metadata")
	}
	selections := []string{}
	for name, info := range r.Catalog {
		parts := strings.Split(info.Key, "/")
		if !identity.MatchString(name) || len(parts) != 2 || !identity.MatchString(parts[0]) || parts[1] != name || info.Requires == nil {
			return r, fmt.Errorf("invalid catalog identity: %s", name)
		}
		if _, err := closure([]string{name}, r.Catalog); err != nil {
			return r, err
		}
		selections = append(selections, name)
	}
	for name, members := range r.Bundles {
		if !identity.MatchString(name) || len(members) == 0 {
			return r, fmt.Errorf("invalid bundle: %s", name)
		}
		if _, ok := r.Catalog[name]; ok {
			return r, fmt.Errorf("bundle shadows skill: %s", name)
		}
		if _, err := closure(members, r.Catalog); err != nil {
			return r, err
		}
		selections = append(selections, name)
	}
	seen := map[string]bool{}
	names := map[string][]string{}
	for _, record := range r.Packages {
		key := record.Provider + "/" + record.Name
		if (record.Provider != "codex" && record.Provider != "claude") || !identity.MatchString(record.Name) || seen[key] || record.Name != packageName(record.Selection, r.Bundles) {
			return r, fmt.Errorf("invalid or duplicate package: %s", key)
		}
		seen[key] = true
		if err := validatePackage(root, record, r, snapshot); err != nil {
			return r, fmt.Errorf("%s: %w", key, err)
		}
		names[record.Provider] = append(names[record.Provider], record.Name)
	}
	if len(seen) != len(selections)*2 {
		return r, fmt.Errorf("release package membership mismatch")
	}
	for _, provider := range providers {
		for _, selection := range selections {
			if !seen[provider+"/"+packageName(selection, r.Bundles)] {
				return r, fmt.Errorf("missing package: %s/%s", provider, selection)
			}
		}
		sort.Strings(names[provider])
		entries, err := os.ReadDir(filepath.Join(root, provider, "plugins"))
		if err != nil {
			return r, err
		}
		actual := []string{}
		for _, entry := range entries {
			actual = append(actual, entry.Name())
		}
		if !reflect.DeepEqual(actual, names[provider]) {
			return r, fmt.Errorf("unexpected package directories: %s", provider)
		}
		if err := validateMarketplace(root, provider, names[provider], snapshot); err != nil {
			return r, err
		}
	}
	return r, nil
}

func validatePackage(root string, record packageRecord, r release, snapshot bool) error {
	pkg := filepath.Join(root, record.Provider, "plugins", record.Name)
	if err := validateTree(pkg, false, ""); err != nil {
		return err
	}
	files, _, err := inventory(pkg, false)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(files, record.Files) {
		return fmt.Errorf("package inventory mismatch")
	}
	if _, ok := files["LICENSE"]; !ok {
		return fmt.Errorf("missing license")
	}
	requested, ok := r.Bundles[record.Selection]
	if !ok {
		requested = []string{record.Selection}
	}
	expected, err := closure(requested, r.Catalog)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, record.Skills) {
		return fmt.Errorf("incomplete helper closure")
	}
	entries, err := os.ReadDir(filepath.Join(pkg, "skills"))
	if err != nil {
		return err
	}
	actual := []string{}
	for _, entry := range entries {
		actual = append(actual, entry.Name())
	}
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("skill membership mismatch")
	}
	for _, name := range expected {
		path := filepath.Join(pkg, "skills", name)
		data, err := os.ReadFile(filepath.Join(path, "SKILL.md"))
		if err != nil {
			return err
		}
		text := string(data)
		parts := strings.SplitN(text, "---", 3)
		if !strings.HasPrefix(text, "---\n") || len(parts) != 3 || !strings.Contains(parts[1], "\nname: "+name+"\n") || !strings.Contains(parts[1], "\ndescription: ") {
			return fmt.Errorf("invalid skill front matter: %s", name)
		}
		info := r.Catalog[name]
		if record.Provider == "claude" && info.ClaudeExplicit && !strings.Contains(parts[1], "\ndisable-model-invocation: true\n") {
			return fmt.Errorf("missing Claude explicit policy: %s", name)
		}
		if record.Provider == "codex" && info.CodexExplicit {
			data, err := os.ReadFile(filepath.Join(path, "agents/openai.yaml"))
			if err != nil {
				return err
			}
			if !strings.Contains(string(data), "allow_implicit_invocation: false") {
				return fmt.Errorf("missing Codex explicit policy: %s", name)
			}
		}
	}
	if err := validateManifest(pkg, record, r.Version); err != nil {
		return err
	}
	if !snapshot {
		return validateArchive(filepath.Join(root, record.Provider, "zips", record.Name+".zip"), pkg, record)
	}
	return nil
}

func validateManifest(pkg string, record packageRecord, version string) error {
	var m struct {
		Name, Version, Description string
		Author                     struct{ Name string }
		Skills                     string
		Interface                  map[string]json.RawMessage
	}
	if err := readJSON(filepath.Join(pkg, "."+record.Provider+"-plugin/plugin.json"), &m); err != nil {
		return err
	}
	if m.Name != record.Name || m.Version != version || m.Description == "" || m.Author.Name == "" {
		return fmt.Errorf("invalid plugin manifest")
	}
	if record.Provider == "codex" {
		if m.Skills != "./skills/" {
			return fmt.Errorf("invalid skills declaration")
		}
		for _, key := range []string{"displayName", "shortDescription", "longDescription", "developerName", "category"} {
			var s string
			if json.Unmarshal(m.Interface[key], &s) != nil || s == "" {
				return fmt.Errorf("missing interface.%s", key)
			}
		}
		var caps []string
		if json.Unmarshal(m.Interface["capabilities"], &caps) != nil || caps == nil {
			return fmt.Errorf("invalid capabilities")
		}
	}
	return nil
}

func validateArchive(path, pkg string, record packageRecord) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if digest(data) != record.Archive {
		return fmt.Errorf("archive checksum mismatch")
	}
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	if len(r.File) != len(record.Files) {
		return fmt.Errorf("duplicate or missing ZIP member")
	}
	seen := map[string]bool{}
	for _, file := range r.File {
		if !safeName(file.Name) || seen[file.Name] {
			return fmt.Errorf("unsafe ZIP member: %s", file.Name)
		}
		seen[file.Name] = true
		if _, ok := record.Files[file.Name]; !ok {
			return fmt.Errorf("unexpected ZIP member: %s", file.Name)
		}
		if !file.Mode().IsRegular() || (file.Mode().Perm() != 0644 && file.Mode().Perm() != 0755) {
			return fmt.Errorf("unsupported ZIP member mode: %s", file.Name)
		}
		info, err := os.Stat(filepath.Join(pkg, file.Name))
		if err != nil {
			return err
		}
		if file.Mode().Perm() != mode(info) {
			return fmt.Errorf("ZIP mode mismatch: %s", file.Name)
		}
		reader, err := file.Open()
		if err != nil {
			return err
		}
		// Limit decompression to the corresponding regular file's length.
		content, readErr := io.ReadAll(io.LimitReader(reader, info.Size()+1))
		closeErr := reader.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if digest(content) != record.Files[file.Name] {
			return fmt.Errorf("ZIP content mismatch: %s", file.Name)
		}
	}
	return nil
}

func validateMarketplace(root, provider string, names []string, combined bool) error {
	base := root
	if !combined {
		base = filepath.Join(root, provider)
	}
	var m struct {
		Name    string
		Owner   struct{ Name string }
		Plugins []struct {
			Name   string
			Source json.RawMessage
			Policy map[string]string
		}
	}
	if err := readJSON(filepath.Join(base, marketplacePath(provider)), &m); err != nil {
		return err
	}
	if m.Name != "agent-toolkit" || (provider == "claude" && m.Owner.Name == "") {
		return fmt.Errorf("invalid marketplace identity")
	}
	actual := []string{}
	for _, e := range m.Plugins {
		actual = append(actual, e.Name)
		expected := "./plugins/" + e.Name
		if combined {
			expected = "./" + provider + "/plugins/" + e.Name
		}
		var path string
		if provider == "codex" {
			var source struct{ Source, Path string }
			if err := json.Unmarshal(e.Source, &source); err != nil {
				return err
			}
			if source.Source != "local" || !reflect.DeepEqual(e.Policy, map[string]string{"installation": "AVAILABLE", "authentication": "ON_INSTALL"}) {
				return fmt.Errorf("invalid marketplace policy")
			}
			path = source.Path
		} else if err := json.Unmarshal(e.Source, &path); err != nil {
			return err
		}
		if path != expected {
			return fmt.Errorf("invalid marketplace source: %s", path)
		}
	}
	sort.Strings(actual)
	if !reflect.DeepEqual(actual, names) {
		return fmt.Errorf("marketplace membership mismatch: %s", provider)
	}
	return nil
}
