package connector

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/conductorone/baton-asana/pkg/asana"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"

	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const (
	member = "Member"
	admin  = "Admin"
	guest  = "Guest"
)

var workspaceRoles = []string{
	admin,
	member,
	guest,
}

type workspaceResourceType struct {
	resourceType      *v2.ResourceType
	client            *asana.Client
	allowedWorkspaces *[]string
}

func (o *workspaceResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func workspaceBuilder(client *asana.Client, allowedWorkspaces *[]string) *workspaceResourceType {
	return &workspaceResourceType{
		resourceType:      resourceTypeWorkspace,
		client:            client,
		allowedWorkspaces: allowedWorkspaces,
	}
}

// Create a new connector resource for an Asana workspace.
func workspaceResource(ctx context.Context, workspace asana.Workspace) (*v2.Resource, error) {
	profile := make(map[string]interface{})
	profile["workspace_id"] = workspace.Gid
	profile["workspace_name"] = workspace.Name
	profile["is_organization"] = workspace.IsOrganization

	groupTrait := []rs.GroupTraitOption{
		rs.WithGroupProfile(profile),
	}
	workspaceOptions := []rs.ResourceOption{
		rs.WithAnnotation(
			&v2.ChildResourceType{ResourceTypeId: resourceTypeUser.Id},
			&v2.ChildResourceType{ResourceTypeId: resourceTypeTeam.Id},
		),
	}

	ret, err := rs.NewGroupResource(workspace.Name, resourceTypeWorkspace, workspace.Gid, groupTrait, workspaceOptions...)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *workspaceResourceType) List(ctx context.Context, resourceId *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if o.allowedWorkspaces == nil {
		return nil, "", nil, nil
	}

	rv := make([]*v2.Resource, 0, len(*o.allowedWorkspaces))
	for _, workspaceId := range *o.allowedWorkspaces {
		workspaceInfo, _, err := o.client.GetWorkspace(ctx, workspaceId)
		if err != nil {
			return nil, "", nil, err
		}
		wr, err := workspaceResource(ctx, workspaceInfo)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, wr)
	}

	return rv, "", nil, nil
}

func (o *workspaceResourceType) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	for _, role := range workspaceRoles {
		permissionOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(resourceTypeUser),
			ent.WithDescription(fmt.Sprintf("Role in %s Asana workspace", resource.DisplayName)),
			ent.WithDisplayName(fmt.Sprintf("%s Workspace %s", resource.DisplayName, role)),
		}

		permissionEn := ent.NewPermissionEntitlement(resource, role, permissionOptions...)
		rv = append(rv, permissionEn)
	}
	return rv, "", nil, nil
}

func (o *workspaceResourceType) Grants(ctx context.Context, resource *v2.Resource, pt *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	bag, err := parsePageToken(pt.Token, resource.Id)
	if err != nil {
		return nil, "", nil, err
	}

	workspaceTrait, err := rs.GetGroupTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	workspaceId, ok := rs.GetProfileStringValue(workspaceTrait.Profile, "workspace_id")
	if !ok {
		return nil, "", nil, fmt.Errorf("error fetching workspace_id from workspace profile")
	}

	workspaceMembership, offset, _, err := o.client.GetWorkspaceMemberships(ctx, asana.GetWorkspaceMembershipsVars{WorkspaceId: workspaceId, Limit: ResourcesPageSize, Offset: bag.PageToken()})
	if err != nil {
		return nil, "", nil, err
	}

	pageToken, err := bag.NextToken(offset)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Grant

	for _, workspaceMember := range workspaceMembership {
		var roleName string
		switch {
		case workspaceMember.IsActive:
			roleName = member
		case workspaceMember.IsAdmin:
			roleName = admin
		case workspaceMember.IsGuest:
			roleName = guest
		}
		workspaceMemberCopy := workspaceMember
		ur, err := userResource(ctx, &workspaceMemberCopy.User, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		permissionGrant := grant.NewGrant(resource, roleName, ur.Id)
		rv = append(rv, permissionGrant)
	}

	return rv, pageToken, nil, nil
}

func (o *workspaceResourceType) Grant(ctx context.Context, resource *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	if resource.Id.ResourceType == resourceTypeUser.Id {
		workspaceId := entitlement.Resource.Id.Resource
		userId := resource.Id.Resource

		err := o.client.AddUserToWorkspace(ctx, workspaceId, userId)
		if err != nil {
			if status.Code(err) == codes.PermissionDenied {
				return nil, nil, errors.Join(err, errors.New("user does not have permission to add user to workspace or the user was previous removed from the workspace"))
			}

			return nil, nil, err
		}

		workspaceEntitlement, err := getWorkspaceEntitlement(entitlement)
		if err != nil {
			return nil, nil, err
		}

		userRsId, err := rs.NewResourceID(resourceTypeUser, userId)
		if err != nil {
			return nil, nil, err
		}

		rv := []*v2.Grant{
			grant.NewGrant(resource, workspaceEntitlement, userRsId),
		}

		return rv, nil, nil
	}

	return nil, nil, fmt.Errorf("invalid resource type %s", resource.Id.ResourceType)
}

func (o *workspaceResourceType) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	if grant.Principal.Id.ResourceType == resourceTypeUser.Id {
		workspaceId := grant.Entitlement.Resource.Id.Resource
		userId := grant.Principal.Id.Resource

		err := o.client.RemoveUserToWorkspace(ctx, workspaceId, userId)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	return nil, fmt.Errorf("invalid resource type %s", grant.Principal.Id.ResourceType)
}

func getWorkspaceEntitlement(entitlement *v2.Entitlement) (string, error) {
	id := strings.Split(entitlement.Id, ":")

	if len(id) != 3 {
		return "", fmt.Errorf("invalid entitlement id: %s", entitlement.Id)
	}

	return id[2], nil
}
