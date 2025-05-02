package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-asana/pkg/asana"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type enterpriseUserResourceType struct {
	resourceType            *v2.ResourceType
	client                  *asana.Client
	accountCreationSettings AccountCreationSettings
}

// Create a new connector resource for an Asana enterprise user from SCIM response.
func enterpriseUserResource(ctx context.Context, user *asana.ScimUser, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	var email string
	for _, e := range user.Emails {
		if e.Primary {
			email = e.Value
			break
		}
	}
	if email == "" && len(user.Emails) > 0 {
		email = user.Emails[0].Value
	}

	profile := map[string]interface{}{
		"login":     user.UserName,
		"user_id":   user.ID,
		"title":     user.Title,
		"active":    user.Active,
		"user_type": user.UserType,
	}

	// Add name fields if available
	if user.Name.Formatted != "" {
		profile["name"] = user.Name.Formatted
	}
	if user.Name.GivenName != "" {
		profile["first_name"] = user.Name.GivenName
	}
	if user.Name.FamilyName != "" {
		profile["last_name"] = user.Name.FamilyName
	}

	// Add enterprise fields if available
	ext := user.EnterpriseExtension
	if ext.Department != "" {
		profile["department"] = ext.Department
	}
	if ext.Organization != "" {
		profile["organization"] = ext.Organization
	}
	if ext.CostCenter != "" {
		profile["cost_center"] = ext.CostCenter
	}
	if ext.Division != "" {
		profile["division"] = ext.Division
	}
	if ext.EmployeeNumber != "" {
		profile["employee_number"] = ext.EmployeeNumber
	}
	if ext.Manager != nil && ext.Manager.Value != "" {
		profile["manager_id"] = ext.Manager.Value
	}

	// Add address fields if available
	if len(user.Addresses) > 0 {
		var addr asana.ScimAddress
		for _, a := range user.Addresses {
			if a.Primary {
				addr = a
				break
			}
		}
		if addr == (asana.ScimAddress{}) {
			addr = user.Addresses[0]
		}

		if addr.Country != "" {
			profile["country"] = addr.Country
		}
		if addr.Region != "" {
			profile["region"] = addr.Region
		}
		if addr.Locality != "" {
			profile["locality"] = addr.Locality
		}
	}

	// Add phone number if available
	if len(user.PhoneNumbers) > 0 {
		var phone asana.ScimPhoneNumber
		for _, p := range user.PhoneNumbers {
			if p.Primary {
				phone = p
				break
			}
		}
		if phone == (asana.ScimPhoneNumber{}) {
			phone = user.PhoneNumbers[0]
		}

		if phone.Value != "" {
			profile["phone"] = phone.Value
		}
	}

	// Add language if available
	if user.PreferredLanguage != "" {
		profile["preferred_language"] = user.PreferredLanguage
	}

	userTraitOptions := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithEmail(email, true),
	}

	// Set user status based on active flag
	if user.Active {
		userTraitOptions = append(userTraitOptions, rs.WithStatus(v2.UserTrait_Status_STATUS_ENABLED))
	} else {
		userTraitOptions = append(userTraitOptions, rs.WithStatus(v2.UserTrait_Status_STATUS_DISABLED))
	}

	displayName := user.Name.Formatted
	if displayName == "" {
		displayName = user.UserName
	}

	ret, err := rs.NewUserResource(
		displayName,
		resourceTypeUser,
		user.ID,
		userTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *enterpriseUserResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func (o *enterpriseUserResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}

	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	// Get the start index from the page token
	startIdx := 1 // SCIM API uses 1-based indexing
	if bag.PageToken() != "" {
		idx, err := strconv.Atoi(bag.PageToken())
		if err == nil && idx > 0 {
			startIdx = idx
		}
	}

	// Get users via SCIM API
	usersResp, err := o.client.GetScimUsers(ctx, ResourcesPageSize, startIdx, "")
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, user := range usersResp.Resources {
		userCopy := user
		ur, err := enterpriseUserResource(ctx, &userCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}

	// Calculate next page token
	nextToken := ""
	if len(usersResp.Resources) > 0 {
		nextStartIdx := startIdx + len(usersResp.Resources)
		if nextStartIdx <= usersResp.TotalResults {
			nextToken = strconv.Itoa(nextStartIdx)
		}
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	return rv, pageToken, nil, nil
}

func (o *enterpriseUserResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *enterpriseUserResourceType) Grants(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	// Enterprise users can have license grants
	userTrait, err := rs.GetUserTrait(resource)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to get user trait: %w", err)
	}

	profile := userTrait.Profile.AsMap()
	userType, ok := profile["user_type"].(string)
	if !ok || userType == "" {
		// No user type means no license
		return nil, "", nil, nil
	}

	var rv []*v2.Grant

	// Create license resource IDs
	var licenseType string
	switch userType {
	case "enterprise":
		licenseType = licenseTypeEnterprise
	case "view only":
		licenseType = licenseTypeViewOnly
	default:
		// Unknown license type
		return nil, "", nil, nil
	}

	licenseResourceID := &v2.ResourceId{
		ResourceType: resourceTypeLicense.Id,
		Resource:     licenseType,
	}

	// Create grant from license to user
	userResID := &v2.ResourceId{
		ResourceType: resourceTypeUser.Id,
		Resource:     resource.Id.Resource,
	}

	grant := &v2.Grant{
		Principal: &v2.Resource{
			Id: userResID,
		},
		Entitlement: &v2.Entitlement{
			Id: licenseType,
			Resource: &v2.Resource{
				Id: licenseResourceID,
			},
		},
		Id: fmt.Sprintf("%s:%s", licenseType, resource.Id.Resource),
	}

	rv = append(rv, grant)

	return rv, "", nil, nil
}

// CreateAccountCapabilityDetails describes the account creation capabilities this connector supports.
func (o *enterpriseUserResourceType) CreateAccountCapabilityDetails(ctx context.Context) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	// For Asana, we don't need to handle passwords as Asana handles that through email invites
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
	}, nil, nil
}

// CreateAccount provisions a new user account in Asana using the SCIM API.
func (o *enterpriseUserResourceType) CreateAccount(
	ctx context.Context,
	accountInfo *v2.AccountInfo,
	credentialOptions *v2.CredentialOptions,
) (
	connectorbuilder.CreateAccountResponse,
	[]*v2.PlaintextData,
	annotations.Annotations,
	error,
) {
	l := ctxzap.Extract(ctx)

	// Get account information from the profile
	profile := accountInfo.Profile.AsMap()

	// Extract email
	email, ok := profile["email"].(string)
	if !ok || email == "" {
		return nil, nil, nil, fmt.Errorf("email is required")
	}

	// Check for workspace ID in the request, otherwise use the default
	workspaceID := o.accountCreationSettings.DefaultWorkspaceID
	if requestWorkspaceID, ok := profile["workspace_id"].(string); ok && requestWorkspaceID != "" {
		workspaceID = requestWorkspaceID
	}

	// Validate that we have a workspace ID
	if workspaceID == "" {
		return nil, nil, nil, fmt.Errorf("workspace ID not provided and default workspace ID not set")
	}

	l.Info("baton-asana: creating user account via SCIM API",
		zap.String("email", email),
		zap.String("workspace_id", workspaceID))

	// Prepare SCIM user data
	scimUser := &asana.ScimUser{
		Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		UserName: email,
		Emails: []asana.ScimEmail{
			{
				Value:   email,
				Primary: true,
				Type:    "work",
			},
		},
		Active: true,
	}

	// Add name if provided
	if firstName, ok := profile["first_name"].(string); ok && firstName != "" {
		scimUser.Name.GivenName = firstName
	}

	if lastName, ok := profile["last_name"].(string); ok && lastName != "" {
		scimUser.Name.FamilyName = lastName
	}

	if scimUser.Name.GivenName != "" || scimUser.Name.FamilyName != "" {
		switch {
		case scimUser.Name.GivenName != "" && scimUser.Name.FamilyName != "":
			scimUser.Name.Formatted = scimUser.Name.GivenName + " " + scimUser.Name.FamilyName
		case scimUser.Name.GivenName != "":
			scimUser.Name.Formatted = scimUser.Name.GivenName
		default:
			scimUser.Name.Formatted = scimUser.Name.FamilyName
		}
	}

	// Handle license type if provided
	if licenseType, ok := profile["license_type"].(string); ok && licenseType != "" {
		scimUser.UserType = licenseType
	}

	// Create the user via SCIM API
	createdUser, err := o.client.CreateScimUser(ctx, scimUser)
	if err != nil {
		l.Error("baton-asana: failed to create user via SCIM API",
			zap.String("email", email),
			zap.Error(err))
		return nil, nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Log success with user ID
	l.Info("baton-asana: user created successfully via SCIM API",
		zap.String("email", email),
		zap.String("user_id", createdUser.ID))

	// Create the user resource
	userRes, err := enterpriseUserResource(ctx, createdUser, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create user resource: %w", err)
	}

	// Return successful creation
	successResult := &v2.CreateAccountResponse_SuccessResult{
		Resource: userRes,
	}

	l.Info("baton-asana: successfully created user account",
		zap.String("email", email),
		zap.String("user_id", createdUser.ID))

	// Asana doesn't require password handling as it sends email invites
	return successResult, []*v2.PlaintextData{}, nil, nil
}

func enterpriseUserBuilder(client *asana.Client, accountCreationSettings AccountCreationSettings) *enterpriseUserResourceType {
	return &enterpriseUserResourceType{
		resourceType:            resourceTypeUser,
		client:                  client,
		accountCreationSettings: accountCreationSettings,
	}
}
