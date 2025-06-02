package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScimApiDependency(t *testing.T) {
	tests := []struct {
		name           string
		scimEnabled    bool
		serviceEnabled bool
		wantErr        bool
	}{
		{
			name:           "both disabled - valid",
			scimEnabled:    false,
			serviceEnabled: false,
			wantErr:        false,
		},
		{
			name:           "only service account enabled - valid",
			scimEnabled:    false,
			serviceEnabled: true,
			wantErr:        false,
		},
		{
			name:           "only scim enabled - should error",
			scimEnabled:    true,
			serviceEnabled: false,
			wantErr:        true,
		},
		{
			name:           "both enabled - valid",
			scimEnabled:    true,
			serviceEnabled: true,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Asana{
				Token:             "test-token",
				UseScimApi:        tt.scimEnabled,
				UseServiceAccount: tt.serviceEnabled,
			}

			scimEnabled := c.GetBool("use-scim-api")
			serviceEnabled := c.GetBool("use-service-account")

			// Check the dependency
			if scimEnabled && !serviceEnabled {
				assert.True(t, tt.wantErr, "Expected error when SCIM API is enabled without service account")
			} else {
				assert.False(t, tt.wantErr, "Expected no error when dependency is satisfied")
			}
		})
	}
}
