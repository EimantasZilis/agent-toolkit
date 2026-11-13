package install

import (
	"os"
	"path/filepath"
	"slices"
)

func sameNames(a, b []string) bool { return slices.Equal(unique(a), unique(b)) }

// migrationSelection requires an unchanged dependency closure for each saved
// provider. State records constrain selection; validateLegacy supplies ownership.
func (s *Service) migrationSelection(status Status, selected []string) error {
	path := filepath.Join(s.Root, "config/inventory.json")
	settings, err := os.ReadFile(filepath.Join(s.Root, "config/settings.env"))
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var inv inventory
	if err := decodeStrict(data, &inv); err != nil {
		return err
	}
	if inv.Revision != digest(settings) {
		return problem("migration_state", path, "saved settings and inventory do not match")
	}
	for _, provider := range []string{"codex", "claude"} {
		var expected []string
		if contains(status.Desired.Providers, provider) {
			expected = selected
		}
		if !sameNames(expected, status.Applied[provider]) {
			return problem("migration_selection", path, "saved dependency selection changed for "+provider+"; restore the matching checkout and state")
		}
		// Missing skills and extra checkout-marked skills cannot be reconciled
		// while promising to preserve the saved selection.
		if !sameNames(expected, status.Observed[provider]) {
			root, _ := s.destinations(provider)
			for _, name := range unique(append(append([]string(nil), expected...), status.Observed[provider]...)) {
				if contains(expected, name) != contains(status.Observed[provider], name) {
					return problem("migration_state", filepath.Join(root, name), "installed skills do not match the saved selection")
				}
			}
		}
	}
	return nil
}
