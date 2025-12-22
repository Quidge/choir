package backend

import (
	"fmt"
	"sync"
)

// BackendConfig contains configuration needed to initialize a backend.
// This is distinct from config.CreateConfig: BackendConfig is used for
// backend initialization/registration (one-time setup), while CreateConfig
// is passed to Backend.Create() for each workspace creation.
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

var (
	// registry holds the registered backend factories.
	registry = make(map[string]BackendFactory)

	// registryMu protects concurrent access to the registry.
	registryMu sync.RWMutex
)

// Register registers a backend factory for the given type.
// This should be called during package init.
// Panics if the same backend type is registered twice.
func Register(backendType string, factory BackendFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[backendType]; exists {
		panic(fmt.Sprintf("backend type %q already registered", backendType))
	}
	registry[backendType] = factory
}

// Get returns a new backend instance for the given configuration.
// Returns an error if the backend type is not registered.
func Get(cfg BackendConfig) (Backend, error) {
	registryMu.RLock()
	factory, ok := registry[cfg.Type]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown backend type: %s", cfg.Type)
	}
	return factory(cfg)
}

// RegisteredTypes returns a list of all registered backend types.
func RegisteredTypes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}

// resetRegistry clears all registered backends. Only for testing.
func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]BackendFactory)
}
