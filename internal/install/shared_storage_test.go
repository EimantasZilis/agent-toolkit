package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSharedStorageRenderingAndReload(t *testing.T) {
	for _, split := range []bool{false, true} {
		for _, guidance := range []bool{false, true} {
			t.Run(strings.Join([]string{map[bool]string{false: "shared", true: "split"}[split], map[bool]string{false: "skill-only", true: "guidance"}[guidance]}, "/"), func(t *testing.T) {
				s := testService(t)
				root := filepath.Join(s.Root, "shared artifacts")
				desired := Settings{Providers: []string{"codex", "claude"}, Requested: []string{"ticket-to-plan"}, Output: root, SharedStorage: true, SeparateByProvider: split, FullTemplate: guidance}
				plan := previewFor(t, s, Request{Desired: desired})
				if exists(root) {
					t.Fatal("preview wrote output root")
				}
				status, err := s.Apply(context.Background(), plan, Confirmations{CreateOutput: true})
				if err != nil {
					t.Fatal(err)
				}
				if !status.Desired.SharedStorage || status.Desired.SeparateByProvider != split {
					t.Fatal("shared settings lost", status.Desired)
				}
				for _, id := range desired.Providers {
					skillRoot, guide := s.destinations(id)
					paths := []string{guide}
					want := root + "/<repo-slug>/<skill-name>/<branch-slug>/"
					if split {
						want += id + "/"
					}
					for _, path := range paths {
						data, err := os.ReadFile(path)
						if err != nil || !strings.Contains(string(data), want) {
							t.Fatalf("wrong layout in %s: %v\n%s", path, err, data)
						}
					}
					data, err := os.ReadFile(filepath.Join(skillRoot, "ticket-to-plan/SKILL.md"))
					if err != nil || !strings.Contains(string(data), s.Root+"/.agents/global/markdown-output.md") || strings.Contains(string(data), "Do not prefer the current") {
						t.Fatalf("skill must link shared instructions: %v\n%s", err, data)
					}
				}
				applyFor(t, s, previewFor(t, s, Request{Intent: Refresh}))
			})
		}
	}
}
