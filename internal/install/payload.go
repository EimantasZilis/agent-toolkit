package install

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
	"github.com/EimantasZilis/agent-toolkit/internal/render"
)

// Provenance identifies the complete working-tree input to a Development install.
// Digest includes relative paths, bytes and ordinary permission bits, without Git.
type Provenance struct {
	Kind     string `json:"kind"`
	Checkout string `json:"checkout"`
	Digest   string `json:"digest"`
}

type payloadFile struct {
	Data      []byte
	Mode      fs.FileMode
	Directory bool
}
type payload map[string]payloadFile

type ownedFile struct {
	Path      string `json:"path"`
	Hash      string `json:"sha256,omitempty"`
	Mode      uint32 `json:"mode"`
	Directory bool   `json:"directory,omitempty"`
}
type ownershipRecord struct {
	Version int         `json:"schema_version"`
	Source  Provenance  `json:"source"`
	Files   []ownedFile `json:"files"`
}

func safeRelative(name string) bool {
	return name != "." && fs.ValidPath(name) && !strings.ContainsAny(name, "\\\x00\r\n") && path.Clean(name) == name
}

func readTree(root string) (payload, error) {
	files := payload{}
	err := filepath.WalkDir(root, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if (!info.IsDir() && !info.Mode().IsRegular()) || info.Mode()&(fs.ModeSetuid|fs.ModeSetgid|fs.ModeSticky) != 0 {
			return problem("unsafe_source", file, "only real directories and regular files with ordinary permissions are supported")
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !safeRelative(rel) {
			return problem("unsafe_source", file, "unsafe relative path")
		}
		item := payloadFile{Mode: info.Mode().Perm(), Directory: info.IsDir()}
		if !item.Directory {
			item.Data, err = os.ReadFile(file)
			if err != nil {
				return err
			}
		}
		files[rel] = item
		return nil
	})
	return files, err
}

func payloadDigest(files payload) string {
	// encoding/json sorts map keys and encodes arbitrary resource bytes losslessly.
	data, _ := json.Marshal(files)
	return digest(data)
}

func writePayload(root string, files payload) error {
	// Parents are writable during materialization; apply reviewed directory modes last.
	for name, item := range files {
		if !safeRelative(name) {
			return fmt.Errorf("unsafe payload path: %q", name)
		}
		target := filepath.Join(root, filepath.FromSlash(name))
		if item.Directory {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, item.Data, 0600); err != nil {
			return err
		}
		if err := os.Chmod(target, item.Mode); err != nil {
			return err
		}
	}
	for name, item := range files {
		if item.Directory {
			if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(name)), 0755); err != nil {
				return err
			}
		}
	}
	// Deepest directories first so restrictive parent modes cannot hide children.
	names := sortedPaths(files)
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		if files[name].Directory {
			if err := os.Chmod(filepath.Join(root, filepath.FromSlash(name)), files[name].Mode); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) captureSource() (payload, Provenance, error) {
	files, err := readTree(filepath.Join(s.Root, ".agents"))
	if err != nil {
		return nil, Provenance{}, err
	}
	return files, Provenance{Kind: "development", Checkout: s.Root, Digest: payloadDigest(files)}, nil
}

func skillPayload(c *catalog.Catalog, name, provider, root string, settings Settings, selected []string) (payload, error) {
	skill, err := c.Lookup(name)
	if err != nil {
		return nil, err
	}
	files, err := readTree(skill.Source)
	if err != nil {
		return nil, err
	}
	if _, ok := files[markerName]; ok {
		return nil, fmt.Errorf("reserved ownership path in %s", name)
	}
	for rel, item := range files {
		if item.Directory || !strings.EqualFold(filepath.Ext(rel), ".md") {
			continue
		}
		text, err := render.Markdown(c, name, rel, string(item.Data), render.Options{Mode: render.Local, Provider: provider, SkillsRoot: root, OutputRoot: settings.Output, Selected: selected})
		if err != nil {
			return nil, err
		}
		item.Data = []byte(text)
		files[rel] = item
	}
	if provider == "codex" && skill.CodexAdapter != "" {
		if item, ok := files["agents"]; ok && !item.Directory {
			return nil, fmt.Errorf("adapter directory collision: %s", name)
		}
		if _, ok := files["agents/openai.yaml"]; ok {
			return nil, fmt.Errorf("adapter collision: %s", name)
		}
		info, err := os.Lstat(skill.CodexAdapter)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("unsafe adapter: %s", skill.CodexAdapter)
		}
		data, err := os.ReadFile(skill.CodexAdapter)
		if err != nil {
			return nil, err
		}
		if _, ok := files["agents"]; !ok {
			files["agents"] = payloadFile{Mode: 0755, Directory: true}
		}
		files["agents/openai.yaml"] = payloadFile{Data: data, Mode: info.Mode().Perm()}
	}
	return files, nil
}

func ownershipBytes(files payload, source Provenance) ([]byte, error) {
	record := ownershipRecord{Version: 2, Source: source}
	for _, name := range sortedPaths(files) {
		item := files[name]
		hash := ""
		if !item.Directory {
			hash = digest(item.Data)
		}
		record.Files = append(record.Files, ownedFile{Path: name, Hash: hash, Mode: uint32(item.Mode), Directory: item.Directory})
	}
	return json.MarshalIndent(record, "", "  ")
}

func validProvenance(source Provenance) bool {
	return source.Kind == "development" && filepath.IsAbs(source.Checkout) && filepath.Clean(source.Checkout) == source.Checkout && !strings.ContainsAny(source.Checkout, "\x00\r\n") && validHash(source.Digest)
}
func validHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, c := range hash {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}

func (s *Service) copiedRecord(destination string) (ownershipRecord, bool) {
	var record ownershipRecord
	marker := filepath.Join(destination, markerName)
	if regularOrAbsent(marker) != nil {
		return record, false
	}
	data, err := os.ReadFile(marker)
	if err != nil || decodeStrict(data, &record) != nil || record.Version != 2 || !validProvenance(record.Source) || record.Source.Checkout != s.Root {
		return record, false
	}
	seen := map[string]bool{}
	for _, item := range record.Files {
		if !safeRelative(item.Path) || item.Path == markerName || item.Mode > 0777 || seen[item.Path] || (item.Directory && item.Hash != "") || (!item.Directory && !validHash(item.Hash)) {
			return record, false
		}
		seen[item.Path] = true
	}
	if !seen["SKILL.md"] {
		return record, false
	}
	return record, true
}

func (s *Service) safeCopied(destination string) bool {
	record, ok := s.copiedRecord(destination)
	if !ok {
		return false
	}
	files, err := readTree(destination)
	if err != nil {
		return false
	}
	delete(files, markerName)
	if len(files) != len(record.Files) {
		return false
	}
	for _, expected := range record.Files {
		actual, ok := files[expected.Path]
		if !ok || actual.Directory != expected.Directory || uint32(actual.Mode) != expected.Mode {
			return false
		}
		if !actual.Directory && digest(actual.Data) != expected.Hash {
			return false
		}
	}
	return true
}

func sortedPaths(files payload) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
