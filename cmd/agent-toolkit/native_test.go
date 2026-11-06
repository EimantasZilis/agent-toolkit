package main

import (
	"os"
	"testing"
)

func TestNativeArgumentsRejectWithoutWrites(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"publish"},
		{"build"},
		{"build", "--version", "1.0.0"},
		{"build", "--version", "1.0.0", "--output", "unused", "extra"},
		{"validate"},
		{"validate", "one", "two"},
		{"snapshot"},
		{"snapshot", "--version", "1.0.0"},
		{"validate-snapshot"},
		{"validate-snapshot", "one", "two"},
	} {
		t.Run("arguments", func(t *testing.T) {
			root := t.TempDir()
			if err := runNative(root, args); err == nil {
				t.Fatalf("accepted arguments: %v", args)
			}

			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("invalid arguments wrote files: %v %v", entries, err)
			}
		})
	}
}
