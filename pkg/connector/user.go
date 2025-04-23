package connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/conductorone/baton-asana/pkg/asana"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type userResourceType struct {
	resourceType            *v2.ResourceType
	client                  *asana.Client
	accountCreationSettings AccountCreationSettings
}

// AccountCreationSettings holds settings for account creation.
type AccountCreationSettings struct {
	DefaultWorkspaceID string
}

func (o *userResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for an Asana user.
func userResource(ctx context.Context, user *asana.User, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	names := strings.SplitN(user.Name, " ", 2)
	var firstName, lastName string
	switch len(names) {
	case 1:
		firstName = names[0]
	case 2:
		firstName = names[0]
		lastName = names[1]
	}

	profile := map[string]interface{}{
		"first_name": firstName,
		"last_name":  lastName,
		"login":      user.Email,
		"user_id":    user.Gid,
	}

	userTraitOptions := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithEmail(user.Email, true),
		rs.WithStatus(v2.UserTrait_Status_STATUS_ENABLED),
	}

	ret, err := rs.NewUserResource(
		user.Name,
		resourceTypeUser,
		user.Gid,
		userTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *userResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}

	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	users, nextToken, _, err := o.client.GetUsers(ctx, asana.GetUsersVars{WorkspaceId: parentId.Resource, Limit: ResourcesPageSize, Offset: bag.PageToken()})
	if err != nil {
		return nil, "", nil, err
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, user := range users {
		userCopy := user
		ur, err := userResource(ctx, &userCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}

	return rv, pageToken, nil, nil
}

func (o *userResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *userResourceType) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// CreateAccountCapabilityDetails describes the account creation capabilities this connector supports.
func (o *userResourceType) CreateAccountCapabilityDetails(ctx context.Context) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	// For Asana, we don't need to handle passwords as Asana handles that through email invites
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
	}, nil, nil
}

// CreateAccount provisions a new user account in Asana.
func (o *userResourceType) CreateAccount(
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

	// Check if we have a default workspace ID
	if o.accountCreationSettings.DefaultWorkspaceID == "" {
		return nil, nil, nil, fmt.Errorf("default workspace ID not set")
	}

	// Get account information from the profile
	profile := accountInfo.Profile.AsMap()

	// Extract email
	email, ok := profile["email"].(string)
	if !ok || email == "" {
		return nil, nil, nil, fmt.Errorf("email is required")
	}

	// Extract name
	firstName, _ := profile["first_name"].(string)
	lastName, _ := profile["last_name"].(string)
	var fullName string

	switch {
	case firstName != "" && lastName != "":
		fullName = firstName + " " + lastName
	case firstName != "":
		fullName = firstName
	case lastName != "":
		fullName = lastName
	default:
		// If no name is provided, use the part of the email before @
		parts := strings.Split(email, "@")
		fullName = parts[0]
	}

	l.Info("baton-asana: creating user account",
		zap.String("email", email),
		zap.String("name", fullName),
		zap.String("workspace_id", o.accountCreationSettings.DefaultWorkspaceID))

	// Invite the user to Asana via the workspace and get user details directly from the response
	invitedUser, err := o.client.InviteUserToWorkspace(ctx, o.accountCreationSettings.DefaultWorkspaceID, email)
	if err != nil {
		l.Error("baton-asana: failed to invite user to workspace",
			zap.String("email", email),
			zap.String("workspace_id", o.accountCreationSettings.DefaultWorkspaceID),
			zap.Error(err))
		return nil, nil, nil, fmt.Errorf("failed to invite user to workspace: %w", err)
	}

	// Log success with user ID
	l.Info("baton-asana: user invited successfully",
		zap.String("email", email),
		zap.String("user_id", invitedUser.Gid))

	// Create the user resource
	userRes, err := userResource(ctx, invitedUser, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create user resource: %w", err)
	}

	// Return successful creation
	successResult := &v2.CreateAccountResponse_SuccessResult{
		Resource: userRes,
	}

	l.Info("baton-asana: successfully created user account",
		zap.String("email", email),
		zap.String("user_id", invitedUser.Gid))

	// Asana doesn't require password handling as it sends email invites
	return successResult, []*v2.PlaintextData{}, nil, nil
}

func userBuilder(client *asana.Client, accountCreationSettings AccountCreationSettings) *userResourceType {
	return &userResourceType{
		resourceType:            resourceTypeUser,
		client:                  client,
		accountCreationSettings: accountCreationSettings,
	}
}
