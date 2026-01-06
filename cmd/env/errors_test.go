package env

import (
	"strings"
	"testing"
	"time"

	"github.com/Quidge/choir/internal/state"
)

func TestIsVisibleStatus(t *testing.T) {
	tests := []struct {
		status  state.EnvironmentStatus
		visible bool
	}{
		{state.StatusProvisioning, true},
		{state.StatusReady, true},
		{state.StatusFailed, false},
		{state.StatusRemoved, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := isVisibleStatus(tt.status); got != tt.visible {
				t.Errorf("isVisibleStatus(%q) = %v, want %v", tt.status, got, tt.visible)
			}
		})
	}
}

func TestFormatAmbiguousPrefixError(t *testing.T) {
	now := time.Now()

	t.Run("all visible", func(t *testing.T) {
		err := &state.AmbiguousPrefixError{
			Prefix: "abc",
			Matches: []*state.Environment{
				{ID: "abc123def456abc123def456abc12345", Status: state.StatusReady, CreatedAt: now},
				{ID: "abc456def789abc456def789abc45678", Status: state.StatusProvisioning, CreatedAt: now},
			},
		}

		formatted := FormatAmbiguousPrefixError(err)
		msg := formatted.Error()

		// Check header
		if !strings.Contains(msg, `ambiguous environment ID "abc"`) {
			t.Errorf("expected header with prefix, got: %s", msg)
		}
		if !strings.Contains(msg, "matches 2 environments") {
			t.Errorf("expected match count, got: %s", msg)
		}

		// Check environment listings
		if !strings.Contains(msg, "abc123def456") {
			t.Errorf("expected first short ID, got: %s", msg)
		}
		if !strings.Contains(msg, "abc456def789") {
			t.Errorf("expected second short ID, got: %s", msg)
		}

		// Check visibility labels - both should be visible
		if strings.Count(msg, "(visible)") != 2 {
			t.Errorf("expected 2 visible labels, got: %s", msg)
		}

		// Check hint
		if !strings.Contains(msg, "use a longer prefix") {
			t.Errorf("expected hint about longer prefix, got: %s", msg)
		}
		if !strings.Contains(msg, `choir env list --all`) {
			t.Errorf("expected hint about --all flag, got: %s", msg)
		}
	})

	t.Run("all hidden", func(t *testing.T) {
		err := &state.AmbiguousPrefixError{
			Prefix: "def",
			Matches: []*state.Environment{
				{ID: "def123abc456def123abc456def12345", Status: state.StatusFailed, CreatedAt: now},
				{ID: "def456abc789def456abc789def45678", Status: state.StatusRemoved, CreatedAt: now},
			},
		}

		formatted := FormatAmbiguousPrefixError(err)
		msg := formatted.Error()

		// Both should be hidden
		if strings.Contains(msg, "(visible)") {
			t.Errorf("expected no visible labels, got: %s", msg)
		}
		if strings.Count(msg, "hidden, use --all to see") != 2 {
			t.Errorf("expected 2 hidden labels, got: %s", msg)
		}
	})

	t.Run("mixed visible and hidden", func(t *testing.T) {
		err := &state.AmbiguousPrefixError{
			Prefix: "44",
			Matches: []*state.Environment{
				{ID: "440707e51272440707e51272440707e5", Status: state.StatusReady, CreatedAt: now},
				{ID: "4406ed1dd8ba4406ed1dd8ba4406ed1d", Status: state.StatusFailed, CreatedAt: now},
			},
		}

		formatted := FormatAmbiguousPrefixError(err)
		msg := formatted.Error()

		// Check the exact scenario from the issue
		if !strings.Contains(msg, `ambiguous environment ID "44"`) {
			t.Errorf("expected header with prefix, got: %s", msg)
		}

		// One visible, one hidden
		if strings.Count(msg, "(visible)") != 1 {
			t.Errorf("expected 1 visible label, got: %s", msg)
		}
		if strings.Count(msg, "hidden, use --all to see") != 1 {
			t.Errorf("expected 1 hidden label, got: %s", msg)
		}

		// Check status is shown
		if !strings.Contains(msg, "ready") {
			t.Errorf("expected ready status, got: %s", msg)
		}
		if !strings.Contains(msg, "failed") {
			t.Errorf("expected failed status, got: %s", msg)
		}
	})

	t.Run("large number of matches", func(t *testing.T) {
		matches := make([]*state.Environment, 5)
		for i := range matches {
			matches[i] = &state.Environment{
				ID:        "aaa123456789012345678901234567" + string(rune('0'+i)),
				Status:    state.StatusReady,
				CreatedAt: now,
			}
		}

		err := &state.AmbiguousPrefixError{
			Prefix:  "aaa",
			Matches: matches,
		}

		formatted := FormatAmbiguousPrefixError(err)
		msg := formatted.Error()

		if !strings.Contains(msg, "matches 5 environments") {
			t.Errorf("expected 5 matches, got: %s", msg)
		}
	})
}
