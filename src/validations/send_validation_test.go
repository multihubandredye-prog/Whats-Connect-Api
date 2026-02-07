package validations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
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
			err:      pkgError.ValidationError("Duração inválida. Os valores permitidos são 0 (desativado), 86400 (24 horas), 604800 (7 dias), ou 7776000 (90 dias)"),
		},
		{
			name:     "should error with negative duration",
			duration: func() *int { d := -1; return &d }(),
			err:      pkgError.ValidationError("Duração inválida. Os valores permitidos são 0 (desativado), 86400 (24 horas), 604800 (7 dias), ou 7776000 (90 dias)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDuration(tt.duration)
			assert.Equal(t, tt.err, err)
		})
	}
}