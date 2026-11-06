package native

import (
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

var sourceInputs = []string{".agents", "cmd", "internal", "go.mod", "go.sum", "LICENSE"}
var commitPattern = regexp.MustCompile(`^([a-f0-9]{40}|[a-f0-9]{64})$`)

type snapshotRecord struct {
	Schema  int               `json:"schema_version"`
	Commit  string            `json:"source_commit"`
	Version string            `json:"version"`
	Files   map[string]string `json:"files"`
	Modes   map[string]int    `json:"modes"`
}

func gitQuery(root string, args ...string) (string, error) {
	base := []string{"--no-optional-locks", "-C", root}
	cmd := exec.Command("git", append(base, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}

func sourceCommit(root string) (string, error) {
	top, err := gitQuery(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	top, err = filepath.EvalSymlinks(top)
	if err != nil {
		return "", err
	}
	if canonical != top {
		return "", fmt.Errorf("snapshot source must be a Git repository root")
	}

	args := []string{"-c", "core.filemode=true", "diff", "--no-ext-diff", "--no-textconv", "--exit-code", "HEAD", "--"}
	if _, err := gitQuery(root, append(args, sourceInputs...)...); err != nil {
		return "", fmt.Errorf("commit source inputs before generating a snapshot: %w", err)
	}
	args = []string{"ls-tree", "-r", "-z", "--full-tree", "HEAD", "--"}
	listing, err := gitQuery(root, append(args, sourceInputs...)...)
	if err != nil {
		return "", err
	}
	type sourceFile struct{ mode, blob string }
	tracked := map[string]sourceFile{}
	for _, row := range strings.Split(listing, "\x00") {
		if row == "" {
			continue
		}
		header, name, ok := strings.Cut(row, "\t")
		fields := strings.Fields(header)
		if !ok || len(fields) != 3 || (fields[0] != "100644" && fields[0] != "100755") || fields[1] != "blob" || !safeName(name) {
			return "", fmt.Errorf("unsafe committed source input: %s", row)
		}
		tracked[name] = sourceFile{fields[0], fields[2]}
	}

	for _, name := range sourceInputs {
		err := filepath.WalkDir(filepath.Join(root, name), func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.Type()&os.ModeSymlink != 0 || (!e.IsDir() && !e.Type().IsRegular()) {
				return fmt.Errorf("unsafe source input: %s", path)
			}
			if e.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			expected, ok := tracked[filepath.ToSlash(rel)]
			if !ok {
				return fmt.Errorf("untracked or ignored source input: %s", path)
			}
			info, err := e.Info()
			if err != nil {
				return err
			}
			actual := "100644"
			if info.Mode()&0111 != 0 {
				actual = "100755"
			}
			if actual != expected.mode {
				return fmt.Errorf("source mode differs from HEAD: %s", path)
			}

			// Compare raw bytes as well as Git's diff. Skip-worktree and
			// assume-unchanged flags must not weaken source provenance.
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			object := append([]byte(fmt.Sprintf("blob %d\x00", len(data))), data...)
			var actualBlob string
			if len(expected.blob) == 64 {
				actualBlob = fmt.Sprintf("%x", sha256.Sum256(object))
			} else {
				// SHA-1 is used only to match Git repositories using that object format.
				actualBlob = fmt.Sprintf("%x", sha1.Sum(object))
			}
			if actualBlob != expected.blob {
				return fmt.Errorf("source bytes differ from HEAD: %s", path)
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return gitQuery(root, "rev-parse", "HEAD")
}

// Snapshot builds a combined marketplace from HEAD-matching source inputs.
// It refuses existing output and checks source cleanliness before and after rendering.
func Snapshot(source, version, destination string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("version must be X.Y.Z without leading zeros")
	}
	root, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	commit, err := sourceCommit(root)
	if err != nil {
		return err
	}
	c, err := catalog.Load(root)
	if err != nil {
		return err
	}
	return stage(destination, func(staged string) error {
		if err := populate(c, version, staged); err != nil {
			return err
		}
		var r release
		if err := readJSON(filepath.Join(staged, "release.json"), &r); err != nil {
			return err
		}
		selections := []string{}
		for name := range r.Catalog {
			selections = append(selections, name)
		}
		for name := range r.Bundles {
			selections = append(selections, name)
		}
		sort.Strings(selections)

		for _, provider := range providers {
			if err := writeMarketplace(staged, provider, selections, r.Bundles, true); err != nil {
				return err
			}
			for _, obsolete := range []string{strings.SplitN(marketplacePath(provider), "/", 2)[0], "zips"} {
				if err := os.RemoveAll(filepath.Join(staged, provider, obsolete)); err != nil {
					return err
				}
			}
		}
		for i := range r.Packages {
			r.Packages[i].Archive = ""
		}
		if err := writeJSON(filepath.Join(staged, "release.json"), r); err != nil {
			return err
		}
		files, modes, err := inventory(staged, true)
		if err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(staged, "snapshot.json"), snapshotRecord{1, commit, version, files, modes}); err != nil {
			return err
		}
		after, err := sourceCommit(root)
		if err != nil {
			return err
		}
		if after != commit {
			return fmt.Errorf("source commit changed during snapshot generation")
		}
		if err := validateTree(staged, true, root); err != nil {
			return err
		}
		return ValidateSnapshot(staged)
	})
}

// ValidateSnapshot checks a relocated or published snapshot without its source.
// Root Git administration is excluded; payload hashes, modes and catalogs are exact.
func ValidateSnapshot(root string) error {
	if err := validateTree(root, true, ""); err != nil {
		return err
	}
	var record snapshotRecord
	if err := readJSON(filepath.Join(root, "snapshot.json"), &record); err != nil {
		return err
	}
	if record.Schema != 1 || !commitPattern.MatchString(record.Commit) {
		return fmt.Errorf("invalid snapshot schema or source commit provenance")
	}
	files, modes, err := inventory(root, true)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(files, record.Files) {
		return fmt.Errorf("snapshot file inventory mismatch")
	}
	if !reflect.DeepEqual(modes, record.Modes) {
		return fmt.Errorf("snapshot permission inventory mismatch")
	}
	r, err := validateRelease(root, true)
	if err != nil {
		return err
	}
	if r.Version != record.Version {
		return fmt.Errorf("snapshot version mismatch")
	}
	return nil
}
