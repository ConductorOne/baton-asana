package asana

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/stretchr/testify/require"
)

// newTestClient serves body from a stub Asana API and points the package-level URLs at it.
// Each test gets its own server, so the uhttp response cache cannot carry a body between them.
func newTestClient(t *testing.T, body string) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	baseUrl, scimBaseUrl := BaseUrl, ScimBaseUrl
	t.Cleanup(func() { BaseUrl, ScimBaseUrl = baseUrl, scimBaseUrl })
	SetBaseUrl(server.URL)

	ctx := context.Background()
	httpClient, err := uhttp.NewClient(ctx)
	require.NoError(t, err)
	baseHttpClient, err := uhttp.NewBaseHttpClientWithContext(ctx, httpClient)
	require.NoError(t, err)

	return NewClient("test-token", baseHttpClient)
}

// closeBody drains the response the client hands back. It is nil whenever the request
// failed, which is every missing-pagination case below.
func closeBody(resp *http.Response) {
	if resp != nil {
		_ = resp.Body.Close()
	}
}

func TestGetUsersReturnsTheNextOffset(t *testing.T) {
	c := newTestClient(t, `{"data":[{"gid":"1","name":"Ada"}],"next_page":{"offset":"eyJ0eXAiOjl9"}}`)

	users, nextOffset, resp, err := c.GetUsers(context.Background(), GetUsersVars{WorkspaceId: "w1", Limit: 1})
	defer closeBody(resp)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "eyJ0eXAiOjl9", nextOffset)
}

// Asana sends next_page: null on the last page. That is the end of the sync, not a failure.
func TestGetUsersNullNextPageEndsPagination(t *testing.T) {
	c := newTestClient(t, `{"data":[{"gid":"1","name":"Ada"}],"next_page":null}`)

	users, nextOffset, resp, err := c.GetUsers(context.Background(), GetUsersVars{WorkspaceId: "w1", Limit: 1})
	defer closeBody(resp)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Empty(t, nextOffset)
}

// The case the opt-in exists for: a 200 with users but no next_page key at all. Without
// WithPaginationData this silently looks like the last page and the sync drops everything
// after it.
func TestGetUsersMissingNextPageFailsTheRequest(t *testing.T) {
	c := newTestClient(t, `{"data":[{"gid":"1","name":"Ada"}]}`)

	_, _, resp, err := c.GetUsers(context.Background(), GetUsersVars{WorkspaceId: "w1", Limit: 1})
	defer closeBody(resp)
	require.Error(t, err)
	require.True(t, errors.Is(err, uhttp.ErrMissingPaginationData), "got %v", err)
}

func TestGetTeamsMissingNextPageFailsTheRequest(t *testing.T) {
	c := newTestClient(t, `{"data":[{"gid":"1","name":"Platform"}]}`)

	_, _, resp, err := c.GetTeams(context.Background(), GetTeamsVars{WorkspaceId: "w1", Limit: 1})
	defer closeBody(resp)
	require.Error(t, err)
	require.True(t, errors.Is(err, uhttp.ErrMissingPaginationData), "got %v", err)
}

func TestGetWorkspaceMembershipsMissingNextPageFailsTheRequest(t *testing.T) {
	c := newTestClient(t, `{"data":[{"gid":"1"}]}`)

	_, _, resp, err := c.GetWorkspaceMemberships(context.Background(), GetWorkspaceMembershipsVars{WorkspaceId: "w1", Limit: 1})
	defer closeBody(resp)
	require.Error(t, err)
	require.True(t, errors.Is(err, uhttp.ErrMissingPaginationData), "got %v", err)
}

func TestGetTeamMembershipsMissingNextPageFailsTheRequest(t *testing.T) {
	c := newTestClient(t, `{"data":[{"gid":"1"}]}`)

	_, _, resp, err := c.GetTeamMemberships(context.Background(), GetTeamMembershipsVars{TeamId: "t1", Limit: 1})
	defer closeBody(resp)
	require.Error(t, err)
	require.True(t, errors.Is(err, uhttp.ErrMissingPaginationData), "got %v", err)
}

// GetWorkspace is not paginated, so it keeps decoding with WithJSONResponse and must not
// start demanding a next_page key.
func TestGetWorkspaceDoesNotRequirePaginationData(t *testing.T) {
	c := newTestClient(t, `{"data":{"gid":"w1","name":"Acme","is_organization":true}}`)

	workspace, resp, err := c.GetWorkspace(context.Background(), "w1")
	defer closeBody(resp)
	require.NoError(t, err)
	require.Equal(t, "w1", workspace.Gid)
}
