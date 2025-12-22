package backend

import (
	"errors"
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	resetRegistry()

	called := false
	factory := func(cfg BackendConfig) (Backend, error) {
		called = true
		if cfg.Name != "test-backend" {
			t.Errorf("expected Name 'test-backend', got %q", cfg.Name)
		}
		return nil, nil
	}

	Register("test", factory)

	_, err := Get(BackendConfig{Type: "test", Name: "test-backend"})
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if !called {
		t.Error("factory was not called")
	}
}

func TestGetUnknownBackendType(t *testing.T) {
	resetRegistry()

	_, err := Get(BackendConfig{Type: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown backend type, got nil")
	}

	expected := "unknown backend type: nonexistent"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetReturnsFactoryError(t *testing.T) {
	resetRegistry()

	expectedErr := errors.New("factory error")
	factory := func(cfg BackendConfig) (Backend, error) {
		return nil, expectedErr
	}

	Register("failing", factory)

	_, err := Get(BackendConfig{Type: "failing"})
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRegisteredTypes(t *testing.T) {
	resetRegistry()

	Register("alpha", func(cfg BackendConfig) (Backend, error) { return nil, nil })
	Register("beta", func(cfg BackendConfig) (Backend, error) { return nil, nil })

	types := RegisteredTypes()
	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(types))
	}

	found := make(map[string]bool)
	for _, typ := range types {
		found[typ] = true
	}

	if !found["alpha"] {
		t.Error("expected 'alpha' in registered types")
	}
	if !found["beta"] {
		t.Error("expected 'beta' in registered types")
	}
}

func TestRegisteredTypesEmpty(t *testing.T) {
	resetRegistry()

	types := RegisteredTypes()
	if len(types) != 0 {
		t.Errorf("expected 0 types, got %d", len(types))
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	resetRegistry()

	factory := func(cfg BackendConfig) (Backend, error) { return nil, nil }
	Register("duplicate", factory)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate registration, got none")
		}
		expected := `backend type "duplicate" already registered`
		if r != expected {
			t.Errorf("expected panic message %q, got %q", expected, r)
		}
	}()

	Register("duplicate", factory)
}
