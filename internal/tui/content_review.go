package tui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

// terminalText escapes controls visibly so source bytes cannot issue terminal
// commands, hide review rows, or reorder names through Unicode format controls.
func terminalText(value string) string {
	var out strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029' {
			quoted := strconv.QuoteRune(r)
			out.WriteString(quoted[1 : len(quoted)-1])
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func contentReviewLines(changes []install.ContentChange) []string {
	if len(changes) == 0 {
		return []string{"Installed content: unchanged."}
	}
	lines := []string{"Content changes (guidance shows managed blocks only):"}
	provider, name := "", ""
	for _, change := range changes {
		if change.Provider != provider || change.Name != name {
			provider, name = change.Provider, change.Name
			lines = append(lines, terminalText(provider+" / "+name))
		}
		lines = append(lines, "  "+terminalText(change.Kind+" "+change.Path), "    "+reviewMode(change.Before)+" -> "+reviewMode(change.After))
		if change.Binary {
			lines = append(lines, "    Binary content:", "    before: "+reviewBytes(change.Before), "    after:  "+reviewBytes(change.After))
		}
		for _, line := range change.Lines {
			lines = append(lines, "    "+terminalText(line))
		}
		if change.Truncated {
			lines = append(lines, "    [Text excerpt truncated: limited to 100 lines / 8192 bytes; path and modes retained.]")
		}
	}
	return lines
}

func reviewMode(state *install.ContentState) string {
	if state == nil {
		return "absent"
	}
	return fmt.Sprintf("%s mode %04o", state.Type, state.Mode)
}

func reviewBytes(state *install.ContentState) string {
	if state == nil {
		return "absent"
	}
	if state.Type == "directory" {
		return "directory"
	}
	return fmt.Sprintf("%d bytes, sha256 %s", state.Size, state.SHA256)
}
