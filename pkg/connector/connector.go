package connector

import (
	"context"
	"fmt"

	"github.com/ConductorOne/baton-asana/pkg/asana"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

var allowedWorkspaces []string

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
	allowedWorkspaces *[]string
}

func (as *Asana) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(as.client),
		workspaceBuilder(as.client, as.allowedWorkspaces),
		teamBuilder(as.client),
	}
}

// Metadata returns metadata about the connector.
func (as *Asana) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Asana",
	}, nil
}

// Validate hits the Asana API to validate that the API key passed has admin rights.
func (as *Asana) Validate(ctx context.Context) (annotations.Annotations, error) {
	workspaceMemberships, err := as.client.AuthCheck(ctx)
	if err != nil {
		return nil, fmt.Errorf("linear-connector: failed to authenticate. Error: %w", err)
	}

	for _, workspaceMembership := range workspaceMemberships {
		if !workspaceMembership.IsGuest {
			allowedWorkspaces = append(allowedWorkspaces, workspaceMembership.Workspace.Gid)
		}
	}
	return nil, nil
}

// New returns the Asana connector.
func New(ctx context.Context, accessToken string) (*Asana, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	return &Asana{
		client:            asana.NewClient(accessToken, httpClient),
		allowedWorkspaces: &allowedWorkspaces,
	}, nil
}
