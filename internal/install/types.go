// Package install previews and applies owned local installations without invoking
// external tooling. Inventory records observations; it never grants ownership.
package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

// Settings holds installation preferences. Shared selection fields support legacy callers.
type Settings struct {
	Documentation                     *bool `json:",omitempty"`
	ByProvider                        map[string]ProviderSettings
	Providers, Requested, Retained    []string
	Output                            string
	SharedStorage, SeparateByProvider bool
	OutputProvider                    string
	Concise, FullTemplate             bool
}

// ProviderSettings holds independent skills and guidance. Output is legacy storage.
type ProviderSettings struct {
	Documentation         *bool  `json:",omitempty"`
	Output                string `json:",omitempty"`
	Requested, Retained   []string
	Concise, FullTemplate bool
}

// ForProvider returns independent installation preferences for one provider.
func (s Settings) ForProvider(provider string) Settings {
	result := s
	result.ByProvider = nil
	result.OutputProvider = provider
	if p, ok := s.ByProvider[provider]; ok {
		result.Requested, result.Retained = p.Requested, p.Retained
		result.Concise, result.FullTemplate = p.Concise, p.FullTemplate
		result.Documentation = p.Documentation
		if !s.SharedStorage {
			result.Output = p.Output
		}
	}
	result.Requested = append([]string(nil), result.Requested...)
	result.Retained = append([]string(nil), result.Retained...)
	return result
}

// ProviderPreferences returns the selection stored for one provider.
func (s Settings) ProviderPreferences(provider string) ProviderSettings {
	p := s.ForProvider(provider)
	if s.SharedStorage {
		p.Output = ""
	}
	return ProviderSettings{Documentation: p.Documentation, Requested: p.Requested, Retained: p.Retained, Concise: p.Concise, FullTemplate: p.FullTemplate, Output: p.Output}
}

// DocumentationEnabled preserves combined guidance preferences when the separate
// documentation setting has not yet been saved.
func (s Settings) DocumentationEnabled() bool {
	if s.Documentation == nil {
		return s.FullTemplate
	}
	return *s.Documentation
}

// OutputPaths returns the distinct external output locations of enabled providers.
func (s Settings) OutputPaths() []string {
	var paths []string
	for _, id := range s.Providers {
		if path := s.ForProvider(id).Output; path != "" {
			paths = append(paths, path)
		}
	}
	return unique(paths)
}

// Intent identifies a local management action.
type Intent string

const (
	// Configure reconciles reviewed settings with installed content.
	Configure Intent = "configure"
	// Refresh reapplies saved preferences without removing installed skills.
	Refresh Intent = "refresh"
	// Migrate converts verified linked skills using unchanged saved preferences.
	Migrate Intent = "migrate"
	// Uninstall removes verified managed content and forgets saved state.
	Uninstall Intent = "uninstall"
)

// Request describes desired settings and explicitly removed helpers.
type Request struct {
	Intent            Intent
	Desired           Settings
	Removed           []string
	RemovedByProvider map[string][]string
}

// Operation names a proposed change or ownership conflict at Path.
type Operation struct {
	Provider, Action, Name, Path string
	// Reason and Recovery explain why a conflict blocks the operation.
	Reason, Recovery string
}

// ContentState describes a reviewed file, directory, or legacy resource link.
// Size and SHA256 describe file bytes (the target bytes for a verified link).
type ContentState struct {
	Type   string
	Mode   uint32
	Size   int
	SHA256 string
}

// ContentChange describes installed content independently of executable actions.
// A nil state means absent. Lines contain bounded, prefixed text changes;
// Binary and Truncated explain why full text is unavailable.
type ContentChange struct {
	Provider, Name, Path, Kind string
	Before, After              *ContentState
	Lines                      []string
	Binary, Truncated          bool
}

// Selection describes why a skill is included in a preview.
type Selection struct{ Name, Role string }

// NativeOverlap reports separately managed native packages.
type NativeOverlap struct {
	Status, Message  string
	Skills, Overlaps []string
}

// GuidanceStatus describes the observed toolkit instructions in a provider file.
type GuidanceStatus struct {
	Path, State                                   string
	FullTemplate, Concise, Output                 bool
	CodingPath, ConcisePath, OutputPath, Location string
	Documentation                                 bool
	DocumentationPath                             string
}

// Status contains saved settings and independently observed installations.
// Linked reports marker candidates; migration still verifies their full trees.
type Status struct {
	Installed, Linked bool
	Desired           Settings
	Applied, Observed map[string][]string
	Guidance          map[string]GuidanceStatus
	Revision          string
	Native            NativeOverlap
}

// Preview contains read-only operations tied to a source/state revision.
type Preview struct {
	Request               Request
	Desired               Settings
	Revision              string
	Selection             []Selection
	SelectionByProvider   map[string][]Selection
	Operations, Conflicts []Operation
	Changes               []ContentChange
	Confirmation          bool
	Native                NativeOverlap
	Source                Provenance
	payloads              map[string]payload
	guidance              map[string]string
	seal                  string
}

// Confirmations records explicit acceptance of destructive and directory actions.
type Confirmations struct{ Removals, CreateOutput bool }

// Error supplies a stable code and affected path for callers.
type Error struct{ Code, Path, Message string }

func (e *Error) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Path)
	}
	return e.Code + ": " + e.Message
}

// Homes makes provider destinations explicit and testable.
type Homes struct{ Agents, Codex, Claude string }

// EnvironmentHomes resolves the established provider-home overrides.
func EnvironmentHomes() (Homes, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Homes{}, err
	}
	value := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return filepath.Join(home, fallback)
	}
	return Homes{value("AGENTS_HOME", ".agents"), value("CODEX_HOME", ".codex"), value("CLAUDE_HOME", ".claude")}, nil
}

// Service owns the source root and destination configuration. It is read-only
// outside Apply; separate services coordinate mutations through an on-disk lock.
type Service struct {
	Root            string
	Homes           Homes
	NativeInventory string
	fail            func(string) error // private deterministic transaction failure hook
}

// New resolves source paths without creating state or provider directories.
func New(root string, homes Homes) (*Service, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	return &Service{Root: canonical, Homes: homes}, nil
}

// Catalog reads the current validated catalog.
func (s *Service) Catalog() (*catalog.Catalog, error) { return catalog.Load(s.Root) }

func (s *Service) destinations(provider string) (skills, guidance string) {
	if provider == "codex" {
		return filepath.Join(s.Homes.Agents, "skills/agent-toolkit"), filepath.Join(s.Homes.Codex, "AGENTS.md")
	}
	return filepath.Join(s.Homes.Claude, "skills"), filepath.Join(s.Homes.Claude, "CLAUDE.md")
}

func problem(code, path, message string) error { return &Error{code, path, message} }
