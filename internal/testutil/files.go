// Package testutil supports isolated filesystem and terminal verification.
// Helpers never use real provider homes unless explicitly supplied by a caller.
package testutil

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Environment overlays explicit values without leaving duplicate environment keys.
func Environment(values map[string]string) []string {
	env := []string{}
	for _, row := range os.Environ() {
		key, _, _ := strings.Cut(row, "=")
		if _, ok := values[key]; !ok {
			env = append(env, row)
		}
	}
	keys := []string{}
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = append(env, key+"="+values[key])
	}
	return env
}

// Fingerprint hashes watched trees without following symlinks. Missing paths,
// directories and permissions are included so unexpected creation is detectable.
func Fingerprint(paths ...string) (map[string]string, error) {
	result := map[string]string{}
	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
			if os.IsNotExist(err) {
				result[path] = "absent"
				return nil
			}
			if err != nil {
				return err
			}
			info, err := e.Info()
			if err != nil {
				return err
			}
			value := info.Mode().String()
			if e.Type()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				value += " link:" + target
			} else if info.Mode().IsRegular() {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				value += fmt.Sprintf(" %x", sha256.Sum256(data))
			} else if !e.IsDir() {
				return fmt.Errorf("unsupported watched file: %s", path)
			}
			result[path] = value
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
