package asana

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

const BaseUrl = "https://app.asana.com/api/1.0"

type Client struct {
	httpClient  *uhttp.BaseHttpClient
	accessToken string
}

type UsersResponse struct {
	Data     []User         `json:"data"`
	NextPage PaginationData `json:"next_page"`
}

type WorkspaceResponse struct {
	Data Workspace `json:"data"`
}

type AuthCheckResponse struct {
	Data []WorkspaceMembership `json:"data"`
}

type WorkspaceMembershipsResponse struct {
	Data     []WorkspaceMembership `json:"data"`
	NextPage PaginationData        `json:"next_page"`
}

type TeamMembershipsResponse struct {
	Data     []TeamMembership `json:"data"`
	NextPage PaginationData   `json:"next_page"`
}

type GetUsersVars struct {
	Limit       int    `json:"limit"`
	Offset      string `json:"offset"`
	WorkspaceId string
}

type GetWorkspaceMembershipsVars struct {
	Limit       int    `json:"limit"`
	Offset      string `json:"offset"`
	WorkspaceId string
}

type GetTeamMembershipsVars struct {
	Limit  int    `json:"limit"`
	Offset string `json:"offset"`
	TeamId string
}

type GetTeamsVars struct {
	Limit       int    `json:"limit"`
	Offset      string `json:"offset"`
	WorkspaceId string
}

type TeamsResponse struct {
	Data     []Team         `json:"data"`
	NextPage PaginationData `json:"next_page"`
}

// CreateUserResponse is the response from the Asana API when creating a user.
type CreateUserResponse struct {
	Data User `json:"data"`
}

func NewClient(accessToken string, httpClient *uhttp.BaseHttpClient) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient:  httpClient,
	}
}

// returns query params with pagination options.
func paginationQuery(q url.Values, limit int, offset string) url.Values {
	q.Add("limit", strconv.Itoa(limit))
	if offset != "" {
		q.Add("offset", offset)
	}
	return q
}

// GetUsers returns all users for a single workspace.
func (c *Client) GetUsers(ctx context.Context, getUsersVars GetUsersVars) ([]User, string, *http.Response, error) {
	usersUrl, err := getPath(BaseUrl, "/users")
	if err != nil {
		return nil, "", nil, err
	}

	q := url.Values{}
	q.Add("workspace", getUsersVars.WorkspaceId)
	q.Add("opt_fields", "email,name")
	q = paginationQuery(q, getUsersVars.Limit, getUsersVars.Offset)

	usersUrl.RawQuery = q.Encode()

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		usersUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, "", nil, err
	}

	var res UsersResponse
	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&res),
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return nil, "", nil, FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	if (res.NextPage != PaginationData{}) {
		return res.Data, res.NextPage.Offset, resp, nil
	}

	return res.Data, "", resp, nil
}

// GetWorkspace returns details of a single workspace.
func (c *Client) GetWorkspace(ctx context.Context, workspaceId string) (Workspace, *http.Response, error) {
	workspaceUrl, err := getPath(BaseUrl, fmt.Sprintf("/workspaces/%s", workspaceId))
	if err != nil {
		return Workspace{}, nil, err
	}
	q := url.Values{}
	q.Add("opt_fields", "is_organization,name,email_domains")
	workspaceUrl.RawQuery = q.Encode()

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		workspaceUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return Workspace{}, nil, err
	}

	var res WorkspaceResponse
	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&res),
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return Workspace{}, nil, FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	return res.Data, resp, nil
}

// GetWorkspaceMemberships returns all workspace memberships for a single workspace.
func (c *Client) GetWorkspaceMemberships(ctx context.Context, getWorkspaceMembershipsVars GetWorkspaceMembershipsVars) ([]WorkspaceMembership, string, *http.Response, error) {
	membershipsUrl, err := getPath(BaseUrl, fmt.Sprintf("/workspaces/%s/workspace_memberships", getWorkspaceMembershipsVars.WorkspaceId))
	if err != nil {
		return nil, "", nil, err
	}
	q := url.Values{}
	q.Add("opt_fields", "name,is_active,is_admin,is_guest,workspace.name,user.name,user.email")
	q = paginationQuery(q, getWorkspaceMembershipsVars.Limit, getWorkspaceMembershipsVars.Offset)
	membershipsUrl.RawQuery = q.Encode()

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		membershipsUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, "", nil, err
	}

	var res WorkspaceMembershipsResponse
	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&res),
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return nil, "", nil, FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	if (res.NextPage != PaginationData{}) {
		return res.Data, res.NextPage.Offset, resp, nil
	}

	return res.Data, "", resp, nil
}

// GetTeams returns all teams for a single workspace.
func (c *Client) GetTeams(ctx context.Context, getTeamsVars GetTeamsVars) ([]Team, string, *http.Response, error) {
	teamsUrl, err := getPath(BaseUrl, fmt.Sprintf("/workspaces/%s/teams", getTeamsVars.WorkspaceId))
	if err != nil {
		return nil, "", nil, err
	}
	q := url.Values{}
	q.Add("opt_fields", "name,organization.name,organization.id,user.name,user.email")
	q = paginationQuery(q, getTeamsVars.Limit, getTeamsVars.Offset)
	teamsUrl.RawQuery = q.Encode()

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		teamsUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, "", nil, err
	}

	var res TeamsResponse
	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&res),
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return nil, "", nil, FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	if (res.NextPage != PaginationData{}) {
		return res.Data, res.NextPage.Offset, resp, nil
	}

	return res.Data, "", resp, nil
}

// GetTeamMemberships returns all team memberships for a single team.
func (c *Client) GetTeamMemberships(ctx context.Context, getTeamMembershipsVars GetTeamMembershipsVars) ([]TeamMembership, string, *http.Response, error) {
	teamMembershipsUrl, err := getPath(BaseUrl, fmt.Sprintf("/teams/%s/team_memberships", getTeamMembershipsVars.TeamId))
	if err != nil {
		return nil, "", nil, err
	}
	q := url.Values{}
	q.Add("opt_fields", "team.name,is_limited_access,is_admin,is_guest,user.name,user.email")
	q = paginationQuery(q, getTeamMembershipsVars.Limit, getTeamMembershipsVars.Offset)
	teamMembershipsUrl.RawQuery = q.Encode()

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		teamMembershipsUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, "", nil, err
	}

	var res TeamMembershipsResponse
	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&res),
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return nil, "", nil, FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	if (res.NextPage != PaginationData{}) {
		return res.Data, res.NextPage.Offset, resp, nil
	}

	return res.Data, "", resp, nil
}

// AuthCheck returns workspace permissions of an authenticated user.
func (c *Client) AuthCheck(ctx context.Context) ([]WorkspaceMembership, error) {
	authUrl, err := getPath(BaseUrl, "/users/me/workspace_memberships")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Add("opt_fields", "workspace.name,workspace.gid,is_active,is_admin,is_guest")
	authUrl.RawQuery = q.Encode()

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		authUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var res AuthCheckResponse
	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&res),
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return nil, FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	return res.Data, nil
}

// ListAllWorkspaces returns all workspaces visible to the authenticated token.
// This is particularly useful for service account tokens which may not have direct workspace memberships.
func (c *Client) ListAllWorkspaces(ctx context.Context) ([]Workspace, error) {
	workspacesUrl, err := getPath(BaseUrl, "/workspaces")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Add("opt_fields", "gid,name,is_organization,email_domains")
	workspacesUrl.RawQuery = q.Encode()

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		workspacesUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
	)
	if err != nil {
		return nil, err
	}

	var res struct {
		Data []Workspace `json:"data"`
	}
	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&res),
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return nil, FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	return res.Data, nil
}

// AddUserToWorkspace adds a user to a workspace.
func (c *Client) AddUserToWorkspace(ctx context.Context, workspaceId, userId string) error {
	addUserToWorkspaceUrl, err := getPath(BaseUrl, fmt.Sprintf("/workspaces/%s/addUser", workspaceId))
	if err != nil {
		return err
	}

	body := baseMutationBody{
		Data: struct {
			User string `json:"user"`
		}{
			User: userId,
		},
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPost,
		addUserToWorkspaceUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return err
	}

	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	return nil
}

// RemoveUserToWorkspace removes a user from a workspace.
func (c *Client) RemoveUserToWorkspace(ctx context.Context, workspaceId, userId string) error {
	removeUserToWorkspaceUrl, err := getPath(BaseUrl, fmt.Sprintf("/workspaces/%s/removeUser", workspaceId))
	if err != nil {
		return err
	}

	body := baseMutationBody{
		Data: struct {
			User string `json:"user"`
		}{
			User: userId,
		},
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPost,
		removeUserToWorkspaceUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return err
	}

	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	return nil
}

// InviteUserToWorkspace invites a user by email to a workspace.
func (c *Client) InviteUserToWorkspace(ctx context.Context, workspaceId, email string) (*User, error) {
	inviteUserUrl, err := getPath(BaseUrl, fmt.Sprintf("/workspaces/%s/addUser", workspaceId))
	if err != nil {
		return nil, err
	}

	// According to Asana API docs, we need to use the 'user' field
	// instead of 'email' when inviting by email
	body := baseMutationBody{
		Data: struct {
			User string `json:"user"`
		}{
			User: email,
		},
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPost,
		inviteUserUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return nil, err
	}

	var result CreateUserResponse
	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&result),
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return nil, FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	return &result.Data, nil
}

// AddUserToTeam adds a user to a team.
func (c *Client) AddUserToTeam(ctx context.Context, teamId, userId string) error {
	addUserToTeamUrl, err := getPath(BaseUrl, fmt.Sprintf("/teams/%s/addUser", teamId))
	if err != nil {
		return err
	}

	body := baseMutationBody{
		Data: struct {
			User string `json:"user"`
		}{
			User: userId,
		},
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPost,
		addUserToTeamUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return err
	}

	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	return nil
}

// RemoveUserToTeam removes a user to a team.
func (c *Client) RemoveUserToTeam(ctx context.Context, teamId, userId string) error {
	removesUserToTeamUrl, err := getPath(BaseUrl, fmt.Sprintf("/teams/%s/removeUser", teamId))
	if err != nil {
		return err
	}

	body := baseMutationBody{
		Data: struct {
			User string `json:"user"`
		}{
			User: userId,
		},
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPost,
		removesUserToTeamUrl,
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return err
	}

	var asanaError AsanaError
	resp, err := c.httpClient.Do(req,
		uhttp.WithErrorResponse(&asanaError),
	)
	if err != nil {
		return FormatAsanaError(&asanaError, err)
	}
	defer resp.Body.Close()

	return nil
}
