package tui

import (
	"reflect"
	"strings"
	"testing"
)

func TestDiscoveryVisibilityAndProviderIndependence(t *testing.T) {
	m, b := readyModel(t, false)
	m.catalog, m.page = selectionCatalog(), 1
	m.draft.Providers = []string{"codex", "claude"}
	m.storeProvider()
	m.status.Observed = map[string][]string{"codex": {"ticket-to-plan"}}
	before := clone(m.draft)
	m, _ = press(m, "/")
	for _, key := range []string{"p", "l", "a", "n", "enter", "f"} {
		m, _ = press(m, key)
	}
	if m.filters["codex"] != 1 || m.filters["claude"] != 0 || m.searches["claude"] != "" {
		t.Fatal("discovery leaked between agents")
	}
	for _, row := range m.providerSkillRows("codex") {
		if row.name != "" && row.name != "ticket-to-plan" {
			t.Fatal("combined search and filter ignored")
		}
	}
	if !reflect.DeepEqual(before, m.draft) || len(b.calls) != 0 {
		t.Fatal("discovery mutated configuration")
	}
	m, _ = press(m, "/")
	m, _ = press(m, "z")
	m, _ = press(m, "enter")
	if !strings.Contains(strings.Join(m.selectionLines(), "\n"), "No matching skills") {
		t.Fatal("missing empty-state help")
	}
	m, _ = press(m, "/")
	m, _ = press(m, "ctrl+u")
	m, _ = press(m, "esc")
	if m.searches["codex"] != "planz" {
		t.Fatal("cancel did not restore query")
	}
}

func TestSelectedFilterAndBulkIncludesHiddenSkills(t *testing.T) {
	m, _ := readyModel(t, false)
	m.catalog, m.page = selectionCatalog(), 1
	m.searches = map[string]string{"codex": "no matches"}
	m.filters = map[string]int{"codex": 2}
	m, _ = press(m, "a")
	if len(m.draft.Requested) != len(m.catalog.Skills) {
		t.Fatal("bulk action omitted hidden skills")
	}
	m.searches["codex"] = ""
	m, _ = press(m, "down")
	m, _ = press(m, "down")
	m, _ = press(m, "enter")
	if m.cursor >= len(m.skillRows()) {
		t.Fatal("selection change left focus outside filtered rows")
	}
}
