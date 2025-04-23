package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-asana/pkg/asana"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
	}
	resourceTypeWorkspace = &v2.ResourceType{
		Id:          "workspace",
		DisplayName: "Workspace",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
	}
	resourceTypeTeam = &v2.ResourceType{
		Id:          "team",
		DisplayName: "Team",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
	}
)

type Asana struct {
	client                  *asana.Client
	useServiceAccount       bool
	accountCreationSettings AccountCreationSettings
}

func (as *Asana) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(as.client, as.accountCreationSettings),
		workspaceBuilder(as.client, as.useServiceAccount),
		teamBuilder(as.client),
	}
}

// Metadata returns metadata about the connector.
func (as *Asana) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Asana",
		Description: "Connector syncing users, teams and workspaces from Asana to Baton",
		AccountCreationSchema: &v2.ConnectorAccountCreationSchema{
			FieldMap: map[string]*v2.ConnectorAccountCreationSchema_Field{
				"first_name": {
					DisplayName: "First Name",
					Required:    true,
					Description: "The first name of the user.",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "First Name",
					Order:       1,
				},
				"last_name": {
					DisplayName: "Last Name",
					Required:    false,
					Description: "The last name of the user.",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "Last Name",
					Order:       2,
				},
				"email": {
					DisplayName: "Email",
					Required:    true,
					Description: "The email address of the user. This will be used for login.",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "user@example.com",
					Order:       3,
				},
				"workspace_id": {
					DisplayName: "Workspace ID",
					Required:    false,
					Description: "The Asana workspace ID to add the user to. If not provided, the default workspace ID configured for the connector will be used.",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "1234567890",
					Order:       4,
				},
			},
		},
	}, nil
}

// Validate hits the Asana API to validate that the API key passed has admin rights.
func (as *Asana) Validate(ctx context.Context) (annotations.Annotations, error) {
	if as.useServiceAccount {
		// For service account tokens, validate with ListAllWorkspaces
		_, err := as.client.ListAllWorkspaces(ctx)
		if err != nil {
			return nil, fmt.Errorf("baton-asana: failed to authenticate with service account token. Error: %w", err)
		}
	} else {
		// For regular user tokens, validate with AuthCheck
		_, err := as.client.AuthCheck(ctx)
		if err != nil {
			return nil, fmt.Errorf("baton-asana: failed to authenticate. Error: %w", err)
		}
	}

	return nil, nil
}

// New returns the Asana connector.
func New(ctx context.Context, accessToken string, opts ...Option) (*Asana, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	uhttpClient, err := uhttp.NewBaseHttpClientWithContext(ctx, httpClient)
	if err != nil {
		return nil, err
	}

	c := &Asana{
		client: asana.NewClient(accessToken, uhttpClient),
		accountCreationSettings: AccountCreationSettings{
			DefaultWorkspaceID: "", // This will need to be provided via options
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Option allows for configuration of the Asana connector.
type Option func(*Asana)

// WithServiceAccount configures the connector to use a service account token.
func WithServiceAccount(useServiceAccount bool) Option {
	return func(a *Asana) {
		a.useServiceAccount = useServiceAccount
	}
}

// WithDefaultWorkspaceID configures the default workspace ID to use for account creation.
func WithDefaultWorkspaceID(workspaceID string) Option {
	return func(a *Asana) {
		a.accountCreationSettings.DefaultWorkspaceID = workspaceID
	}
}
