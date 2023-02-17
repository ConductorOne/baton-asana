package asana

type BaseResource struct {
	Gid          string `json:"gid"`
	Name         string `json:"name"`
	ResourceType string `json:"resource_type"`
}

type User struct {
	BaseResource
	Email string `json:"email"`
}

type Team struct {
	BaseResource
	Email string `json:"email"`
}

type Workspace struct {
	BaseResource
	IsOrganization bool     `json:"is_organization"`
	EmailDomains   []string `json:"email_domains"`
}

type WorkspaceMembership struct {
	Gid          string    `json:"gid"`
	ResourceType string    `json:"resource_type"`
	User         User      `json:"user"`
	Workspace    Workspace `json:"workspace"`
	IsActive     bool      `json:"is_active"`
	IsAdmin      bool      `json:"is_admin"`
	IsGuest      bool      `json:"is_guest"`
}

type PaginationData struct {
	Offset string `json:"offset,omitempty"`
}

type TeamMembership struct {
	Gid             string `json:"gid"`
	ResourceType    string `json:"resource_type"`
	User            User   `json:"user"`
	Team            Team   `json:"team"`
	IsAdmin         bool   `json:"is_admin"`
	IsGuest         bool   `json:"is_guest"`
	IsLimitedAccess bool   `json:"is_limited_access"`
}
