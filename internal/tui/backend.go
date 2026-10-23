// Package tui implements the repository installation setup and settings UI.
package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

const protocolVersion = 1

type settings struct {
	Providers    []string `json:"providers"`
	Requested    []string `json:"requested_skills"`
	Retained     []string `json:"retained_helpers"`
	Output       string   `json:"output_path"`
	Concise      bool     `json:"concise_output"`
	FullTemplate bool     `json:"full_template"`
}

type skill struct {
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Helpers  []string `json:"required_helpers"`
}

type bundle struct {
	Name   string   `json:"name"`
	Skills []string `json:"skills"`
}

type operation struct {
	Provider string `json:"provider"`
	Action   string `json:"action"`
	Name     string `json:"name"`
	Path     string `json:"path"`
}

type selection struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type response struct {
	Protocol     int                 `json:"protocol_version"`
	OK           bool                `json:"ok"`
	Kind         string              `json:"kind"`
	Revision     string              `json:"revision"`
	Desired      settings            `json:"desired"`
	Observed     map[string][]string `json:"observed"`
	Applied      map[string][]string `json:"applied"`
	Skills       []skill             `json:"skills"`
	Bundles      []bundle            `json:"bundles"`
	Selection    []selection         `json:"selection"`
	Operations   []operation         `json:"operations"`
	Conflicts    []operation         `json:"conflicts"`
	Confirmation bool                `json:"confirmation_required"`
	Native       struct {
		Message string `json:"message"`
	} `json:"native_overlap"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type backend interface {
	run(string, []string) (response, error)
}

type shellBackend struct{ root string }

func (b shellBackend) run(action string, args []string) (response, error) {
	argv := append([]string{action, "--protocol", strconv.Itoa(protocolVersion)}, args...)
	// Do not kill apply on UI cancellation: Bash must finish or perform rollback.
	cmd := exec.Command(filepath.Join(b.root, "scripts", "managed.sh"), argv...)
	cmd.Dir = b.root
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	r, decodeErr := decodeResponse(stdout.Bytes(), action)
	if err != nil || decodeErr != nil {
		return r, fmt.Errorf("%s: %w\n%s", action, firstError(err, decodeErr), stderr.String())
	}
	return r, nil
}

func firstError(a, b error) error {
	if b != nil {
		return b
	}
	return a
}

func decodeResponse(data []byte, kind string) (response, error) {
	var r response
	if err := json.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("decode backend response: %w", err)
	}
	if r.Protocol != protocolVersion {
		return r, fmt.Errorf("unsupported protocol %d (need %d)", r.Protocol, protocolVersion)
	}
	if !r.OK {
		return r, fmt.Errorf("%s: %s", r.Error.Code, r.Error.Message)
	}
	if r.Kind != kind {
		return r, fmt.Errorf("unexpected backend response %q, expected %q", r.Kind, kind)
	}
	if kind != "catalog" && r.Revision == "" {
		return r, fmt.Errorf("backend response has no revision")
	}
	return r, nil
}

func (s settings) arguments(removed []string) []string {
	args := []string{}
	if len(s.Providers) == 0 {
		args = append(args, "--no-providers")
	}
	for _, p := range s.Providers {
		args = append(args, "--provider", p)
	}
	if len(s.Requested) == 0 {
		args = append(args, "--no-skills")
	}
	for _, name := range s.Requested {
		args = append(args, "--skill", name)
	}
	for _, name := range removed {
		args = append(args, "--remove-skill", name)
	}
	return append(args, "--output-path", s.Output, "--concise", strconv.FormatBool(s.Concise), "--full-template", strconv.FormatBool(s.FullTemplate))
}

func savedInstallation(root string) (bool, error) {
	_, err := os.Stat(filepath.Join(root, "config", "settings.env"))
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
