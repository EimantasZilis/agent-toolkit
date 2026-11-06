// Package install previews and applies owned local installations without invoking
// external tooling. Inventory records observations; it never grants ownership.
package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

// Settings preserves the version 1 managed settings preferences.
type Settings struct {
	Providers, Requested, Retained []string
	Output                         string
	Concise, FullTemplate          bool
}

// Intent identifies a local management action.
type Intent string

const (
	// Configure reconciles reviewed settings with installed content.
	Configure Intent = "configure"
	// Refresh reapplies saved preferences without removing installed skills.
	Refresh Intent = "refresh"
	// Uninstall removes verified managed content and forgets saved state.
	Uninstall Intent = "uninstall"
)

// Request describes desired settings and explicitly removed helpers.
type Request struct {
	Intent  Intent
	Desired Settings
	Removed []string
}

// Operation names a proposed change or ownership conflict at Path.
type Operation struct{ Provider, Action, Name, Path string }

// Selection describes why a skill is included in a preview.
type Selection struct{ Name, Role string }

// NativeOverlap reports separately managed native packages.
type NativeOverlap struct {
	Status, Message  string
	Skills, Overlaps []string
}

// Status contains saved settings and independently observed installations.
type Status struct {
	Installed         bool
	Desired           Settings
	Applied, Observed map[string][]string
	Revision          string
	Native            NativeOverlap
}

// Preview contains read-only operations tied to a source/state revision.
type Preview struct {
	Request               Request
	Desired               Settings
	Revision              string
	Selection             []Selection
	Operations, Conflicts []Operation
	Confirmation          bool
	Native                NativeOverlap
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
	return &Service{Root: abs, Homes: homes}, nil
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
