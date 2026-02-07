// Package auth provides authentication and authorization interfaces for EAS.
// This is a stub implementation for v1, with planned SSO/OIDC integration.
package auth

import (
	"context"
	"fmt"
)

// Role represents a user role in the system.
type Role interface {
	// Name returns the role name (e.g., "admin", "developer", "viewer")
	Name() string
	// Permissions returns the set of permissions granted to this role
	Permissions() []Permission
}

// Permission represents a specific permission or capability.
type Permission interface {
	// Resource returns the resource type (e.g., "task", "workspace", "config")
	Resource() string
	// Action returns the action allowed (e.g., "read", "write", "execute")
	Action() string
	// String returns a human-readable representation
	String() string
}

// Authorizer checks if operations are authorized.
type Authorizer interface {
	// Authorize checks if the given role has permission to perform an action on a resource
	Authorize(ctx context.Context, role Role, resource, action string) error
	// HasPermission checks if a role has a specific permission
	HasPermission(role Role, permission Permission) bool
}

// basicRole implements the Role interface.
type basicRole struct {
	name        string
	permissions []Permission
}

func (r *basicRole) Name() string {
	return r.name
}

func (r *basicRole) Permissions() []Permission {
	return r.permissions
}

// NewRole creates a new role with the given name and permissions.
func NewRole(name string, permissions []Permission) Role {
	return &basicRole{
		name:        name,
		permissions: permissions,
	}
}

// basicPermission implements the Permission interface.
type basicPermission struct {
	resource string
	action   string
}

func (p *basicPermission) Resource() string {
	return p.resource
}

func (p *basicPermission) Action() string {
	return p.action
}

func (p *basicPermission) String() string {
	return fmt.Sprintf("%s:%s", p.resource, p.action)
}

// NewPermission creates a new permission for a resource and action.
func NewPermission(resource, action string) Permission {
	return &basicPermission{
		resource: resource,
		action:   action,
	}
}

// NoOpAuthorizer is a stub authorizer that allows all operations.
// This is for v1 development; production systems should use a real authorizer.
type NoOpAuthorizer struct{}

// NewNoOpAuthorizer creates a new no-op authorizer.
func NewNoOpAuthorizer() *NoOpAuthorizer {
	return &NoOpAuthorizer{}
}

// Authorize always returns nil (allows all operations).
func (a *NoOpAuthorizer) Authorize(ctx context.Context, role Role, resource, action string) error {
	return nil
}

// HasPermission always returns true (allows all permissions).
func (a *NoOpAuthorizer) HasPermission(role Role, permission Permission) bool {
	return true
}

// DefaultAuthorizer implements a simple role-based authorizer.
type DefaultAuthorizer struct{}

// NewDefaultAuthorizer creates a new default authorizer.
func NewDefaultAuthorizer() *DefaultAuthorizer {
	return &DefaultAuthorizer{}
}

// Authorize checks if the role has the required permission.
func (a *DefaultAuthorizer) Authorize(ctx context.Context, role Role, resource, action string) error {
	for _, perm := range role.Permissions() {
		// Exact match
		if perm.Resource() == resource && perm.Action() == action {
			return nil
		}
		// Wildcard support - both must match
		resourceMatch := perm.Resource() == resource || perm.Resource() == "*"
		actionMatch := perm.Action() == action || perm.Action() == "*"
		if resourceMatch && actionMatch {
			return nil
		}
	}
	return fmt.Errorf("unauthorized: role '%s' lacks permission %s:%s", role.Name(), resource, action)
}

// HasPermission checks if the role has a specific permission.
func (a *DefaultAuthorizer) HasPermission(role Role, permission Permission) bool {
	for _, perm := range role.Permissions() {
		// Exact match
		if perm.Resource() == permission.Resource() && perm.Action() == permission.Action() {
			return true
		}
		// Wildcard support - both must match
		resourceMatch := perm.Resource() == permission.Resource() || perm.Resource() == "*"
		actionMatch := perm.Action() == permission.Action() || perm.Action() == "*"
		if resourceMatch && actionMatch {
			return true
		}
	}
	return false
}
