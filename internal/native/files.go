// Package native builds and validates portable provider releases without
// executing packaged resources or writing to local provider installations.
package native

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var identity = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
var linkPattern = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
var remotePattern = regexp.MustCompile(`^(https?://|mailto:)`)

func digest(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func writeFile(path string, data []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, append(data, '\n'), 0644)
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(string(data))
	if len(text) == 0 || text[0] != '{' {
		return fmt.Errorf("expected JSON object: %s", path)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func safeName(name string) bool {
	return name != "" && name != "." && !strings.Contains(name, "\\") && !filepath.IsAbs(name) && filepath.ToSlash(filepath.Clean(name)) == name && name != ".." && !strings.HasPrefix(name, "../")
}

func mode(info fs.FileInfo) fs.FileMode {
	if info.Mode()&0111 != 0 {
		return 0755
	}
	return 0644
}

func inventory(root string, snapshot bool) (map[string]string, map[string]int, error) {
	files, modes := map[string]string{}, map[string]int{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink member: %s", path)
		}
		if snapshot && rel == ".git" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			if entry.Type().IsRegular() {
				return nil
			}
			return fmt.Errorf("invalid Git administration: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported member: %s", path)
		}
		if snapshot && rel == "snapshot.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = digest(data)
		modes[filepath.ToSlash(rel)] = 0100000 | int(mode(info))
		return nil
	})
	return files, modes, err
}

func validateTree(root string, snapshot bool, forbidden string) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("invalid tree root: %s", root)
	}
	files, _, err := inventory(root, snapshot)
	if err != nil {
		return err
	}
	for name := range files {
		path := filepath.Join(root, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		if (forbidden != "" && strings.Contains(text, forbidden)) || strings.Contains(text, ".agents/skills/coding/python-standards/scripts/") {
			return fmt.Errorf("clone-dependent resource: %s", path)
		}
		if filepath.Ext(path) != ".md" {
			continue
		}
		for _, match := range linkPattern.FindAllStringSubmatch(text, -1) {
			ref := strings.Trim(match[1], "<>")
			if strings.HasPrefix(ref, "#") || remotePattern.MatchString(ref) {
				continue
			}
			ref, err = url.PathUnescape(strings.SplitN(ref, "#", 2)[0])
			if err != nil {
				return err
			}
			target := filepath.Join(filepath.Dir(path), ref)
			rel, err := filepath.Rel(root, target)
			if err != nil || filepath.IsAbs(ref) || !safeName(filepath.ToSlash(rel)) {
				return fmt.Errorf("escaping link in %s: %s", path, ref)
			}
			if _, err := os.Stat(target); err != nil {
				return fmt.Errorf("broken link in %s: %w", path, err)
			}
		}
	}
	return nil
}

func stage(destination string, populate func(string) error) error {
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return fmt.Errorf("output exists or cannot be inspected: %s", destination)
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(parent, ".native-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	root := filepath.Join(temporary, "release")
	if err := populate(root); err != nil {
		return err
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return fmt.Errorf("output appeared during generation: %s", destination)
	}
	return os.Rename(root, destination)
}
