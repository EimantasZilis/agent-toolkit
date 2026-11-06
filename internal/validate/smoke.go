package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/EimantasZilis/agent-toolkit/internal/testutil"
	"github.com/charmbracelet/x/ansi"
)

// SmokeSource launches current source twice under a real terminal in an isolated
// spaced checkout. A visible source edit detects stale applications; quitting
// must leave settings and provider files unchanged on both launches.
func SmokeSource(ctx context.Context, source string) error {
	abs, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	base, err := os.MkdirTemp("", "toolkit source smoke ")
	if err != nil {
		return err
	}
	defer os.RemoveAll(base)
	root := filepath.Join(base, "checkout with spaces")
	for _, name := range []string{".agents", "cmd", "internal"} {
		if err := os.CopyFS(filepath.Join(root, name), os.DirFS(filepath.Join(abs, name))); err != nil {
			return err
		}
	}
	for _, name := range []string{"Makefile", "go.mod", "go.sum", "LICENSE"} {
		data, err := os.ReadFile(filepath.Join(abs, name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0644); err != nil {
			return err
		}
	}

	data, err := exec.CommandContext(ctx, "go", "env", "-json", "GOPATH", "GOCACHE", "GOMODCACHE").Output()
	if err != nil {
		return err
	}
	values := map[string]string{}
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	values["TERM"] = "xterm-256color"
	values["GOWORK"] = "off"
	watched := []string{filepath.Join(root, "config"), filepath.Join(root, ".env")}
	for _, name := range []string{"HOME", "AGENTS_HOME", "CODEX_HOME", "CLAUDE_HOME"} {
		path := filepath.Join(base, strings.ToLower(name))
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
		values[name] = path
		if name != "HOME" {
			watched = append(watched, path)
		}
	}
	env := testutil.Environment(values)
	before, err := testutil.Fingerprint(watched...)
	if err != nil {
		return err
	}
	for _, theme := range []string{"Midnight", "SourceChanged"} {
		if theme == "SourceChanged" {
			path := filepath.Join(root, "internal/tui/model.go")
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !strings.Contains(string(data), "Midnight") {
				return fmt.Errorf("source smoke theme fixture is missing")
			}
			if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(data), "Midnight", theme)), 0644); err != nil {
				return err
			}
		}
		launch, cancel := context.WithTimeout(ctx, 90*time.Second)
		cmd := exec.Command("make", "run")
		cmd.Dir, cmd.Env = root, env
		quitting, discarded := false, false
		output, err := testutil.RunTerminal(launch, cmd, func(output string) string {
			plain := ansi.Strip(output)
			if !quitting && strings.Contains(plain, "LLM providers") && strings.Contains(plain, "q quit") && strings.Contains(plain, theme) {
				quitting = true
				return "q"
			}
			if quitting && !discarded && strings.Contains(plain, "Discard") {
				discarded = true
				return "d"
			}
			return ""
		})
		cancel()
		if err != nil || !quitting {
			if len(output) > 4000 {
				output = output[len(output)-4000:]
			}
			return fmt.Errorf("source startup/quit failed (%s, quit=%t): %v\n%s", theme, quitting, err, output)
		}
		after, err := testutil.Fingerprint(watched...)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(before, after) {
			return fmt.Errorf("source startup/quit changed provider files or settings")
		}
	}
	return nil
}
