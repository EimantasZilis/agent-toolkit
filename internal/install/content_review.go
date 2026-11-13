package install

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

func (s *Service) contentChanges(p Preview, legacy *catalog.Catalog) ([]ContentChange, error) {
	var changes []ContentChange
	for _, op := range p.Operations {
		if op.Action == "conflict" || op.Action == "keep" {
			continue
		}
		if strings.Contains(op.Action, "guidance") {
			change, err := s.guidanceChange(p, op)
			if err != nil {
				return nil, err
			}
			changes = append(changes, change)
			continue
		}
		var before payload
		links := map[string]bool{}
		var err error
		if op.Action == "convert" {
			// Revalidate before reading any supported link. Never use readTree on a
			// legacy tree: only this reader may inspect validated resource targets.
			if err := s.validateLegacy(legacy, op.Provider, op.Name); err != nil {
				return nil, err
			}
			before, links, err = readLegacyReview(op.Path)
		} else if op.Action != "install" {
			if !s.safeCopied(op.Path) {
				return nil, problem("ownership_conflict", op.Path, "copied content changed during review")
			}
			before, err = readTree(op.Path)
		}
		if err != nil {
			return nil, err
		}
		delete(before, markerName)
		after := p.payloads[op.Provider+"/"+op.Name]
		if op.Action == "remove" {
			after = nil
		}
		paths := payload{}
		for name := range before {
			paths[name] = payloadFile{}
		}
		for name := range after {
			paths[name] = payloadFile{}
		}
		for _, name := range sortedPaths(paths) {
			old, oldOK := before[name]
			next, nextOK := after[name]
			a, b := contentState(old, oldOK), contentState(next, nextOK)
			if links[name] {
				a.Type = "symlink"
			}
			if sameContent(a, b) {
				continue
			}
			changes = append(changes, describeChange(op, filepath.Join(op.Path, filepath.FromSlash(name)), a, b, old.Data, next.Data))
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Provider != changes[j].Provider {
			return changes[i].Provider < changes[j].Provider
		}
		return changes[i].Path < changes[j].Path
	})
	return changes, nil
}

func readLegacyReview(root string) (payload, map[string]bool, error) {
	files, links := payload{}, map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == markerName {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		item := payloadFile{Mode: info.Mode().Perm(), Directory: info.IsDir()}
		if !item.Directory {
			// validateLegacy has checked exact link text, canonical regular targets,
			// and all directory ancestors. The final revision check seals these reads.
			item.Data, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		rel = filepath.ToSlash(rel)
		files[rel] = item
		links[rel] = info.Mode()&os.ModeSymlink != 0
		return nil
	})
	return files, links, err
}

func contentState(item payloadFile, present bool) *ContentState {
	if !present {
		return nil
	}
	state := &ContentState{Type: "file", Mode: uint32(item.Mode.Perm())}
	if item.Directory {
		state.Type = "directory"
	} else {
		state.Size, state.SHA256 = len(item.Data), digest(item.Data)
	}
	return state
}

func sameContent(a, b *ContentState) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func describeChange(op Operation, path string, before, after *ContentState, old, next []byte) ContentChange {
	change := ContentChange{Provider: op.Provider, Name: op.Name, Path: path, Kind: "modified", Before: before, After: after}
	switch {
	case before == nil:
		change.Kind = "added"
	case after == nil:
		change.Kind = "removed"
	case before.Type != after.Type:
		change.Kind = "replaced"
		if before.Type == "symlink" {
			change.Kind = "converted"
		}
	case before.SHA256 == after.SHA256:
		change.Kind = "mode"
	}
	if bytes.Equal(old, next) {
		return change
	}
	if !reviewText(old) || !reviewText(next) {
		change.Binary = true
		return change
	}
	change.Lines, change.Truncated = textChanges(string(old), string(next))
	return change
}

func reviewText(data []byte) bool { return utf8.Valid(data) && !bytes.ContainsRune(data, 0) }

// textChanges omits common leading/trailing lines and bounds the remaining
// excerpt by both bytes and lines. Each changed path is retained separately.
func textChanges(old, next string) ([]string, bool) {
	split := func(value string) []string {
		if value == "" {
			return nil
		}
		return strings.SplitAfter(value, "\n")
	}
	a, b := split(old), split(next)
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		a, b = a[1:], b[1:]
	}
	for len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		a, b = a[:len(a)-1], b[:len(b)-1]
	}
	var lines []string
	remaining := 8192
	for i, group := range [][]string{a, b} {
		prefix := "- "
		if i == 1 {
			prefix = "+ "
		}
		for _, line := range group {
			if len(lines) >= 100 || remaining <= len(prefix) {
				return lines, true
			}
			line = strings.TrimSuffix(line, "\n")
			if len(line)+len(prefix) > remaining {
				line = line[:remaining-len(prefix)]
				for !utf8.ValidString(line) {
					line = line[:len(line)-1]
				}
				return append(lines, prefix+line), true
			}
			lines = append(lines, prefix+line)
			remaining -= len(line) + len(prefix)
		}
	}
	return lines, false
}
