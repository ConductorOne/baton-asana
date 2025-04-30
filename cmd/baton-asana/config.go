package main

import (
	"errors"

	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/spf13/viper"
)

var (
	TokenField = field.StringField(
		"token",
		field.WithRequired(true),
		field.WithDescription("Your Asana API key (Personal Access Token or Service Account Token)"),
	)

	UseServiceAccountField = field.BoolField(
		"use-service-account",
		field.WithDescription("Set to true if using a service account token instead of a personal access token"),
	)

	DefaultWorkspaceIDField = field.StringField(
		"default-workspace-id",
		field.WithDescription("The default workspace ID to use for account provisioning"),
	)

	UseScimApiField = field.BoolField(
		"use-scim-api",
		field.WithDescription("Set to true to use the Asana SCIM API for enterprise license management and user provisioning"),
	)

	// ConfigurationFields defines the external configuration required for the
	// connector to run. Note: these fields can be marked as optional or
	// required.
	ConfigurationFields = []field.SchemaField{
		TokenField,
		UseServiceAccountField,
		DefaultWorkspaceIDField,
		UseScimApiField,
	}

	// FieldRelationships defines relationships between the fields listed in
	// ConfigurationFields that can be automatically validated. For example, a
	// username and password can be required together, or an access token can be
	// marked as mutually exclusive from the username password pair.
	FieldRelationships = []field.SchemaFieldRelationship{}
)

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(v *viper.Viper) error {
	if v.GetString(TokenField.FieldName) == "" {
		return errors.New("token is required")
	}

	// If SCIM API is enabled, verify that service account is also enabled
	if v.GetBool(UseScimApiField.FieldName) && !v.GetBool(UseServiceAccountField.FieldName) {
		return errors.New("service account token is required when SCIM API is enabled")
	}

	return nil
}
