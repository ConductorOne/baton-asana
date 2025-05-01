package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/conductorone/baton-asana/pkg/asana"
)

const (
	scimUserSchema           = "urn:ietf:params:scim:schemas:core:2.0:User"
	scimEnterpriseUserSchema = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	scimGroupSchema          = "urn:ietf:params:scim:schemas:core:2.0:Group"
	scimPatchSchema          = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	scimListResponseSchema   = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	
	resourceTypeGroup        = "Group"
	membersPath              = "members"
)

// ScimTestServer implements a mock SCIM API server for testing.
type ScimTestServer struct {
	users       map[string]asana.ScimUser
	teams       map[string]asana.ScimTeam
	workspaces  []asana.Workspace
	mu          sync.RWMutex
}

// NewScimTestServer creates a new SCIM test server.
func NewScimTestServer() *ScimTestServer {
	return &ScimTestServer{
		users:  make(map[string]asana.ScimUser),
		teams:  make(map[string]asana.ScimTeam),
		workspaces: []asana.Workspace{
			{
				BaseResource: asana.BaseResource{
					Gid:          "1",
					Name:         "Test Workspace",
					ResourceType: "workspace",
				},
				IsOrganization: true,
				EmailDomains:   []string{"example.com"},
			},
		},
	}
}

// Start starts the SCIM test server.
func (s *ScimTestServer) Start(port int) error {
	mux := http.NewServeMux()

	// Meta information endpoints
	mux.HandleFunc("/api/1.0/scim/ServiceProviderConfig", s.handleServiceProviderConfig)
	mux.HandleFunc("/api/1.0/scim/ResourceTypes", s.handleResourceTypes)
	mux.HandleFunc("/api/1.0/scim/Schemas", s.handleSchemas)

	// Workspaces endpoints for the connector validation
	mux.HandleFunc("/api/1.0/workspaces", s.handleWorkspaces)
	mux.HandleFunc("/api/1.0/workspaces/", s.handleWorkspaceById)
	
	// Teams REST API endpoint
	mux.HandleFunc("/api/1.0/teams/", s.handleTeamRestById)

	// User endpoints
	mux.HandleFunc("/api/1.0/scim/Users", s.handleUsers)
	mux.HandleFunc("/api/1.0/scim/Users/", s.handleUserById)

	// Group endpoints
	mux.HandleFunc("/api/1.0/scim/Groups", s.handleGroups)
	mux.HandleFunc("/api/1.0/scim/Groups/", s.handleGroupById)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Add some test data
	s.addTestData()

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v\n", err)
		}
	}()

	log.Printf("SCIM Test Server started on port %d\n", port)
	return server.ListenAndServe()
}

// Handle non-SCIM Asana API endpoints.
func (s *ScimTestServer) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a list of workspaces
	response := map[string]interface{}{
		"data": s.workspaces,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleWorkspaceById handles GET requests for a specific workspace.
func (s *ScimTestServer) handleWorkspaceById(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := r.URL.Path
	// Check if this is a teams request
	if strings.Contains(path, "/teams") {
		s.handleWorkspaceTeams(w, r)
		return
	}
	
	// Check if this is a workspace_memberships request
	if strings.Contains(path, "/workspace_memberships") {
		s.handleWorkspaceMemberships(w, r)
		return
	}
	
	// Check for add/remove user endpoints
	if strings.Contains(path, "/addUser") {
		s.handleAddUserToWorkspace(w, r)
		return
	}
	
	if strings.Contains(path, "/removeUser") {
		s.handleRemoveUserFromWorkspace(w, r)
		return
	}

	workspaceId := strings.TrimPrefix(path, "/api/1.0/workspaces/")
	if workspaceId == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Workspace ID is required")
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find the matching workspace
	var workspace *asana.Workspace
	for i, ws := range s.workspaces {
		if ws.Gid == workspaceId {
			workspace = &s.workspaces[i]
			break
		}
	}

	if workspace == nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Workspace with ID %s not found", workspaceId))
		return
	}

	// Return the workspace
	response := map[string]interface{}{
		"data": workspace,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleWorkspaceMemberships handles GET requests for workspace memberships.
func (s *ScimTestServer) handleWorkspaceMemberships(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/1.0/workspaces/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Workspace ID is required")
		return
	}

	workspaceId := pathParts[0]

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if workspace exists
	var workspace *asana.Workspace
	for i, ws := range s.workspaces {
		if ws.Gid == workspaceId {
			workspace = &s.workspaces[i]
			break
		}
	}

	if workspace == nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Workspace with ID %s not found", workspaceId))
		return
	}

	// Get query parameters
	limit := 50 // Default limit
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	// We don't use offset for the test implementation, but we parse it for completeness
	_ = r.URL.Query().Get("offset")

	// Create workspace memberships for all users
	var memberships []asana.WorkspaceMembership
	for userId, user := range s.users {
		workspaceMembership := asana.WorkspaceMembership{
			Gid:          fmt.Sprintf("wm-%s-%s", workspaceId, userId),
			ResourceType: "workspace_membership",
			User: asana.User{
				BaseResource: asana.BaseResource{
					Gid:          userId,
					Name:         user.Name.Formatted,
					ResourceType: "user",
				},
				Email: user.UserName,
			},
			Workspace: *workspace,
			IsActive:  true,
			IsAdmin:   false, // Default to regular member
			IsGuest:   false,
		}
		
		// Make the first user an admin for testing
		if userId == "1" {
			workspaceMembership.IsAdmin = true
		}
		
		memberships = append(memberships, workspaceMembership)
	}

	// Pagination
	hasMore := false
	var nextOffset string
	if len(memberships) > limit {
		hasMore = true
		memberships = memberships[:limit]
		nextOffset = fmt.Sprintf("%d", limit)
	}

	// Create pagination data
	var nextPage asana.PaginationData
	if hasMore {
		nextPage.Offset = nextOffset
	}

	response := asana.WorkspaceMembershipsResponse{
		Data:     memberships,
		NextPage: nextPage,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleWorkspaceTeams handles GET requests for teams in a workspace.
func (s *ScimTestServer) handleWorkspaceTeams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/1.0/workspaces/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Workspace ID is required")
		return
	}

	workspaceId := pathParts[0]

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if workspace exists
	var workspaceExists bool
	for _, ws := range s.workspaces {
		if ws.Gid == workspaceId {
			workspaceExists = true
			break
		}
	}

	if !workspaceExists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Workspace with ID %s not found", workspaceId))
		return
	}

	// Get query parameters
	limit := 50 // Default limit
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	// We don't use offset for the test implementation, but we parse it for completeness
	_ = r.URL.Query().Get("offset")

	// Convert ScimTeams to Asana Teams 
	var teams []asana.Team
	for _, scimTeam := range s.teams {
		team := asana.Team{
			BaseResource: asana.BaseResource{
				Gid:          scimTeam.ID,
				Name:         scimTeam.DisplayName,
				ResourceType: "team",
			},
		}
		teams = append(teams, team)
	}

	// Pagination
	hasMore := false
	var nextOffset string
	if len(teams) > limit {
		hasMore = true
		teams = teams[:limit]
		nextOffset = fmt.Sprintf("%d", limit)
	}

	// Create pagination data
	var nextPage asana.PaginationData
	if hasMore {
		nextPage.Offset = nextOffset
	}

	response := asana.TeamsResponse{
		Data:     teams,
		NextPage: nextPage,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleTeamRestById handles requests for a specific team.
func (s *ScimTestServer) handleTeamRestById(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// Check if this is a team_memberships request
	if strings.Contains(path, "/team_memberships") {
		s.handleTeamMemberships(w, r)
		return
	}
	
	// Check for add/remove user endpoints
	if strings.Contains(path, "/addUser") {
		s.handleAddUserToTeam(w, r)
		return
	}
	
	if strings.Contains(path, "/removeUser") {
		s.handleRemoveUserFromTeam(w, r)
		return
	}
	
	// Regular team GET request
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	teamId := strings.TrimPrefix(path, "/api/1.0/teams/")
	if teamId == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Team ID is required")
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find the matching team
	scimTeam, exists := s.teams[teamId]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Team with ID %s not found", teamId))
		return
	}

	// Convert to Asana Team
	team := asana.Team{
		BaseResource: asana.BaseResource{
			Gid:          scimTeam.ID,
			Name:         scimTeam.DisplayName,
			ResourceType: "team",
		},
	}

	// Return the team
	response := map[string]interface{}{
		"data": team,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleAddUserToTeam handles adding a user to a team.
func (s *ScimTestServer) handleAddUserToTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract the team ID from the path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid URL format")
		return
	}
	
	teamId := pathParts[3]

	// Read the request body
	var requestBody struct {
		Data struct {
			User string `json:"user"`
		} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId := requestBody.Data.User

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the team exists
	team, teamExists := s.teams[teamId]
	if !teamExists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Team with ID %s not found", teamId))
		return
	}

	// Check if the user exists
	user, userExists := s.users[userId]
	if !userExists {
		// Check if this is an email address instead of a user ID
		found := false
		for id, u := range s.users {
			if u.UserName == userId {
				userId = id
				user = u
				found = true
				break
			}
		}
		if !found {
			writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("User with ID/email %s not found", userId))
			return
		}
	}

	// Check if user is already a member
	for _, member := range team.Members {
		if member.Value == userId {
			// User is already a member, return success
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	// Add the user to the team
	team.Members = append(team.Members, asana.ScimMember{
		Value:   userId,
		Display: user.Name.Formatted,
	})
	s.teams[teamId] = team

	// Return success
	w.WriteHeader(http.StatusOK)
}

// handleRemoveUserFromTeam handles removing a user from a team.
func (s *ScimTestServer) handleRemoveUserFromTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract the team ID from the path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid URL format")
		return
	}
	
	teamId := pathParts[3]

	// Read the request body
	var requestBody struct {
		Data struct {
			User string `json:"user"`
		} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId := requestBody.Data.User

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the team exists
	team, teamExists := s.teams[teamId]
	if !teamExists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Team with ID %s not found", teamId))
		return
	}

	// Check if this is an email address instead of a user ID
	if !strings.HasPrefix(userId, "1") { // Simple check for user ID format
		for id, u := range s.users {
			if u.UserName == userId {
				userId = id
				break
			}
		}
	}

	// Remove the user from the team
	var newMembers []asana.ScimMember
	memberFound := false
	for _, member := range team.Members {
		if member.Value != userId {
			newMembers = append(newMembers, member)
		} else {
			memberFound = true
		}
	}

	// Only update if we found the member
	if memberFound {
		team.Members = newMembers
		s.teams[teamId] = team
	}

	// Return success
	w.WriteHeader(http.StatusOK)
}

// handleTeamMemberships handles GET requests for team memberships.
func (s *ScimTestServer) handleTeamMemberships(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/1.0/teams/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Team ID is required")
		return
	}

	teamId := pathParts[0]

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if team exists
	scimTeam, exists := s.teams[teamId]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Team with ID %s not found", teamId))
		return
	}

	// Convert ScimTeam to Asana Team
	team := asana.Team{
		BaseResource: asana.BaseResource{
			Gid:          scimTeam.ID,
			Name:         scimTeam.DisplayName,
			ResourceType: "team",
		},
	}

	// Create team memberships
	var memberships []asana.TeamMembership
	for _, member := range scimTeam.Members {
		user, userExists := s.users[member.Value]
		if !userExists {
			continue
		}

		asanaUser := asana.User{
			BaseResource: asana.BaseResource{
				Gid:          user.ID,
				Name:         user.Name.Formatted,
				ResourceType: "user",
			},
			Email: user.UserName,
		}

		membership := asana.TeamMembership{
			Gid:             fmt.Sprintf("tm-%s-%s", teamId, member.Value),
			ResourceType:    "team_membership",
			User:            asanaUser,
			Team:            team,
			IsAdmin:         false, // Default to regular member
			IsGuest:         false,
			IsLimitedAccess: false,
		}
		memberships = append(memberships, membership)
	}

	// Get query parameters
	limit := 50 // Default limit
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	// We don't use offset for the test implementation, but we parse it for completeness
	_ = r.URL.Query().Get("offset")

	// Pagination
	hasMore := false
	var nextOffset string
	if len(memberships) > limit {
		hasMore = true
		memberships = memberships[:limit]
		nextOffset = fmt.Sprintf("%d", limit)
	}

	// Create pagination data
	var nextPage asana.PaginationData
	if hasMore {
		nextPage.Offset = nextOffset
	}

	response := asana.TeamMembershipsResponse{
		Data:     memberships,
		NextPage: nextPage,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleAddUserToWorkspace handles adding a user to a workspace.
func (s *ScimTestServer) handleAddUserToWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract the workspace ID from the path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid URL format")
		return
	}
	
	workspaceId := pathParts[3]

	// Read the request body
	var requestBody struct {
		Data struct {
			User string `json:"user"`
		} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId := requestBody.Data.User

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the workspace exists
	var workspaceExists bool
	for _, ws := range s.workspaces {
		if ws.Gid == workspaceId {
			workspaceExists = true
			break
		}
	}

	if !workspaceExists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Workspace with ID %s not found", workspaceId))
		return
	}

	// If this is an email address, return a user object
	if !strings.HasPrefix(userId, "1") && strings.Contains(userId, "@") {
		// For emails, create a new user
		newUserId := fmt.Sprintf("%d", len(s.users)+1)
		nameParts := strings.Split(strings.Split(userId, "@")[0], ".")
		
		var firstName, lastName string
		if len(nameParts) > 0 {
			firstName = nameParts[0]
			if len(firstName) > 0 {
				firstName = strings.ToUpper(firstName[:1]) + firstName[1:]
			}
		}
		if len(nameParts) > 1 {
			lastName = nameParts[1]
			if len(lastName) > 0 {
				lastName = strings.ToUpper(lastName[:1]) + lastName[1:]
			}
		}
		
		formattedName := strings.TrimSpace(firstName + " " + lastName)
		if formattedName == "" {
			formattedName = userId
		}
		
		newUser := asana.ScimUser{
			ID:       newUserId,
			UserName: userId,
			Name: asana.ScimName{
				GivenName:  firstName,
				FamilyName: lastName,
				Formatted:  formattedName,
			},
			Emails: []asana.ScimEmail{
				{
					Value:   userId,
					Type:    "work",
					Primary: true,
				},
			},
			Active:   true,
			UserType: "enterprise", // Default to enterprise license
			Schemas: []string{
				scimUserSchema,
			},
		}
		
		s.users[newUserId] = newUser
		
		// Return the created user as per Asana API
		user := asana.User{
			BaseResource: asana.BaseResource{
				Gid:          newUserId,
				Name:         formattedName,
				ResourceType: "user",
			},
			Email: userId,
		}
		
		response := map[string]interface{}{
			"data": user,
		}
		
		writeJSONResponse(w, http.StatusOK, response)
		return
	}

	// For user IDs, just verify the user exists
	_, userExists := s.users[userId]
	if !userExists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("User with ID %s not found", userId))
		return
	}

	// Return success (user already exists in the system)
	w.WriteHeader(http.StatusOK)
}

// handleRemoveUserFromWorkspace handles removing a user from a workspace.
func (s *ScimTestServer) handleRemoveUserFromWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract the workspace ID from the path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid URL format")
		return
	}
	
	workspaceId := pathParts[3]

	// Read the request body
	var requestBody struct {
		Data struct {
			User string `json:"user"`
		} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userId := requestBody.Data.User

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the workspace exists
	var workspaceExists bool
	for _, ws := range s.workspaces {
		if ws.Gid == workspaceId {
			workspaceExists = true
			break
		}
	}

	if !workspaceExists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Workspace with ID %s not found", workspaceId))
		return
	}

	// In a real implementation, we would remove user from workspace
	// For this test implementation, we'll just verify that the user exists
	if strings.HasPrefix(userId, "1") {
		_, userExists := s.users[userId]
		if !userExists {
			writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("User with ID %s not found", userId))
			return
		}
	}

	// Return success
	w.WriteHeader(http.StatusOK)
}

// addTestData adds some test users and teams.
func (s *ScimTestServer) addTestData() {
	// Add some test users with license types
	user1 := asana.ScimUser{
		ID:       "1",
		UserName: "john.doe@example.com",
		Name: asana.ScimName{
			GivenName:  "John",
			FamilyName: "Doe",
			Formatted:  "John Doe",
		},
		Emails: []asana.ScimEmail{
			{
				Value:   "john.doe@example.com",
				Type:    "work",
				Primary: true,
			},
		},
		Active:   true,
		Title:    "Software Engineer",
		UserType: "enterprise", // Enterprise license
		Schemas: []string{
			scimUserSchema,
			scimEnterpriseUserSchema,
		},
		EnterpriseExtension: asana.ScimEnterpriseExtension{
			Department: "Engineering",
		},
	}

	user2 := asana.ScimUser{
		ID:       "2",
		UserName: "jane.smith@example.com",
		Name: asana.ScimName{
			GivenName:  "Jane",
			FamilyName: "Smith",
			Formatted:  "Jane Smith",
		},
		Emails: []asana.ScimEmail{
			{
				Value:   "jane.smith@example.com",
				Type:    "work",
				Primary: true,
			},
		},
		Active:   true,
		Title:    "Product Manager",
		UserType: "view only", // View-only license
		Schemas: []string{
			scimUserSchema,
			scimEnterpriseUserSchema,
		},
		EnterpriseExtension: asana.ScimEnterpriseExtension{
			Department: "Product",
		},
	}

	s.users[user1.ID] = user1
	s.users[user2.ID] = user2

	// Add some test teams
	team1 := asana.ScimTeam{
		ID:          "1",
		DisplayName: "Engineering",
		Members: []asana.ScimMember{
			{Value: "1", Display: "John Doe"},
		},
		Schemas: []string{scimGroupSchema},
	}
	team1.Meta.ResourceType = resourceTypeGroup

	team2 := asana.ScimTeam{
		ID:          "2",
		DisplayName: "Product",
		Members: []asana.ScimMember{
			{Value: "2", Display: "Jane Smith"},
		},
		Schemas: []string{scimGroupSchema},
	}
	team2.Meta.ResourceType = resourceTypeGroup

	// Add a new team for testing
	team3 := asana.ScimTeam{
		ID:          "3",
		DisplayName: "Marketing",
		Members:     []asana.ScimMember{},
		Schemas:     []string{scimGroupSchema},
	}
	team3.Meta.ResourceType = resourceTypeGroup

	s.teams[team1.ID] = team1
	s.teams[team2.ID] = team2
	s.teams[team3.ID] = team3
}

// Utility functions.
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errResp := asana.ScimError{
		Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		Detail:  msg,
		Status:  strconv.Itoa(statusCode),
	}
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}

// Meta information endpoint handlers.
func (s *ScimTestServer) handleServiceProviderConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	config := map[string]interface{}{
		"schemas":          []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"documentationUri": "https://developers.asana.com/docs/scim-api",
		"patch": map[string]interface{}{
			"supported": true,
		},
		"bulk": map[string]interface{}{
			"supported":      false,
			"maxOperations":  0,
			"maxPayloadSize": 0,
		},
		"filter": map[string]interface{}{
			"supported":  true,
			"maxResults": 100,
		},
		"changePassword": map[string]interface{}{
			"supported": false,
		},
		"sort": map[string]interface{}{
			"supported": false,
		},
		"etag": map[string]interface{}{
			"supported": false,
		},
		"authenticationSchemes": []map[string]interface{}{
			{
				"type":             "oauth2",
				"name":             "OAuth 2.0",
				"description":      "OAuth 2.0 Bearer Token",
				"specUri":          "https://oauth.net/2/",
				"documentationUri": "https://developers.asana.com/docs/oauth",
			},
		},
	}

	writeJSONResponse(w, http.StatusOK, config)
}

func (s *ScimTestServer) handleResourceTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	resourceTypes := []map[string]interface{}{
		{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":          "User",
			"name":        "User",
			"endpoint":    "/Users",
			"description": "User Account",
			"schema":      scimUserSchema,
			"schemaExtensions": []map[string]interface{}{
				{
					"schema":   scimEnterpriseUserSchema,
					"required": false,
				},
			},
		},
		{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":          "Group",
			"name":        "Group",
			"endpoint":    "/Groups",
			"description": "Group",
			"schema":      scimGroupSchema,
		},
	}

	writeJSONResponse(w, http.StatusOK, resourceTypes)
}

func (s *ScimTestServer) handleSchemas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// For simplicity, we'll return a minimal schema representation
	schemas := []map[string]interface{}{
		{
			"id":          scimUserSchema,
			"name":        "User",
			"description": "User Account",
			"attributes":  []map[string]interface{}{},
		},
		{
			"id":          scimEnterpriseUserSchema,
			"name":        "Enterprise User",
			"description": "Enterprise User Extension",
			"attributes":  []map[string]interface{}{},
		},
		{
			"id":          scimGroupSchema,
			"name":        "Group",
			"description": "Group",
			"attributes":  []map[string]interface{}{},
		},
	}

	writeJSONResponse(w, http.StatusOK, schemas)
}

// User endpoint handlers.
func (s *ScimTestServer) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetUsers(w, r)
	case http.MethodPost:
		s.handleCreateUser(w, r)
	default:
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *ScimTestServer) handleUserById(w http.ResponseWriter, r *http.Request) {
	userId := strings.TrimPrefix(r.URL.Path, "/api/1.0/scim/Users/")
	if userId == "" {
		writeErrorResponse(w, http.StatusBadRequest, "User ID is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetUser(w, r, userId)
	case http.MethodPut:
		s.handleUpdateUser(w, r, userId)
	case http.MethodPatch:
		s.handlePatchUser(w, r, userId)
	case http.MethodDelete:
		s.handleDeleteUser(w, r, userId)
	default:
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *ScimTestServer) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Parse filter if provided
	filter := r.URL.Query().Get("filter")
	var filteredUsers []asana.ScimUser

	if filter != "" {
		// Simple filter parsing for userName eq "value"
		if strings.Contains(filter, "userName eq") {
			parts := strings.Split(filter, "eq")
			if len(parts) == 2 {
				// Extract email from "userName eq \"email@example.com\""
				email := strings.Trim(strings.TrimSpace(parts[1]), "\"")

				for _, user := range s.users {
					if user.UserName == email {
						filteredUsers = append(filteredUsers, user)
						break
					}
				}
			}
		}
	} else {
		// No filter, return all users
		for _, user := range s.users {
			filteredUsers = append(filteredUsers, user)
		}
	}

	response := asana.ScimListResponse{
		Schemas:      []string{scimListResponseSchema},
		TotalResults: len(filteredUsers),
		Resources:    filteredUsers,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

func (s *ScimTestServer) handleGetUser(w http.ResponseWriter, r *http.Request, userId string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userId]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("User with ID %s not found", userId))
		return
	}

	writeJSONResponse(w, http.StatusOK, user)
}

func (s *ScimTestServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var user asana.ScimUser
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if user.UserName == "" {
		writeErrorResponse(w, http.StatusBadRequest, "userName is required")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user already exists
	for _, existingUser := range s.users {
		if existingUser.UserName == user.UserName {
			writeErrorResponse(w, http.StatusConflict, fmt.Sprintf("User with userName %s already exists", user.UserName))
			return
		}
	}

	// Generate a new ID
	nextId := fmt.Sprintf("%d", len(s.users)+1)
	user.ID = nextId

	// Ensure schemas are set
	if user.Schemas == nil {
		user.Schemas = []string{scimUserSchema}
	}

	// Add enterprise schema if extension data is present
	if user.EnterpriseExtension != (asana.ScimEnterpriseExtension{}) {
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

	// Set formatted name if needed
	if user.Name.Formatted == "" && (user.Name.GivenName != "" || user.Name.FamilyName != "") {
		user.Name.Formatted = strings.TrimSpace(user.Name.GivenName + " " + user.Name.FamilyName)
	}

	// If we get formatted name but not given/family names, try to split them
	if user.Name.Formatted != "" && user.Name.GivenName == "" && user.Name.FamilyName == "" {
		parts := strings.SplitN(user.Name.Formatted, " ", 2)
		if len(parts) > 0 {
			user.Name.GivenName = parts[0]
		}
		if len(parts) > 1 {
			user.Name.FamilyName = parts[1]
		}
	}

	// Set primary email
	if len(user.Emails) > 0 {
		for i, email := range user.Emails {
			if email.Primary {
				// Ensure type is set
				if email.Type == "" {
					user.Emails[i].Type = "work"
				}
				break
			}
		}
	}
	
	// Set default license type if not provided
	if user.UserType == "" {
		user.UserType = "enterprise" // Default to enterprise license
	}

	s.users[user.ID] = user
	writeJSONResponse(w, http.StatusCreated, user)
}

func (s *ScimTestServer) handleUpdateUser(w http.ResponseWriter, r *http.Request, userId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.users[userId]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("User with ID %s not found", userId))
		return
	}

	var updatedUser asana.ScimUser
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Preserve the user ID
	updatedUser.ID = userId

	// Ensure schemas are set
	if updatedUser.Schemas == nil {
		updatedUser.Schemas = []string{scimUserSchema}
	}

	// Add enterprise schema if extension data is present
	if updatedUser.EnterpriseExtension != (asana.ScimEnterpriseExtension{}) {
		hasEnterpriseSchema := false
		for _, schema := range updatedUser.Schemas {
			if schema == scimEnterpriseUserSchema {
				hasEnterpriseSchema = true
				break
			}
		}
		if !hasEnterpriseSchema {
			updatedUser.Schemas = append(updatedUser.Schemas, scimEnterpriseUserSchema)
		}
	}

	s.users[userId] = updatedUser
	writeJSONResponse(w, http.StatusOK, updatedUser)
}

func (s *ScimTestServer) handlePatchUser(w http.ResponseWriter, r *http.Request, userId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userId]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("User with ID %s not found", userId))
		return
	}

	var patchReq asana.ScimPatch
	if err := json.NewDecoder(r.Body).Decode(&patchReq); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Apply each operation
	for _, op := range patchReq.Operations {
		switch op.Op {
		case "replace":
			// Handle different attributes based on path
			if op.Path == "" {
				// Replace the value directly in the user object
				if opValue, ok := op.Value.(map[string]interface{}); ok {
					for key, value := range opValue {
						switch key {
						case "active":
							if boolValue, ok := value.(bool); ok {
								user.Active = boolValue
							}
						case "title":
							if strValue, ok := value.(string); ok {
								user.Title = strValue
							}
						case "name":
							if nameValue, ok := value.(map[string]interface{}); ok {
								if formatted, ok := nameValue["formatted"].(string); ok {
									user.Name.Formatted = formatted
								}
							}
						case "userName":
							if strValue, ok := value.(string); ok {
								user.UserName = strValue
							}
						case "preferredLanguage":
							if strValue, ok := value.(string); ok {
								user.PreferredLanguage = strValue
							}
						case "userType":
							if strValue, ok := value.(string); ok {
								user.UserType = strValue
							}
						case "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User":
							if extValue, ok := value.(map[string]interface{}); ok {
								if dept, ok := extValue["department"].(string); ok {
									user.EnterpriseExtension.Department = dept
								}
							}
						}
					}
				}
			} else {
				// Handle specific path replacements
				switch op.Path {
				case "active":
					if boolValue, ok := op.Value.(bool); ok {
						user.Active = boolValue
					}
				case "title":
					if strValue, ok := op.Value.(string); ok {
						user.Title = strValue
					}
				case "name.formatted":
					if strValue, ok := op.Value.(string); ok {
						user.Name.Formatted = strValue
					}
				case "userName":
					if strValue, ok := op.Value.(string); ok {
						user.UserName = strValue
					}
				case "preferredLanguage":
					if strValue, ok := op.Value.(string); ok {
						user.PreferredLanguage = strValue
					}
				case "userType":
					if strValue, ok := op.Value.(string); ok {
						user.UserType = strValue
					}
				case "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department":
					if strValue, ok := op.Value.(string); ok {
						user.EnterpriseExtension.Department = strValue
					}
				}
			}
		}
		// Add support for add and remove operations if needed
	}

	s.users[userId] = user
	writeJSONResponse(w, http.StatusOK, user)
}

func (s *ScimTestServer) handleDeleteUser(w http.ResponseWriter, r *http.Request, userId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.users[userId]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("User with ID %s not found", userId))
		return
	}

	// In a real implementation, we might deactivate the user instead of removing
	// But for this test server, we'll just remove the user
	delete(s.users, userId)
	w.WriteHeader(http.StatusNoContent)
}

// Team endpoint handlers.
func (s *ScimTestServer) handleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetTeams(w, r)
	case http.MethodPost:
		s.handleCreateTeam(w, r)
	default:
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *ScimTestServer) handleGroupById(w http.ResponseWriter, r *http.Request) {
	teamId := strings.TrimPrefix(r.URL.Path, "/api/1.0/scim/Groups/")
	if teamId == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Team ID is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetTeam(w, r, teamId)
	case http.MethodPut:
		s.handleUpdateTeam(w, r, teamId)
	case http.MethodPatch:
		s.handlePatchTeam(w, r, teamId)
	default:
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *ScimTestServer) handleGetTeams(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Parse filter if provided
	filter := r.URL.Query().Get("filter")
	var filteredTeams []asana.ScimTeam

	if filter != "" {
		// Simple filter parsing for displayName eq "value"
		if strings.Contains(filter, "displayName eq") {
			parts := strings.Split(filter, "eq")
			if len(parts) == 2 {
				// Extract name from "displayName eq \"Marketing\""
				name := strings.Trim(strings.TrimSpace(parts[1]), "\"")

				for _, team := range s.teams {
					if team.DisplayName == name {
						filteredTeams = append(filteredTeams, team)
						break
					}
				}
			}
		}
	} else {
		// No filter, return all teams
		for _, team := range s.teams {
			filteredTeams = append(filteredTeams, team)
		}
	}

	response := asana.ScimTeamListResponse{
		Schemas:      []string{scimListResponseSchema},
		TotalResults: len(filteredTeams),
		Resources:    filteredTeams,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

func (s *ScimTestServer) handleGetTeam(w http.ResponseWriter, r *http.Request, teamId string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, exists := s.teams[teamId]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Team with ID %s not found", teamId))
		return
	}

	writeJSONResponse(w, http.StatusOK, team)
}

func (s *ScimTestServer) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	var team asana.ScimTeam
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if team.DisplayName == "" {
		writeErrorResponse(w, http.StatusBadRequest, "displayName is required")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if team already exists
	for _, existingTeam := range s.teams {
		if existingTeam.DisplayName == team.DisplayName {
			writeErrorResponse(w, http.StatusConflict, fmt.Sprintf("Team with displayName %s already exists", team.DisplayName))
			return
		}
	}

	// Generate a new ID
	nextId := fmt.Sprintf("%d", len(s.teams)+1)
	team.ID = nextId

	// Ensure schemas are set
	if team.Schemas == nil {
		team.Schemas = []string{scimGroupSchema}
	}

	// Set meta
	team.Meta.ResourceType = resourceTypeGroup

	s.teams[team.ID] = team
	writeJSONResponse(w, http.StatusCreated, team)
}

func (s *ScimTestServer) handleUpdateTeam(w http.ResponseWriter, r *http.Request, teamId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.teams[teamId]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Team with ID %s not found", teamId))
		return
	}

	var updatedTeam asana.ScimTeam
	if err := json.NewDecoder(r.Body).Decode(&updatedTeam); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Preserve the team ID
	updatedTeam.ID = teamId

	// Ensure schemas are set
	if updatedTeam.Schemas == nil {
		updatedTeam.Schemas = []string{scimGroupSchema}
	}

	// Set meta
	updatedTeam.Meta.ResourceType = resourceTypeGroup

	s.teams[teamId] = updatedTeam
	writeJSONResponse(w, http.StatusOK, updatedTeam)
}

func (s *ScimTestServer) handlePatchTeam(w http.ResponseWriter, r *http.Request, teamId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, exists := s.teams[teamId]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Team with ID %s not found", teamId))
		return
	}

	var patchReq asana.ScimPatch
	if err := json.NewDecoder(r.Body).Decode(&patchReq); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Apply each operation
	for _, op := range patchReq.Operations {
		switch op.Op {
		case "add":
			if op.Path == membersPath {
				// Add members to the team
				if membersValue, ok := op.Value.([]interface{}); ok {
					for _, memberValue := range membersValue {
						if memberMap, ok := memberValue.(map[string]interface{}); ok {
							if memberID, ok := memberMap["value"].(string); ok {
								// Check if member is already in the team
								found := false
								for _, existingMember := range team.Members {
									if existingMember.Value == memberID {
										found = true
										break
									}
								}
								if !found {
									// Get user display name if available
									var display string
									if user, exists := s.users[memberID]; exists {
										display = user.Name.Formatted
									}
									team.Members = append(team.Members, asana.ScimMember{
										Value:   memberID,
										Display: display,
									})
								}
							}
						}
					}
				}
			}
		case "remove":
			if op.Path == membersPath {
				// Remove members from the team
				if membersValue, ok := op.Value.([]interface{}); ok {
					for _, memberValue := range membersValue {
						if memberMap, ok := memberValue.(map[string]interface{}); ok {
							if memberID, ok := memberMap["value"].(string); ok {
								// Filter out the member to be removed
								var newMembers []asana.ScimMember
								for _, existingMember := range team.Members {
									if existingMember.Value != memberID {
										newMembers = append(newMembers, existingMember)
									}
								}
								team.Members = newMembers
							}
						}
					}
				}
			}
		case "replace":
			if op.Path == membersPath {
				// Replace all members
				if membersValue, ok := op.Value.([]interface{}); ok {
					var newMembers []asana.ScimMember
					for _, memberValue := range membersValue {
						if memberMap, ok := memberValue.(map[string]interface{}); ok {
							if memberID, ok := memberMap["value"].(string); ok {
								// Get user display name if available
								var display string
								if user, exists := s.users[memberID]; exists {
									display = user.Name.Formatted
								}
								newMembers = append(newMembers, asana.ScimMember{
									Value:   memberID,
									Display: display,
								})
							}
						}
					}
					team.Members = newMembers
				}
			} else if op.Path == "displayName" {
				if displayName, ok := op.Value.(string); ok {
					team.DisplayName = displayName
				}
			}
		}
	}

	s.teams[teamId] = team
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	server := NewScimTestServer()
	log.Fatal(server.Start(*port))
}
