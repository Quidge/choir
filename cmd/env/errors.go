package env

import (
	"fmt"
	"strings"

	"github.com/Quidge/choir/internal/state"
)

// VisibleStatuses are the statuses shown by default in `choir env list`.
var VisibleStatuses = []state.EnvironmentStatus{
	state.StatusProvisioning,
	state.StatusReady,
}

// isVisibleStatus returns true if the status is visible by default.
func isVisibleStatus(status state.EnvironmentStatus) bool {
	for _, s := range VisibleStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// FormatAmbiguousPrefixError formats an AmbiguousPrefixError into a helpful
// error message that shows all matching environments with their visibility status.
func FormatAmbiguousPrefixError(err *state.AmbiguousPrefixError) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ambiguous environment ID %q: matches %d environments\n", err.Prefix, len(err.Matches)))
	sb.WriteString("\nMatching environments:\n")

	for _, env := range err.Matches {
		shortID := state.ShortID(env.ID)
		visibility := "visible"
		if !isVisibleStatus(env.Status) {
			visibility = "hidden, use --all to see"
		}
		sb.WriteString(fmt.Sprintf("  %s  %s  (%s)\n", shortID, env.Status, visibility))
	}

	sb.WriteString("\nHint: use a longer prefix or run \"choir env list --all\" to see hidden environments")

	return fmt.Errorf("%s", sb.String())
}
