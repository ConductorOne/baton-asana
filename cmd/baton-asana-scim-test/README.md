# Asana SCIM API Test Server

This is a mock implementation of the Asana SCIM API for testing the baton-asana connector. It implements the core SCIM endpoints as defined in the Asana API documentation.

## Features

- Full implementation of SCIM User endpoints (GET, POST, PUT, PATCH, DELETE)
- Full implementation of SCIM Group endpoints (GET, POST, PUT, PATCH)
- Support for meta information endpoints (ServiceProviderConfig, ResourceTypes, Schemas)
- Filtering support for Users and Groups
- Built-in test data for immediate testing

## Usage

### Starting the server

```bash
# Default port (8080)
./baton-asana-scim-test

# Custom port
./baton-asana-scim-test -port 9090
```

### Testing with the baton-asana connector

1. Start the SCIM test server
2. Configure the baton-asana connector to use the test server URL:

```bash
./dist/darwin_arm64/baton-asana \
  --token test-token \
  --use-service-account \
  --use-scim-api \
  --asana-api-url http://localhost:8080/api/1.0
```

## Available Endpoints

### Meta Information

- `GET /api/1.0/scim/ServiceProviderConfig` - Service provider configuration
- `GET /api/1.0/scim/ResourceTypes` - Available resource types
- `GET /api/1.0/scim/Schemas` - Available schemas

### Users

- `GET /api/1.0/scim/Users` - List all users
- `GET /api/1.0/scim/Users?filter=userName eq "john.doe@example.com"` - Filter users
- `GET /api/1.0/scim/Users/{id}` - Get user by ID
- `POST /api/1.0/scim/Users` - Create a new user
- `PUT /api/1.0/scim/Users/{id}` - Update user (replace)
- `PATCH /api/1.0/scim/Users/{id}` - Update user (patch)
- `DELETE /api/1.0/scim/Users/{id}` - Delete user

### Groups

- `GET /api/1.0/scim/Groups` - List all groups
- `GET /api/1.0/scim/Groups?filter=displayName eq "Engineering"` - Filter groups
- `GET /api/1.0/scim/Groups/{id}` - Get group by ID
- `POST /api/1.0/scim/Groups` - Create a new group
- `PUT /api/1.0/scim/Groups/{id}` - Update group (replace)
- `PATCH /api/1.0/scim/Groups/{id}` - Update group (patch)

## Test Data

The server comes pre-loaded with test data:

### Users

1. John Doe (ID: 1)
   - Email: john.doe@example.com
   - Title: Software Engineer
   - Department: Engineering
   - Active: true
   - License: enterprise

2. Jane Smith (ID: 2)
   - Email: jane.smith@example.com
   - Title: Product Manager
   - Department: Product
   - Active: true
   - License: view only

3. Alex Johnson (ID: 3)
   - Email: alex.johnson@example.com
   - Title: Designer
   - Department: Design
   - Active: true
   - License: none (for testing license grant)

### Groups

1. Engineering (ID: 1)
   - Members: John Doe (ID: 1)

2. Product (ID: 2)
   - Members: Jane Smith (ID: 2)

## Example Requests

### List All Users

```bash
curl -X GET "http://localhost:8080/api/1.0/scim/Users"
```

### Create a New User

```bash
curl -X POST "http://localhost:8080/api/1.0/scim/Users" \
  -H "Content-Type: application/json" \
  -d '{
    "schemas": [
      "urn:ietf:params:scim:schemas:core:2.0:User",
      "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
    ],
    "userName": "alex.johnson@example.com",
    "name": {
      "formatted": "Alex Johnson"
    },
    "emails": [
      {
        "primary": true,
        "value": "alex.johnson@example.com"
      }
    ],
    "active": true,
    "title": "Designer",
    "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": {
      "department": "Design"
    }
  }'
```

### Update a User (PATCH)

```bash
curl -X PATCH "http://localhost:8080/api/1.0/scim/Users/1" \
  -H "Content-Type: application/json" \
  -d '{
    "schemas": [
      "urn:ietf:params:scim:api:messages:2.0:PatchOp"
    ],
    "Operations": [
      {
        "op": "replace",
        "value": {
          "title": "Senior Software Engineer"
        }
      }
    ]
  }'
```

### Grant a License (PATCH userType)

```bash
curl -X PATCH "http://localhost:8080/api/1.0/scim/Users/3" \
  -H "Content-Type: application/json" \
  -d '{
    "schemas": [
      "urn:ietf:params:scim:api:messages:2.0:PatchOp"
    ],
    "Operations": [
      {
        "op": "replace",
        "path": "userType",
        "value": "enterprise"
      }
    ]
  }'
```

### Revoke Enterprise License (Change to View Only)

```bash
curl -X PATCH "http://localhost:8080/api/1.0/scim/Users/1" \
  -H "Content-Type: application/json" \
  -d '{
    "schemas": [
      "urn:ietf:params:scim:api:messages:2.0:PatchOp"
    ],
    "Operations": [
      {
        "op": "replace",
        "path": "userType",
        "value": "view only"
      }
    ]
  }'
```

### Revoke View Only License (Deprovision User)

```bash
curl -X PATCH "http://localhost:8080/api/1.0/scim/Users/2" \
  -H "Content-Type: application/json" \
  -d '{
    "schemas": [
      "urn:ietf:params:scim:api:messages:2.0:PatchOp"
    ],
    "Operations": [
      {
        "op": "replace",
        "path": "active",
        "value": false
      }
    ]
  }'
```

### Create a New Group

```bash
curl -X POST "http://localhost:8080/api/1.0/scim/Groups" \
  -H "Content-Type: application/json" \
  -d '{
    "schemas": [
      "urn:ietf:params:scim:schemas:core:2.0:Group"
    ],
    "displayName": "Design",
    "members": [
      {"value": "3"}
    ]
  }'
```