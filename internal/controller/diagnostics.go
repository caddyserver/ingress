package controller

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

const diagnosticTimeout = 60 * time.Second

func getCutoff() time.Time {
	return time.Now().Add(diagnosticTimeout * -1)
}

// Diagnostics is a simple map tracking when a diagnostic was last logged.
type Diagnostics map[string]time.Time

// Gc removes entries from the map that have timed out.
func (diags Diagnostics) Gc() {
	cutoff := getCutoff()
	for key, t := range diags {
		if t.Compare(cutoff) < 0 {
			delete(diags, key)
		}
	}
}

// Warnf formats the the given message and checks if it is present in the map.
// If not, the message will be logged at warning level and added to the map.
func (diags Diagnostics) Warnf(logger *zap.SugaredLogger, template string, args ...any) bool {
	key := fmt.Sprintf(template, args...)
	if t, ok := diags[key]; ok && t.Compare(getCutoff()) >= 0 {
		return false
	}
	diags[key] = time.Now()
	logger.Warnf(template, args...)
	return true
}
