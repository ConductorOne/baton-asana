package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const BaseUrl = "https://app.asana.com/api/1.0"

type Client struct {
	httpClient  *http.Client
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

type PaginationVars struct {
	Limit  int    `json:"limit"`
	Offset string `json:"offset"`
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

func NewClient(accessToken string, httpClient *http.Client) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient:  httpClient,
	}
}

// returns pagination options as a string to pass as query params.
func paginationOptions(paginationVars PaginationVars) string {
	var paginationOptions string
	paginationOptions = fmt.Sprintf("&limit=%d", paginationVars.Limit)
	if paginationVars.Offset != "" {
		paginationOptions = fmt.Sprintf("%s&offset=%s", paginationOptions, paginationVars.Offset)
	}
	return paginationOptions
}

// GetUsers returns all users for a single workspace.
func (c *Client) GetUsers(ctx context.Context, workspaceId string, paginationVars PaginationVars) ([]User, string, *http.Response, error) {
	paginationOptions := paginationOptions(paginationVars)
	url := fmt.Sprint(
		BaseUrl,
		"/users?workspace=",
		workspaceId,
		"&opt_fields=email,name",
		paginationOptions,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", nil, err
	}

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
	url := fmt.Sprint(
		BaseUrl,
		"/workspaces/",
		workspaceId,
		"?opt_fields=is_organization,name,email_domains",
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Workspace{}, nil, err
	}

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
	paginationOptions := paginationOptions(PaginationVars{Offset: getWorkspaceMembershipsVars.Offset, Limit: getWorkspaceMembershipsVars.Limit})
	url := fmt.Sprint(
		BaseUrl,
		"/workspaces/",
		getWorkspaceMembershipsVars.WorkspaceId,
		"/workspace_memberships?opt_fields=name,is_active,is_admin,is_guest,workspace.name,user.name,user.email",
		paginationOptions,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", nil, err
	}

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
	paginationOptions := paginationOptions(PaginationVars{Offset: getTeamsVars.Offset, Limit: getTeamsVars.Limit})
	url := fmt.Sprint(
		BaseUrl,
		"/workspaces/",
		getTeamsVars.WorkspaceId,
		"/teams?opt_fields=name,organization.name,organization.id,user.name,user.email",
		paginationOptions,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", nil, err
	}

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
	paginationOptions := paginationOptions(PaginationVars{Offset: getTeamMembershipsVars.Offset, Limit: getTeamMembershipsVars.Limit})
	url := fmt.Sprint(
		BaseUrl,
		"/teams/",
		getTeamMembershipsVars.TeamId,
		"/team_memberships?opt_fields=team.name,is_limited_access,is_admin,is_guest,user.name,user.email",
		paginationOptions,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", nil, err
	}

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
	url := fmt.Sprint(
		BaseUrl,
		"/users/me",
		"/workspace_memberships?opt_fields=workspace.name,workspace.gid,is_active,is_admin,is_guest",
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

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
