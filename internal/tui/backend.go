// Package tui implements the repository installation setup and settings UI.
package tui

import (
	"context"
	"fmt"
	"sync"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
	"github.com/EimantasZilis/agent-toolkit/internal/install"
)

type settings = install.Settings
type operation = install.Operation
type selection = install.Selection

type skill struct {
	Name, Category string
	Helpers        []string
}
type bundle struct {
	Name   string
	Skills []string
}

// response is a presentation projection; all mutations use the typed plan.
type response struct {
	Desired               settings
	Observed, Applied     map[string][]string
	Skills                []skill
	Bundles               []bundle
	Selection             []selection
	Operations, Conflicts []operation
	Changes               []install.ContentChange
	Confirmation          bool
	Native                install.NativeOverlap
	Plan                  install.Preview
	Installed             bool
	Linked                bool
	Revision              string
}

type backend interface {
	Catalog() (*catalog.Catalog, error)
	Status(context.Context) (install.Status, error)
	Preview(context.Context, install.Request) (install.Preview, error)
	Apply(context.Context, install.Preview, install.Confirmations) (install.Status, error)
}

func catalogResponse(c *catalog.Catalog) response {
	r := response{}
	for _, s := range c.Skills {
		r.Skills = append(r.Skills, skill{s.Name, s.Category, s.Helpers})
	}
	for _, name := range c.BundleNames() {
		r.Bundles = append(r.Bundles, bundle{name, c.Bundles[name]})
	}
	return r
}

func statusResponse(s install.Status) response {
	return response{Desired: s.Desired, Observed: s.Observed, Applied: s.Applied, Native: s.Native, Installed: s.Installed, Linked: s.Linked, Revision: s.Revision}
}

func previewResponse(p install.Preview) response {
	return response{Desired: p.Desired, Selection: p.Selection, Operations: p.Operations, Conflicts: p.Conflicts, Changes: p.Changes, Confirmation: p.Confirmation, Native: p.Native, Revision: p.Revision, Plan: p}
}

// trackedBackend prevents new mutations after shutdown and waits for an active
// transaction when the terminal closes before its result can reach the model.
type trackedBackend struct {
	backend
	mu      sync.Mutex
	closing bool
	active  sync.WaitGroup
}

func (b *trackedBackend) Apply(ctx context.Context, p install.Preview, c install.Confirmations) (install.Status, error) {
	b.mu.Lock()
	if b.closing {
		b.mu.Unlock()
		return install.Status{}, fmt.Errorf("UI closed before apply started")
	}
	b.active.Add(1)
	b.mu.Unlock()
	defer b.active.Done()
	return b.backend.Apply(ctx, p, c)
}

func (b *trackedBackend) close() {
	b.mu.Lock()
	b.closing = true
	b.mu.Unlock()
	b.active.Wait()
}
