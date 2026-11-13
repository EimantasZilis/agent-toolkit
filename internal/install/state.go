package install

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var validName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type inventory struct {
	Source    *Provenance         `json:"source,omitempty"`
	Version   int                 `json:"schema_version"`
	Revision  string              `json:"revision"`
	Providers map[string][]string `json:"providers"`
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func exists(path string) bool   { _, err := os.Lstat(path); return !os.IsNotExist(err) }
func regularOrAbsent(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return problem("invalid_state", path, "expected regular file")
	}
	return nil
}
func unique(names []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, name := range names {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}
func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

func validSettings(settings Settings) error {
	for _, provider := range settings.Providers {
		if provider != "codex" && provider != "claude" {
			return problem("invalid_settings", "", "unknown provider: "+provider)
		}
	}
	for _, name := range append(append([]string(nil), settings.Requested...), settings.Retained...) {
		if !validName.MatchString(name) {
			return problem("invalid_settings", "", "invalid skill: "+name)
		}
	}
	if (settings.Output != "" && !filepath.IsAbs(settings.Output)) || strings.ContainsAny(settings.Output, "\n\r\x00") {
		return problem("invalid_settings", settings.Output, "output path must be absolute and single-line")
	}
	return nil
}

func (s *Service) state() (Settings, map[string][]string, bool, error) {
	settings := Settings{}
	applied := map[string][]string{"codex": {}, "claude": {}}
	dir := filepath.Join(s.Root, "config")
	if info, err := os.Lstat(dir); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return settings, nil, false, problem("invalid_state", dir, "state directory must be a real directory")
	} else if err != nil && !os.IsNotExist(err) {
		return settings, nil, false, err
	}
	settingsPath, inventoryPath := filepath.Join(dir, "settings.env"), filepath.Join(dir, "inventory.json")
	for _, path := range []string{settingsPath, inventoryPath} {
		if err := regularOrAbsent(path); err != nil {
			return settings, nil, false, err
		}
	}
	if !exists(settingsPath) && !exists(inventoryPath) {
		return settings, applied, false, nil
	}
	if !exists(settingsPath) || !exists(inventoryPath) {
		return settings, nil, false, problem("invalid_state", dir, "restore settings and inventory together")
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return settings, nil, true, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return settings, nil, true, problem("invalid_state", settingsPath, "malformed settings row")
		}
		if _, found := values[key]; found {
			return settings, nil, true, problem("invalid_state", settingsPath, "duplicate key: "+key)
		}
		values[key] = value
	}
	for _, key := range []string{"STATE_VERSION", "PROVIDERS", "REQUESTED_SKILLS", "RETAINED_HELPERS", "MARKDOWN_OUTPUT_PATH", "CONCISE_OUTPUT", "GLOBAL_GUIDANCE"} {
		if _, ok := values[key]; !ok {
			return settings, nil, true, problem("invalid_state", settingsPath, "missing key: "+key)
		}
	}
	if values["STATE_VERSION"] != "1" {
		return settings, nil, true, problem("invalid_state", settingsPath, "unsupported settings version")
	}
	settings.Providers, settings.Requested, settings.Retained = strings.Fields(values["PROVIDERS"]), strings.Fields(values["REQUESTED_SKILLS"]), strings.Fields(values["RETAINED_HELPERS"])
	settings.Output = values["MARKDOWN_OUTPUT_PATH"]
	settings.Concise, err = parseBoolean(values["CONCISE_OUTPUT"])
	if err != nil {
		return settings, nil, true, err
	}
	settings.FullTemplate, err = parseBoolean(values["GLOBAL_GUIDANCE"])
	if err != nil {
		return settings, nil, true, err
	}
	if err := validSettings(settings); err != nil {
		return settings, nil, true, err
	}
	data, err = os.ReadFile(inventoryPath)
	if err != nil {
		return settings, nil, true, err
	}
	var inv inventory
	if err := decodeStrict(data, &inv); err != nil {
		return settings, nil, true, fmt.Errorf("decode inventory: %w", err)
	}
	if (inv.Version != 1 && inv.Version != 2) || (inv.Version == 1 && inv.Source != nil) || (inv.Version == 2 && (inv.Source == nil || !validProvenance(*inv.Source))) || len(inv.Providers) != 2 {
		return settings, nil, true, problem("invalid_state", inventoryPath, "invalid inventory schema")
	}
	for _, provider := range []string{"codex", "claude"} {
		names, ok := inv.Providers[provider]
		if !ok {
			return settings, nil, true, problem("invalid_state", inventoryPath, "missing provider")
		}
		for _, name := range names {
			if !validName.MatchString(name) {
				return settings, nil, true, problem("invalid_state", inventoryPath, "invalid skill identity")
			}
		}
	}
	return settings, inv.Providers, true, nil
}

func parseBoolean(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, problem("invalid_state", "", "invalid boolean: "+value)
	}
}

func settingsBytes(settings Settings) []byte {
	return []byte(fmt.Sprintf("STATE_VERSION=1\nPROVIDERS=%s\nREQUESTED_SKILLS=%s\nRETAINED_HELPERS=%s\nMARKDOWN_OUTPUT_PATH=%s\nCONCISE_OUTPUT=%t\nGLOBAL_GUIDANCE=%t\n", strings.Join(unique(settings.Providers), " "), strings.Join(unique(settings.Requested), " "), strings.Join(unique(settings.Retained), " "), settings.Output, settings.Concise, settings.FullTemplate))
}

func inventoryBytes(settings []byte, applied map[string][]string, source ...Provenance) ([]byte, error) {
	// Preserve the version 1 state format, including empty provider rows.
	for _, provider := range []string{"codex", "claude"} {
		if applied[provider] == nil {
			applied[provider] = []string{}
		}
	}
	inv := inventory{Version: 1, Revision: digest(settings), Providers: applied}
	if len(source) > 0 {
		inv.Version = 2
		inv.Source = &source[0]
	}
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return nil, err
	}
	// Existing state files use single-line arrays separated by comma-space.
	for _, provider := range []string{"codex", "claude"} {
		array, err := json.Marshal(applied[provider])
		if err != nil {
			return nil, err
		}
		pretty, err := json.MarshalIndent(applied[provider], "    ", "  ")
		if err != nil {
			return nil, err
		}
		data = bytes.Replace(data, pretty, bytes.ReplaceAll(array, []byte(","), []byte(", ")), 1)
	}
	return append(data, '\n'), nil
}
