package config

import (
	"errors"

	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	TokenField = field.StringField(
		"token",
		field.WithDisplayName("Personal access token"),
		field.WithDescription("Your Asana API key (Personal Access Token or Service Account Token)"),
		field.WithRequired(true),
		field.WithIsSecret(true),
	)

	UseServiceAccountField = field.BoolField(
		"use-service-account",
		field.WithDisplayName("Is service account"),
		field.WithDescription("Set to true if using a service account token instead of a personal access token"),
	)

	DefaultWorkspaceIDField = field.StringField(
		"default-workspace-id",
		field.WithDisplayName("Default workspace"),
		field.WithDescription("The default workspace ID to use for account provisioning"),
	)

	UseScimApiField = field.BoolField(
		"use-scim-api",
		field.WithDisplayName("Use SCIM API"),
		field.WithDescription("Set to true to use the Asana SCIM API for enterprise license management and user provisioning"),
	)

	AsanaApiUrlField = field.StringField(
		"asana-api-url",
		field.WithDisplayName("Asana API URL"),
		field.WithDescription("Override the default Asana API URL (for testing with a mock server)"),
	)
)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	[]field.SchemaField{
		TokenField,
		UseServiceAccountField,
		DefaultWorkspaceIDField,
		UseScimApiField,
		AsanaApiUrlField,
	},
	field.WithConnectorDisplayName("Asana"),
	field.WithIconUrl("/static/app-icons/asana.svg"),
	field.WithConstraints(field.FieldsRequiredTogether(
		UseScimApiField,
		UseServiceAccountField,
	)),
)
