package asana

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

const (
	scimUserSchema           = "urn:ietf:params:scim:schemas:core:2.0:User"
	scimEnterpriseUserSchema = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	scimGroupSchema          = "urn:ietf:params:scim:schemas:core:2.0:Group"
	scimPatchSchema          = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
)

// GetScimUsers gets a list of users via the SCIM API.
func (c *Client) GetScimUsers(ctx context.Context, count int, startIndex int, filter string) (*ScimListResponse, error) {
	usersUrl, err := getPath(ScimBaseUrl, "/Users")
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	if count > 0 {
		q.Add("count", strconv.Itoa(count))
	}
	if startIndex > 0 {
		q.Add("startIndex", strconv.Itoa(startIndex))
	}
	if filter != "" {
		q.Add("filter", filter)
	}
	usersUrl.RawQuery = q.Encode()

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		usersUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimListResponse
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// GetScimUser gets a specific user by ID via the SCIM API.
func (c *Client) GetScimUser(ctx context.Context, userID string) (*ScimUser, error) {
	userUrl, err := getPath(ScimBaseUrl, fmt.Sprintf("/Users/%s", userID))
	if err != nil {
		return nil, err
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		userUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimUser
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// CreateScimUser creates a new user via the SCIM API.
func (c *Client) CreateScimUser(ctx context.Context, user *ScimUser) (*ScimUser, error) {
	userUrl, err := getPath(ScimBaseUrl, "/Users")
	if err != nil {
		return nil, err
	}

	// Ensure schemas are set
	if user.Schemas == nil {
		user.Schemas = []string{scimUserSchema}
	}

	// If there's enterprise extension data, add that schema
	if user.EnterpriseExtension != (ScimEnterpriseExtension{}) {
		hasEnterpriseSchema := false
		for _, schema := range user.Schemas {
			if schema == scimEnterpriseUserSchema {
				hasEnterpriseSchema = true
				break
			}
		}
		if !hasEnterpriseSchema {
			user.Schemas = append(user.Schemas, scimEnterpriseUserSchema)
		}
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPost,
		userUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(user),
		uhttp.WithContentTypeJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimUser
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// UpdateScimUser updates a user via the SCIM API.
func (c *Client) UpdateScimUser(ctx context.Context, userID string, user *ScimUser) (*ScimUser, error) {
	userUrl, err := getPath(ScimBaseUrl, fmt.Sprintf("/Users/%s", userID))
	if err != nil {
		return nil, err
	}

	// Ensure schemas are set
	if user.Schemas == nil {
		user.Schemas = []string{scimUserSchema}
	}

	// If there's enterprise extension data, add that schema
	if user.EnterpriseExtension != (ScimEnterpriseExtension{}) {
		hasEnterpriseSchema := false
		for _, schema := range user.Schemas {
			if schema == scimEnterpriseUserSchema {
				hasEnterpriseSchema = true
				break
			}
		}
		if !hasEnterpriseSchema {
			user.Schemas = append(user.Schemas, scimEnterpriseUserSchema)
		}
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPut,
		userUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(user),
		uhttp.WithContentTypeJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimUser
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// PatchScimUser patches a user via the SCIM API.
func (c *Client) PatchScimUser(ctx context.Context, userID string, operations []ScimPatchOperation) (*ScimUser, error) {
	userUrl, err := getPath(ScimBaseUrl, fmt.Sprintf("/Users/%s", userID))
	if err != nil {
		return nil, err
	}

	patchRequest := ScimPatch{
		Schemas:    []string{scimPatchSchema},
		Operations: operations,
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPatch,
		userUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(patchRequest),
		uhttp.WithContentTypeJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimUser
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// DeleteScimUser deletes (deactivates) a user via the SCIM API.
func (c *Client) DeleteScimUser(ctx context.Context, userID string) error {
	userUrl, err := getPath(ScimBaseUrl, fmt.Sprintf("/Users/%s", userID))
	if err != nil {
		return err
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodDelete,
		userUrl,
		uhttp.WithBearerToken(c.accessToken),
	)
	if err != nil {
		return err
	}

	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetScimUserByUsername gets a user by username (email) via the SCIM API.
func (c *Client) GetScimUserByUsername(ctx context.Context, username string) (*ScimUser, error) {
	filter := fmt.Sprintf("userName eq \"%s\"", username)
	users, err := c.GetScimUsers(ctx, 1, 0, filter)
	if err != nil {
		return nil, err
	}

	if users.TotalResults == 0 || len(users.Resources) == 0 {
		return nil, fmt.Errorf("user with username %s not found", username)
	}

	return &users.Resources[0], nil
}

// DeactivateScimUser deactivates a user via the SCIM API.
func (c *Client) DeactivateScimUser(ctx context.Context, userID string) (*ScimUser, error) {
	operations := []ScimPatchOperation{
		{
			Op:    "replace",
			Path:  "active",
			Value: false,
		},
	}

	return c.PatchScimUser(ctx, userID, operations)
}

// ActivateScimUser activates a user via the SCIM API.
func (c *Client) ActivateScimUser(ctx context.Context, userID string) (*ScimUser, error) {
	operations := []ScimPatchOperation{
		{
			Op:    "replace",
			Path:  "active",
			Value: true,
		},
	}

	return c.PatchScimUser(ctx, userID, operations)
}

// UpdateScimUserLicense updates a user's license type via the SCIM API.
func (c *Client) UpdateScimUserLicense(ctx context.Context, userID string, licenseType string) (*ScimUser, error) {
	operations := []ScimPatchOperation{
		{
			Op:    "replace",
			Path:  "userType",
			Value: licenseType,
		},
	}

	return c.PatchScimUser(ctx, userID, operations)
}

// GetScimTeams gets a list of teams via the SCIM API.
func (c *Client) GetScimTeams(ctx context.Context, count int, startIndex int, filter string) (*ScimTeamListResponse, error) {
	teamsUrl, err := getPath(ScimBaseUrl, "/Groups")
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	if count > 0 {
		q.Add("count", strconv.Itoa(count))
	}
	if startIndex > 0 {
		q.Add("startIndex", strconv.Itoa(startIndex))
	}
	if filter != "" {
		q.Add("filter", filter)
	}
	teamsUrl.RawQuery = q.Encode()

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		teamsUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimTeamListResponse
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// GetScimTeam gets a specific team by ID via the SCIM API.
func (c *Client) GetScimTeam(ctx context.Context, teamID string) (*ScimTeam, error) {
	teamUrl, err := getPath(ScimBaseUrl, fmt.Sprintf("/Groups/%s", teamID))
	if err != nil {
		return nil, err
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		teamUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimTeam
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// CreateScimTeam creates a new team via the SCIM API.
func (c *Client) CreateScimTeam(ctx context.Context, team *ScimTeam) (*ScimTeam, error) {
	teamUrl, err := getPath(ScimBaseUrl, "/Groups")
	if err != nil {
		return nil, err
	}

	// Ensure schemas are set
	if team.Schemas == nil {
		team.Schemas = []string{scimGroupSchema}
	}

	// Set meta if not present
	if team.Meta.ResourceType == "" {
		team.Meta.ResourceType = "Group"
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPost,
		teamUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(team),
		uhttp.WithContentTypeJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimTeam
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// UpdateScimTeam updates a team via the SCIM API.
func (c *Client) UpdateScimTeam(ctx context.Context, teamID string, team *ScimTeam) (*ScimTeam, error) {
	teamUrl, err := getPath(ScimBaseUrl, fmt.Sprintf("/Groups/%s", teamID))
	if err != nil {
		return nil, err
	}

	// Ensure schemas are set
	if team.Schemas == nil {
		team.Schemas = []string{scimGroupSchema}
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPut,
		teamUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(team),
		uhttp.WithContentTypeJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimTeam
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// PatchScimTeam patches a team via the SCIM API.
func (c *Client) PatchScimTeam(ctx context.Context, teamID string, operations []ScimPatchOperation) (*ScimTeam, error) {
	teamUrl, err := getPath(ScimBaseUrl, fmt.Sprintf("/Groups/%s", teamID))
	if err != nil {
		return nil, err
	}

	patchRequest := ScimPatch{
		Schemas:    []string{scimPatchSchema},
		Operations: operations,
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPatch,
		teamUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(patchRequest),
		uhttp.WithContentTypeJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var result ScimTeam
	var scimError ScimError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&scimError),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &result, nil
}

// GetScimTeamByDisplayName gets a team by displayName via the SCIM API.
func (c *Client) GetScimTeamByDisplayName(ctx context.Context, displayName string) (*ScimTeam, error) {
	filter := fmt.Sprintf("displayName eq \"%s\"", displayName)
	teams, err := c.GetScimTeams(ctx, 1, 0, filter)
	if err != nil {
		return nil, err
	}

	if teams.TotalResults == 0 || len(teams.Resources) == 0 {
		return nil, fmt.Errorf("team with displayName %s not found", displayName)
	}

	return &teams.Resources[0], nil
}
