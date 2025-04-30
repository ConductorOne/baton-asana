package connector

import (
	"context"

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
		[]rs.RoleTraitOption{
			rs.WithRoleProfile(profile),
		},
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

func (o *licenseResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	// Licenses don't have entitlements of their own
	return nil, "", nil, nil
}

func (o *licenseResourceType) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	// License grants will be emitted by the enterprise user resource type
	return nil, "", nil, nil
}

func licenseBuilder(client *asana.Client) *licenseResourceType {
	return &licenseResourceType{
		resourceType: resourceTypeLicense,
		client:       client,
	}
}
