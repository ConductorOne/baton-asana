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
	client            *asana.Client
	useServiceAccount bool
}

func (as *Asana) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(as.client),
		workspaceBuilder(as.client, as.useServiceAccount),
		teamBuilder(as.client),
	}
}

// Metadata returns metadata about the connector.
func (as *Asana) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Asana",
		Description: "Connector syncing users, teams and workspaces from Asana to Baton",
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
