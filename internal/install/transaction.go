package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type backup struct {
	path, slot         string
	present, directory bool
}
type transaction struct {
	root    string
	backups []backup
	fail    func(string) error
}

func newTransaction(fail func(string) error) (*transaction, error) {
	root, err := os.MkdirTemp("", "agent-toolkit-install-")
	if err != nil {
		return nil, err
	}
	return &transaction{root: root, fail: fail}, nil
}

// copyPath preserves regular files, modes, directories and link identities;
// it never follows links or opens special files.
func copyPath(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(source)
		if err != nil {
			return err
		}
		return os.Symlink(link, target)
	}
	if info.IsDir() {
		if err := os.Mkdir(target, 0700); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyPath(filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name())); err != nil {
				return err
			}
		}
		return os.Chmod(target, info.Mode().Perm())
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported file: %s", source)
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return err
	}
	return os.Chmod(target, info.Mode().Perm())
}

func (t *transaction) backup(path string) error {
	for _, record := range t.backups {
		if record.path == path {
			return nil
		}
	}
	record := backup{path: path, slot: filepath.Join(t.root, strconv.Itoa(len(t.backups))), present: exists(path)}
	if record.present {
		if err := copyPath(path, record.slot); err != nil {
			return fmt.Errorf("back up %s: %w", path, err)
		}
	}
	t.backups = append(t.backups, record)
	return nil
}

func (t *transaction) mkdir(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	if err := t.mkdir(filepath.Dir(path)); err != nil {
		return err
	}
	if err := os.Mkdir(path, 0755); err != nil {
		return err
	}
	t.backups = append(t.backups, backup{path: path, directory: true})
	return nil
}

func (t *transaction) rollback() error {
	var failures []error
	for i := len(t.backups) - 1; i >= 0; i-- {
		record := t.backups[i]
		if t.fail != nil {
			if err := t.fail("rollback:" + record.path); err != nil {
				failures = append(failures, err)
				continue
			}
		}
		if record.directory {
			// Preserve artifacts another process may have placed in a new folder.
			if err := os.Remove(record.path); err != nil && !os.IsNotExist(err) {
				failures = append(failures, err)
			}
			continue
		}
		if record.present && !exists(record.slot) {
			failures = append(failures, fmt.Errorf("missing backup: %s", record.slot))
			continue
		}
		if err := os.RemoveAll(record.path); err != nil {
			failures = append(failures, err)
			continue
		}
		if record.present {
			if err := copyPath(record.slot, record.path); err != nil {
				failures = append(failures, err)
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("rollback incomplete; recovery backups retained at %s: %w", t.root, errors.Join(failures...))
	}
	return os.RemoveAll(t.root)
}

func atomicWrite(path string, data []byte, mode os.FileMode) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".agent-toolkit-write-")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(f.Name()); removeErr != nil && !os.IsNotExist(removeErr) {
			err = errors.Join(err, removeErr)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return errors.Join(err, f.Close())
	}
	if err := f.Chmod(mode); err != nil {
		return errors.Join(err, f.Close())
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
