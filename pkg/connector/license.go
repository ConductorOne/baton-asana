package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-asana/pkg/asana"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

// Known license types.
const (
	licenseTypeEnterprise = "enterprise"
	licenseTypeViewOnly   = "view only"
)

type licenseResourceType struct {
	resourceType *v2.ResourceType
	client       *asana.Client
}

func (o *licenseResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for an Asana license.
func licenseResource(ctx context.Context, licenseType string) (*v2.Resource, error) {
	displayName := ""
	switch licenseType {
	case licenseTypeEnterprise:
		displayName = "Enterprise License"
	case licenseTypeViewOnly:
		displayName = "View Only License"
	default:
		displayName = licenseType + " License"
	}

	profile := map[string]interface{}{
		"license_type": licenseType,
	}

	ret, err := rs.NewRoleResource(
		displayName,
		resourceTypeLicense,
		licenseType,
		[]rs.RoleTraitOption{},
		rs.WithResourceProfile(profile),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *licenseResourceType) List(ctx context.Context, _ *v2.ResourceId, _ *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	// Return the two known license types
	enterpriseLicense, err := licenseResource(ctx, licenseTypeEnterprise)
	if err != nil {
		return nil, "", nil, err
	}

	viewOnlyLicense, err := licenseResource(ctx, licenseTypeViewOnly)
	if err != nil {
		return nil, "", nil, err
	}

	return []*v2.Resource{enterpriseLicense, viewOnlyLicense}, "", nil, nil
}

func (o *licenseResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	// Create a license-based entitlement
	entitlement := &v2.Entitlement{
		Resource: &v2.Resource{
			Id: resource.Id,
		},
		Id:          resource.Id.Resource,
		DisplayName: fmt.Sprintf("%s Assignment", resource.DisplayName),
		Description: fmt.Sprintf("Grants a %s to a user", resource.DisplayName),
	}

	return []*v2.Entitlement{entitlement}, "", nil, nil
}

func (o *licenseResourceType) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	// License grants will be emitted by the enterprise user resource type
	return nil, "", nil, nil
}

// Grant implements connectorbuilder.ResourceGranter interface.
func (o *licenseResourceType) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	// Validate that principal is a user
	if principal.Id.ResourceType != resourceTypeUser.Id {
		return nil, fmt.Errorf("principal is not a user")
	}

	// Validate that entitlement is for a license
	if entitlement.Resource.Id.ResourceType != resourceTypeLicense.Id {
		return nil, fmt.Errorf("entitlement is not for a license")
	}

	// Get user ID from principal resource
	userID := principal.Id.Resource

	// Get license type from entitlement resource
	licenseType := entitlement.Resource.Id.Resource

	// Validate license type
	switch licenseType {
	case licenseTypeEnterprise, licenseTypeViewOnly:
		// Valid license types
	default:
		return nil, fmt.Errorf("invalid license type: %s", licenseType)
	}

	// Get the current user to check their license type
	scimUser, err := o.client.GetScimUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	// If granting view-only to an enterprise user, do nothing (don't downgrade)
	if licenseType == licenseTypeViewOnly && scimUser.UserType == licenseTypeEnterprise {
		// User already has a higher license type, no need to downgrade
		return nil, nil
	}

	// If user already has the exact license type being granted, return GrantAlreadyExists annotation
	if scimUser.UserType == licenseType {
		return annotations.New(&v2.GrantAlreadyExists{}), nil
	}

	// Update user's license type
	_, err = o.client.UpdateScimUserLicense(ctx, userID, licenseType)
	if err != nil {
		return nil, fmt.Errorf("failed to grant license: %w", err)
	}

	return nil, nil
}

// Revoke implements connectorbuilder.ResourceRevoker interface.
func (o *licenseResourceType) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	// Validate that principal is a user
	if grant.Principal.Id.ResourceType != resourceTypeUser.Id {
		return nil, fmt.Errorf("principal is not a user")
	}

	// Validate that entitlement is for a license
	if grant.Entitlement.Resource.Id.ResourceType != resourceTypeLicense.Id {
		return nil, fmt.Errorf("entitlement is not for a license")
	}

	// Get user ID from principal resource
	userID := grant.Principal.Id.Resource

	// Get license type from entitlement resource
	licenseType := grant.Entitlement.Resource.Id.Resource

	// Get the current user to check their license type
	scimUser, err := o.client.GetScimUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	// If the user doesn't have this license type, return GrantAlreadyRevoked
	if scimUser.UserType != licenseType {
		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	// Special behavior based on license type
	switch licenseType {
	case licenseTypeEnterprise:
		// Downgrade to view-only license
		_, err := o.client.UpdateScimUserLicense(ctx, userID, licenseTypeViewOnly)
		if err != nil {
			return nil, fmt.Errorf("failed to downgrade enterprise license to view-only: %w", err)
		}
	case licenseTypeViewOnly:
		// Deprovision user (set active=false)
		_, err := o.client.DeactivateScimUser(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to deprovision user: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid license type: %s", licenseType)
	}

	return nil, nil
}

func licenseBuilder(client *asana.Client) *licenseResourceType {
	return &licenseResourceType{
		resourceType: resourceTypeLicense,
		client:       client,
	}
}
