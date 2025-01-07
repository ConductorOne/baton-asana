package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"net/http"
	"net/url"
	"strconv"
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
	usersUrl := fmt.Sprint(BaseUrl, "/users")
	q := url.Values{}
	q.Add("workspace", getUsersVars.WorkspaceId)
	q.Add("opt_fields", "email,name")
	q = paginationQuery(q, getUsersVars.Limit, getUsersVars.Offset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usersUrl, nil)
	if err != nil {
		return nil, "", nil, err
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", nil, err
	}
	defer resp.Body.Close()

	var res UsersResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, "", nil, err
	}

	if (res.NextPage != PaginationData{}) {
		return res.Data, res.NextPage.Offset, resp, nil
	}

	return res.Data, "", resp, nil
}

// GetWorkspace returns details of a single workspace.
func (c *Client) GetWorkspace(ctx context.Context, workspaceId string) (Workspace, *http.Response, error) {
	workspaceUrl := fmt.Sprintf("%s/workspaces/%s", BaseUrl, workspaceId)
	q := url.Values{}
	q.Add("opt_fields", "is_organization,name,email_domains")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, workspaceUrl, nil)
	if err != nil {
		return Workspace{}, nil, err
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Workspace{}, nil, err
	}
	defer resp.Body.Close()

	var res WorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return Workspace{}, nil, err
	}

	return res.Data, resp, nil
}

// GetWorkspaceMemberships returns all workspace memberships for a single workspace.
func (c *Client) GetWorkspaceMemberships(ctx context.Context, getWorkspaceMembershipsVars GetWorkspaceMembershipsVars) ([]WorkspaceMembership, string, *http.Response, error) {
	membershipsUrl := fmt.Sprintf("%s/workspaces/%s/workspace_memberships", BaseUrl, getWorkspaceMembershipsVars.WorkspaceId)
	q := url.Values{}
	q.Add("opt_fields", "name,is_active,is_admin,is_guest,workspace.name,user.name,user.email")
	q = paginationQuery(q, getWorkspaceMembershipsVars.Limit, getWorkspaceMembershipsVars.Offset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, membershipsUrl, nil)
	if err != nil {
		return nil, "", nil, err
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", nil, err
	}
	defer resp.Body.Close()

	var res WorkspaceMembershipsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, "", nil, err
	}

	if (res.NextPage != PaginationData{}) {
		return res.Data, res.NextPage.Offset, resp, nil
	}

	return res.Data, "", resp, nil
}

// GetTeams returns all teams for a single workspace.
func (c *Client) GetTeams(ctx context.Context, getTeamsVars GetTeamsVars) ([]Team, string, *http.Response, error) {
	teamsUrl := fmt.Sprintf("%s/workspaces/%s/teams", BaseUrl, getTeamsVars.WorkspaceId)
	q := url.Values{}
	q.Add("opt_fields", "name,organization.name,organization.id,user.name,user.email")
	q = paginationQuery(q, getTeamsVars.Limit, getTeamsVars.Offset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, teamsUrl, nil)
	if err != nil {
		return nil, "", nil, err
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", nil, err
	}
	defer resp.Body.Close()

	var res TeamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, "", nil, err
	}

	if (res.NextPage != PaginationData{}) {
		return res.Data, res.NextPage.Offset, resp, nil
	}

	return res.Data, "", resp, nil
}

// GetTeamMemberships returns all team memberships for a single team.
func (c *Client) GetTeamMemberships(ctx context.Context, getTeamMembershipsVars GetTeamMembershipsVars) ([]TeamMembership, string, *http.Response, error) {
	teamMembershipsUrl := fmt.Sprintf("%s/teams/%s/team_memberships", BaseUrl, getTeamMembershipsVars.TeamId)
	q := url.Values{}
	q.Add("opt_fields", "team.name,is_limited_access,is_admin,is_guest,user.name,user.email")
	q = paginationQuery(q, getTeamMembershipsVars.Limit, getTeamMembershipsVars.Offset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, teamMembershipsUrl, nil)
	if err != nil {
		return nil, "", nil, err
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", nil, err
	}
	defer resp.Body.Close()

	var res TeamMembershipsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, "", nil, err
	}

	if (res.NextPage != PaginationData{}) {
		return res.Data, res.NextPage.Offset, resp, nil
	}

	return res.Data, "", resp, nil
}

// AuthCheck returns workspace permissions of an authenticated user.
func (c *Client) AuthCheck(ctx context.Context) ([]WorkspaceMembership, error) {
	authUrl := fmt.Sprint(BaseUrl, "/users/me/workspace_memberships")
	q := url.Values{}
	q.Add("opt_fields", "workspace.name,workspace.gid,is_active,is_admin,is_guest")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authUrl, nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res AuthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// AddUserToTeam adds a user to a team.
func (c *Client) AddUserToTeam(ctx context.Context, workspaceId, userId string) error {
	addUserToTeamUrl, err := getPath(BaseUrl, fmt.Sprintf("/teams/%s/addUser", workspaceId))
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// RemoveUserToTeam removes a user to a team.
func (c *Client) RemoveUserToTeam(ctx context.Context, workspaceId, userId string) error {
	removesUserToTeamUrl, err := getPath(BaseUrl, fmt.Sprintf("/teams/%s/removeUser", workspaceId))
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
