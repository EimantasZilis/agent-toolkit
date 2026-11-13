package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Apply rechecks a sealed preview under an exclusive lock and commits provider
// files and state together. Cancellation before commit triggers rollback.
// Recovery backups remain available when restoration cannot finish.
func (s *Service) Apply(ctx context.Context, p Preview, confirmed Confirmations) (status Status, err error) {
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	if p.Revision == "" || p.seal == "" || p.seal != previewSignature(p) {
		return Status{}, problem("invalid_preview", "", "request an unchanged preview before applying")
	}
	if len(p.Conflicts) > 0 {
		return Status{}, problem("ownership_conflict", p.Conflicts[0].Path, "resolve conflicts before applying; legacy linked installations require a separate migration")
	}
	if p.Confirmation && !confirmed.Removals {
		return Status{}, problem("confirmation_required", "", "confirm removals")
	}
	if _, _, _, err := s.state(); err != nil {
		return Status{}, err
	}
	stateDir := filepath.Join(s.Root, "config")
	stateExisted := exists(stateDir)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return Status{}, err
	}
	lock := filepath.Join(stateDir, "apply.lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		return Status{}, problem("locked", lock, "another apply is active")
	}
	defer func() {
		err = errors.Join(err, os.Remove(lock))
		if !stateExisted {
			if removeErr := os.Remove(stateDir); removeErr != nil && !os.IsNotExist(removeErr) && err != nil {
				err = errors.Join(err, removeErr)
			}
		}
	}()
	current, err := s.Preview(ctx, p.Request)
	if err != nil {
		return Status{}, err
	}
	if current.Revision != p.Revision || current.seal != p.seal {
		return Status{}, problem("stale_revision", "", "request a new preview before applying")
	}
	p = current
	tx, err := newTransaction(s.fail)
	if err != nil {
		return Status{}, err
	}
	committed := false
	defer func() {
		if !committed {
			err = errors.Join(err, tx.rollback())
		}
	}()
	settingsPath, inventoryPath := filepath.Join(stateDir, "settings.env"), filepath.Join(stateDir, "inventory.json")
	for _, path := range []string{settingsPath, inventoryPath} {
		if err := tx.backup(path); err != nil {
			return Status{}, err
		}
	}
	if confirmed.CreateOutput {
		if p.Desired.Output == "" {
			return Status{}, problem("invalid_settings", "", "directory creation requires an output path")
		}
		if err := tx.mkdir(p.Desired.Output); err != nil {
			return Status{}, err
		}
	}
	for _, provider := range []string{"codex", "claude"} {
		for _, op := range p.Operations {
			if err := ctx.Err(); err != nil {
				return Status{}, err
			}
			if op.Provider != provider || op.Action == "keep" {
				continue
			}
			if err := s.applyOperation(tx, p, op); err != nil {
				return Status{}, err
			}
		}
		if err := s.checkFailure("after-provider:" + provider); err != nil {
			return Status{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	applied := map[string][]string{}
	for _, provider := range []string{"codex", "claude"} {
		names, err := s.observed(provider)
		if err != nil {
			return Status{}, err
		}
		applied[provider] = names
	}
	if p.Request.Intent == Uninstall {
		if err := os.Remove(settingsPath); err != nil && !os.IsNotExist(err) {
			return Status{}, err
		}
		if err := s.checkFailure("after-settings"); err != nil {
			return Status{}, err
		}
		if err := os.Remove(inventoryPath); err != nil && !os.IsNotExist(err) {
			return Status{}, err
		}
	} else {
		settings := settingsBytes(p.Desired)
		inv, err := inventoryBytes(settings, applied, p.Source)
		if err != nil {
			return Status{}, err
		}
		if err := atomicWrite(settingsPath, settings, 0644); err != nil {
			return Status{}, err
		}
		if err := s.checkFailure("after-settings"); err != nil {
			return Status{}, err
		}
		if err := atomicWrite(inventoryPath, inv, 0644); err != nil {
			return Status{}, err
		}
	}
	if err := s.checkFailure("after-inventory"); err != nil {
		return Status{}, err
	}
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	status, err = s.Status(ctx)
	if err != nil {
		return Status{}, err
	}
	committed = true
	if err := os.RemoveAll(tx.root); err != nil {
		return status, fmt.Errorf("applied; clean up backups at %s: %w", tx.root, err)
	}
	return status, nil
}

func (s *Service) checkFailure(point string) error {
	if s.fail != nil {
		return s.fail(point)
	}
	return nil
}

func (s *Service) applyOperation(tx *transaction, p Preview, op Operation) error {
	if err := tx.mkdir(filepath.Dir(op.Path)); err != nil {
		return err
	}
	if strings.Contains(op.Action, "guidance") {
		return s.applyGuidance(tx, p, op)
	}
	if op.Action == "convert" {
		if p.Request.Intent != Migrate {
			return problem("invalid_operation", op.Path, "conversion requires explicit migration")
		}
		c, err := s.Catalog()
		if err != nil {
			return err
		}
		if err := s.validateLegacy(c, op.Provider, op.Name); err != nil {
			return err
		}
	} else if exists(op.Path) && !s.safeCopied(op.Path) {
		return problem("ownership_conflict", op.Path, "copied ownership changed; linked installations require a separate migration")
	}
	if err := tx.backup(op.Path); err != nil {
		return err
	}
	if err := os.RemoveAll(op.Path); err != nil {
		return err
	}
	if op.Action == "remove" {
		return nil
	}
	if op.Action != "install" && op.Action != "update" && op.Action != "convert" {
		return problem("invalid_operation", op.Path, op.Action)
	}
	if err := os.Mkdir(op.Path, 0755); err != nil {
		return err
	}
	files := p.payloads[op.Provider+"/"+op.Name]
	if files == nil {
		return problem("invalid_operation", op.Path, "missing reviewed payload")
	}
	if err := writePayload(op.Path, files); err != nil {
		return err
	}
	marker, err := ownershipBytes(files, p.Source)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(op.Path, markerName), marker, 0644)
}

func (s *Service) applyGuidance(tx *transaction, p Preview, op Operation) error {
	user, _, _, err := s.guidanceParts(op.Path)
	if err != nil {
		return problem("ownership_conflict", op.Path, err.Error())
	}
	marker := filepath.Join(filepath.Dir(op.Path), markerName)
	for _, path := range []string{op.Path, marker} {
		if err := tx.backup(path); err != nil {
			return err
		}
	}
	if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
		return err
	}
	if op.Action == "remove-guidance" {
		if err := os.Remove(op.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if op.Action == "strip-guidance" {
		return atomicWrite(op.Path, []byte(user), 0644)
	}
	if op.Action != "write-guidance" {
		return problem("invalid_operation", op.Path, op.Action)
	}
	block := p.guidance
	if user != "" {
		user += "\n"
	}
	if err := atomicWrite(op.Path, []byte(user+block), 0644); err != nil {
		return err
	}
	return atomicWrite(marker, []byte(s.Root+"\nGUIDANCE_BLOCK "+digest([]byte(block))+"\n"), 0644)
}
