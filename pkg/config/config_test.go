package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name        string
	config      map[string]string
	expectError bool
}

func TestValidateConfig(t *testing.T) {
	tests := []testCase{
		{
			name: "valid config with token",
			config: map[string]string{
				"token": "test-token",
			},
			expectError: false,
		},
		{
			name:        "missing token",
			config:      map[string]string{},
			expectError: true,
		},
		{
			name: "scim api enabled without service account",
			config: map[string]string{
				"token":               "test-token",
				"use-scim-api":        "true",
				"use-service-account": "false",
			},
			expectError: true,
		},
		{
			name: "valid config with scim and service account",
			config: map[string]string{
				"token":               "test-token",
				"use-scim-api":        "true",
				"use-service-account": "true",
			},
			expectError: false,
		},
		{
			name: "valid config with optional fields",
			config: map[string]string{
				"token":                "test-token",
				"default-workspace-id": "123456",
				"asana-api-url":        "https://custom.asana.com",
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new config instance
			c := &Asana{}

			// Set the test values
			for k, v := range tc.config {
				switch k {
				case "token":
					c.Token = v
				case "use-scim-api":
					c.UseScimApi = v == "true"
				case "use-service-account":
					c.UseServiceAccount = v == "true"
				case "default-workspace-id":
					c.DefaultWorkspaceId = v
				case "asana-api-url":
					c.AsanaApiUrl = v
				}
			}

			// Validate the config
			err := ValidateConfig(c)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
