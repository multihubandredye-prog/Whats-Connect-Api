package validations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pkgError "whats-connect-api2/pkg/error"
)

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration *int
		err      any
	}{
		{
			name:     "should success with nil duration",
			duration: nil,
			err:      nil,
		},
		{
			name:     "should success with zero duration",
			duration: func() *int { d := 0; return &d }(),
			err:      nil,
		},
		{
			name:     "should success with 24h duration",
			duration: func() *int { d := 86400; return &d }(),
			err:      nil,
		},
		{
			name:     "should success with 7d duration",
			duration: func() *int { d := 604800; return &d }(),
			err:      nil,
		},
		{
			name:     "should success with 90d duration",
			duration: func() *int { d := 7776000; return &d }(),
			err:      nil,
		},
		{
			name:     "should error with invalid duration",
			duration: func() *int { d := 12345; return &d }(),
			err:      pkgError.ValidationError("invalid duration. allowed values are 0 (disabled), 86400 (24 hours), 604800 (7 days), or 7776000 (90 days)"),
		},
		{
			name:     "should error with negative duration",
			duration: func() *int { d := -1; return &d }(),
			err:      pkgError.ValidationError("invalid duration. allowed values are 0 (disabled), 86400 (24 hours), 604800 (7 days), or 7776000 (90 days)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDuration(tt.duration)
			assert.Equal(t, tt.err, err)
		})
	}
}