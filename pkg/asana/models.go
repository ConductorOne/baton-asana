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

type baseMutationBody struct {
	Data any `json:"data"`
}

// AsanaError represents the error response from the Asana API.
type AsanaError struct {
	Errors []struct {
		Message string `json:"message"`
		Help    string `json:"help,omitempty"`
		Phrase  string `json:"phrase,omitempty"`
	} `json:"errors"`
}

// Message implements the uhttp.ErrorResponse interface.
func (e *AsanaError) Message() string {
	if len(e.Errors) == 0 {
		return "Unknown error from Asana API"
	}

	// Return the first error message
	return e.Errors[0].Message
}

// SCIM API Models

// ScimError represents SCIM API error response.
type ScimError struct {
	Schemas []string `json:"schemas"`
	Detail  string   `json:"detail"`
	Status  string   `json:"status"`
}

// Message implements the uhttp.ErrorResponse interface.
func (e *ScimError) Message() string {
	if e.Detail != "" {
		return e.Detail
	}
	return "Unknown error from SCIM API"
}

// ScimAddress represents a SCIM address.
type ScimAddress struct {
	Locality string `json:"locality,omitempty"`
	Region   string `json:"region,omitempty"`
	Country  string `json:"country,omitempty"`
	Type     string `json:"type,omitempty"`
	Primary  bool   `json:"primary,omitempty"`
}

// ScimPhoneNumber represents a SCIM phone number.
type ScimPhoneNumber struct {
	Value   string `json:"value,omitempty"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// ScimEmail represents a SCIM email.
type ScimEmail struct {
	Value   string `json:"value,omitempty"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// ScimName represents a SCIM name.
type ScimName struct {
	FamilyName string `json:"familyName,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
	Formatted  string `json:"formatted,omitempty"`
}

// ScimEnterpriseExtension represents the enterprise extension data for a user.
type ScimEnterpriseExtension struct {
	Department     string `json:"department,omitempty"`
	CostCenter     string `json:"costCenter,omitempty"`
	Organization   string `json:"organization,omitempty"`
	Division       string `json:"division,omitempty"`
	EmployeeNumber string `json:"employeeNumber,omitempty"`
	Manager        *struct {
		Value string `json:"value,omitempty"`
	} `json:"manager,omitempty"`
}

// ScimUser represents a user in the SCIM API.
type ScimUser struct {
	ID                string            `json:"id,omitempty"`
	Schemas           []string          `json:"schemas,omitempty"`
	Name              ScimName          `json:"name,omitempty"`
	UserName          string            `json:"userName,omitempty"`
	Emails            []ScimEmail       `json:"emails,omitempty"`
	Active            bool              `json:"active,omitempty"`
	Title             string            `json:"title,omitempty"`
	PreferredLanguage string            `json:"preferredLanguage,omitempty"`
	Addresses         []ScimAddress     `json:"addresses,omitempty"`
	PhoneNumbers      []ScimPhoneNumber `json:"phoneNumbers,omitempty"`
	UserType          string            `json:"userType,omitempty"`

	// Enterprise extension data - using the full schema URI as the field name
	EnterpriseExtension ScimEnterpriseExtension `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User,omitempty"`
}

// ScimListResponse represents a list response from the SCIM API.
type ScimListResponse struct {
	Schemas      []string   `json:"schemas"`
	TotalResults int        `json:"totalResults"`
	Resources    []ScimUser `json:"Resources"`
}

// ScimPatchOperation represents a SCIM PATCH operation.
type ScimPatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

// ScimPatch represents a SCIM PATCH request.
type ScimPatch struct {
	Schemas    []string             `json:"schemas"`
	Operations []ScimPatchOperation `json:"Operations"`
}
