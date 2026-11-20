package tui

import (
	"reflect"
	"strings"
	"testing"
)

func TestSkillDetailsPreserveBrowsing(t *testing.T) {
	for _, width := range []int{80, 109, 110, 150} {
		m, b := readyModel(t, false)
		m.width, m.height, m.page, m.catalog = width, 24, 1, selectionCatalog()
		m.catalog.Skills[0].Description = strings.Repeat("Long description text. ", 100) + "LASTWORD"
		for _, item := range m.catalog.Skills {
			m.expanded[item.Category] = true
		}
		for i, row := range m.skillRows() {
			if row.name == m.catalog.Skills[0].Name {
				m.cursor = i
				break
			}
		}
		if !m.skillDescriptionVisible() || m.skillPanelHeight() > 5 {
			t.Fatal("compact pane missing or too large")
		}
		m.scroll, m.manualScroll = 1, true
		before := m.View().Content
		draft := clone(m.draft)
		m, cmd := press(m, "v")
		if cmd != nil || !m.skillDetails || m.skillDescriptionVisible() {
			t.Fatal("details did not open")
		}
		for i := 0; i < 100; i++ {
			m, _ = press(m, "pgdown")
		}
		if !strings.Contains(m.View().Content, "LASTWORD") {
			t.Fatal("description end unreachable")
		}
		m, cmd = press(m, "esc")
		if cmd != nil || m.View().Content != before || !reflect.DeepEqual(draft, m.draft) || len(b.calls) != 0 {
			t.Fatal("details changed browsing or selection")
		}
		m.cursor = 0
		if m.skillDescriptionVisible() {
			t.Fatal("select-all reserves description panel")
		}
	}
}
