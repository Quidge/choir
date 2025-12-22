package backend

import (
	"fmt"
)

// BackendConfig contains configuration needed to initialize a backend.
type BackendConfig struct {
	// Name is the backend name (e.g., "local").
	Name string

	// Type is the backend type (e.g., "worktree", "lima").
	Type string

	// CPUs is the number of CPUs to allocate (VM backends only).
	CPUs int

	// Memory is the memory allocation (e.g., "4GB") (VM backends only).
	Memory string

	// Disk is the disk allocation (e.g., "50GB") (VM backends only).
	Disk string

	// VMType is the VM type for Lima (e.g., "vz", "qemu").
	VMType string
}

// BackendFactory is a function that creates a new backend instance.
type BackendFactory func(cfg BackendConfig) (Backend, error)

// registry holds the registered backend factories.
var registry = make(map[string]BackendFactory)

// Register registers a backend factory for the given type.
// This should be called during package init.
func Register(backendType string, factory BackendFactory) {
	registry[backendType] = factory
}

// Get returns a new backend instance for the given configuration.
// Returns an error if the backend type is not registered.
func Get(cfg BackendConfig) (Backend, error) {
	factory, ok := registry[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown backend type: %s", cfg.Type)
	}
	return factory(cfg)
}

// RegisteredTypes returns a list of all registered backend types.
func RegisteredTypes() []string {
	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
