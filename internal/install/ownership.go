package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

const markerName = ".agent-toolkit-owned"

func realDirectoryPath(path, boundary string) bool {
	if !filepath.IsAbs(boundary) || strings.ContainsAny(boundary, "\n\r\x00") {
		return false
	}
	for current := filepath.Clean(path); current != filepath.Clean(boundary); current = filepath.Dir(current) {
		if current == filepath.Dir(current) {
			return false
		}
		if info, err := os.Lstat(current); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return false
		} else if err != nil && !os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (s *Service) owned(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	marker := filepath.Join(path, markerName)
	if err := regularOrAbsent(marker); err != nil {
		return false
	}
	data, err := os.ReadFile(marker)
	return err == nil && strings.HasPrefix(string(data), s.Root+"\n")
}

func sameLink(path, expected string) bool {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	actual, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	want, err := filepath.EvalSymlinks(expected)
	if err != nil || actual != want {
		return false
	}
	info, err = os.Stat(expected)
	return err == nil && info.Mode().IsRegular()
}

func (s *Service) safeSkill(skill catalog.Skill, provider, destination string) bool {
	if !s.owned(destination) {
		return false
	}
	manifest := filepath.Join(destination, "SKILL.md")
	if err := regularOrAbsent(manifest); err != nil {
		return false
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		return false
	}
	marker, err := os.ReadFile(filepath.Join(destination, markerName))
	if err != nil || string(marker) != s.Root+"\nSKILL.md "+digest(data)+"\n" {
		return false
	}
	err = filepath.WalkDir(destination, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(destination, path)
		if err != nil {
			return err
		}
		if rel == "." || rel == "SKILL.md" || rel == markerName {
			return nil
		}
		expected := filepath.Join(skill.Source, rel)
		if provider == "codex" && rel == "agents/openai.yaml" {
			expected = skill.CodexAdapter
		}
		if entry.IsDir() {
			info, err := os.Stat(expected)
			if err == nil && info.IsDir() {
				return nil
			}
			if rel == "agents" && provider == "codex" && skill.CodexAdapter != "" {
				return nil
			}
			return fmt.Errorf("unexpected directory: %s", path)
		}
		if expected == "" || !sameLink(path, expected) {
			return fmt.Errorf("unexpected resource: %s", path)
		}
		return nil
	})
	return err == nil
}

func resourcesComplete(skill catalog.Skill, provider, destination string) bool {
	err := filepath.WalkDir(skill.Source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path == filepath.Join(skill.Source, "SKILL.md") {
			return nil
		}
		rel, err := filepath.Rel(skill.Source, path)
		if err != nil {
			return err
		}
		if !sameLink(filepath.Join(destination, rel), path) {
			return fmt.Errorf("missing resource")
		}
		return nil
	})
	if err != nil {
		return false
	}
	return provider != "codex" || skill.CodexAdapter == "" || sameLink(filepath.Join(destination, "agents/openai.yaml"), skill.CodexAdapter)
}

func (s *Service) observed(provider string) ([]string, error) {
	root, _ := s.destinations(provider)
	if info, err := os.Lstat(root); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if validName.MatchString(entry.Name()) && s.owned(filepath.Join(root, entry.Name())) {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func fingerprint(path string, out *strings.Builder) error {
	fmt.Fprintf(out, "PATH %q\n", path)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		out.WriteString("ABSENT\n")
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "MODE %s\n", info.Mode())
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "LINK %q\n", target)
		return nil
	}
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := fingerprint(filepath.Join(path, entry.Name()), out); err != nil {
				return err
			}
		}
		return nil
	}
	if info.Mode().IsRegular() {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out.WriteString(digest(data) + "\n")
	}
	return nil
}

func (s *Service) revision() (string, error) {
	var result strings.Builder
	paths := []string{"config/settings.env", "config/inventory.json", ".agents", "internal", "cmd", "go.mod", "go.sum"}
	for _, path := range paths {
		if err := fingerprint(filepath.Join(s.Root, path), &result); err != nil {
			return "", err
		}
	}
	if s.NativeInventory != "" {
		if err := fingerprint(s.NativeInventory, &result); err != nil {
			return "", err
		}
	}
	for _, provider := range []string{"codex", "claude"} {
		skills, guidance := s.destinations(provider)
		for _, path := range []string{skills, guidance, filepath.Join(filepath.Dir(guidance), markerName)} {
			if err := fingerprint(path, &result); err != nil {
				return "", err
			}
		}
	}
	return digest([]byte(result.String())), nil
}
