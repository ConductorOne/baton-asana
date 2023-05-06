package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-asana/pkg/asana"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const (
	teamGuest         = "Guest"
	teamAdmin         = "Admin"
	teamLimitedAccess = "Limited Access"
	teamMember        = "Team Member"
)

var teamRoles = []string{
	teamGuest,
	teamAdmin,
	teamLimitedAccess,
	teamMember,
}

type teamResourceType struct {
	resourceType *v2.ResourceType
	client       *asana.Client
}

func (o *teamResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for an Asana team.
func teamResource(team *asana.Team, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"team_id":   team.Gid,
		"team_name": team.Name,
	}

	groupTraitOptions := []rs.GroupTraitOption{rs.WithGroupProfile(profile)}

	ret, err := rs.NewGroupResource(
		team.Name,
		resourceTypeTeam,
		team.Gid,
		groupTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *teamResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}

	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeTeam.Id})
	if err != nil {
		return nil, "", nil, err
	}

	teams, nextToken, _, err := o.client.GetTeams(ctx, asana.GetTeamsVars{WorkspaceId: parentId.Resource, Offset: bag.PageToken(), Limit: ResourcesPageSize})
	if err != nil {
		return nil, "", nil, fmt.Errorf("linear-connector: failed to list teams: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, team := range teams {
		teamCopy := team
		ur, err := teamResource(&teamCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}
	return rv, pageToken, nil, nil
}

func (o *teamResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	for _, role := range teamRoles {
		permissionOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(resourceTypeUser),
			ent.WithDescription(fmt.Sprintf("Role in %s Asana team", resource.DisplayName)),
			ent.WithDisplayName(fmt.Sprintf("%s Team %s", resource.DisplayName, role)),
		}

		permissionEn := ent.NewPermissionEntitlement(resource, role, permissionOptions...)
		rv = append(rv, permissionEn)
	}
	return rv, "", nil, nil
}

func (o *teamResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	bag, err := parsePageToken(token.Token, resource.Id)
	if err != nil {
		return nil, "", nil, err
	}

	teamTrait, err := rs.GetGroupTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	teamId, ok := rs.GetProfileStringValue(teamTrait.Profile, "team_id")
	if !ok {
		return nil, "", nil, fmt.Errorf("error fetching team_id from team profile")
	}

	teamMemberships, offset, _, err := o.client.GetTeamMemberships(ctx, asana.GetTeamMembershipsVars{TeamId: teamId, Limit: ResourcesPageSize, Offset: bag.PageToken()})
	if err != nil {
		return nil, "", nil, err
	}

	pageToken, err := bag.NextToken(offset)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Grant

	for _, teamMembership := range teamMemberships {
		var roleName string
		switch {
		case teamMembership.IsAdmin:
			roleName = teamAdmin
		case teamMembership.IsLimitedAccess:
			roleName = teamLimitedAccess
		case teamMembership.IsGuest:
			roleName = teamGuest
		default:
			roleName = teamMember
		}
		teamMembershipCopy := teamMembership
		ur, err := userResource(ctx, &teamMembershipCopy.User, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		permissionGrant := grant.NewGrant(resource, roleName, ur.Id)
		rv = append(rv, permissionGrant)
	}

	return rv, pageToken, nil, nil
}

func teamBuilder(client *asana.Client) *teamResourceType {
	return &teamResourceType{
		resourceType: resourceTypeTeam,
		client:       client,
	}
}
