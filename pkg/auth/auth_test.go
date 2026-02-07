package auth

import (
	"context"
	"testing"
)

func TestBasicRole(t *testing.T) {
	perms := []Permission{
		NewPermission("task", "read"),
		NewPermission("task", "write"),
	}
	role := NewRole("developer", perms)

	if role.Name() != "developer" {
		t.Errorf("expected role name 'developer', got '%s'", role.Name())
	}

	if len(role.Permissions()) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(role.Permissions()))
	}
}

func TestBasicPermission(t *testing.T) {
	perm := NewPermission("workspace", "execute")

	if perm.Resource() != "workspace" {
		t.Errorf("expected resource 'workspace', got '%s'", perm.Resource())
	}

	if perm.Action() != "execute" {
		t.Errorf("expected action 'execute', got '%s'", perm.Action())
	}

	expected := "workspace:execute"
	if perm.String() != expected {
		t.Errorf("expected string '%s', got '%s'", expected, perm.String())
	}
}

func TestNoOpAuthorizer(t *testing.T) {
	auth := NewNoOpAuthorizer()
	ctx := context.Background()

	role := NewRole("test", []Permission{})

	// NoOpAuthorizer should allow everything
	err := auth.Authorize(ctx, role, "task", "delete")
	if err != nil {
		t.Errorf("NoOpAuthorizer should allow all operations, got error: %v", err)
	}

	perm := NewPermission("config", "write")
	if !auth.HasPermission(role, perm) {
		t.Error("NoOpAuthorizer should return true for HasPermission")
	}
}

func TestDefaultAuthorizerWithPermissions(t *testing.T) {
	auth := NewDefaultAuthorizer()
	ctx := context.Background()

	perms := []Permission{
		NewPermission("task", "read"),
		NewPermission("task", "write"),
		NewPermission("workspace", "read"),
	}
	role := NewRole("developer", perms)

	tests := []struct {
		name     string
		resource string
		action   string
		wantErr  bool
	}{
		{"allowed read task", "task", "read", false},
		{"allowed write task", "task", "write", false},
		{"allowed read workspace", "workspace", "read", false},
		{"denied write workspace", "workspace", "write", true},
		{"denied delete task", "task", "delete", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.Authorize(ctx, role, tt.resource, tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authorize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultAuthorizerWithWildcard(t *testing.T) {
	auth := NewDefaultAuthorizer()
	ctx := context.Background()

	// Admin role with wildcard permissions
	adminPerms := []Permission{
		NewPermission("*", "*"),
	}
	adminRole := NewRole("admin", adminPerms)

	// Admin should be authorized for everything
	err := auth.Authorize(ctx, adminRole, "task", "delete")
	if err != nil {
		t.Errorf("admin with wildcard should be authorized for everything, got: %v", err)
	}

	err = auth.Authorize(ctx, adminRole, "config", "write")
	if err != nil {
		t.Errorf("admin with wildcard should be authorized for everything, got: %v", err)
	}
}

func TestDefaultAuthorizerResourceWildcard(t *testing.T) {
	auth := NewDefaultAuthorizer()
	ctx := context.Background()

	// Role with wildcard resource
	perms := []Permission{
		NewPermission("*", "read"),
	}
	role := NewRole("viewer", perms)

	// Should allow read on any resource
	err := auth.Authorize(ctx, role, "task", "read")
	if err != nil {
		t.Errorf("wildcard resource should allow read on task: %v", err)
	}

	err = auth.Authorize(ctx, role, "workspace", "read")
	if err != nil {
		t.Errorf("wildcard resource should allow read on workspace: %v", err)
	}

	// Should deny write
	err = auth.Authorize(ctx, role, "task", "write")
	if err == nil {
		t.Error("wildcard resource with read action should deny write")
	}
}

func TestDefaultAuthorizerActionWildcard(t *testing.T) {
	auth := NewDefaultAuthorizer()
	ctx := context.Background()

	// Role with wildcard action
	perms := []Permission{
		NewPermission("task", "*"),
	}
	role := NewRole("taskmaster", perms)

	// Should allow any action on task
	err := auth.Authorize(ctx, role, "task", "read")
	if err != nil {
		t.Errorf("wildcard action should allow read on task: %v", err)
	}

	err = auth.Authorize(ctx, role, "task", "write")
	if err != nil {
		t.Errorf("wildcard action should allow write on task: %v", err)
	}

	err = auth.Authorize(ctx, role, "task", "delete")
	if err != nil {
		t.Errorf("wildcard action should allow delete on task: %v", err)
	}

	// Should deny other resources
	err = auth.Authorize(ctx, role, "workspace", "read")
	if err == nil {
		t.Error("wildcard action on task should deny workspace")
	}
}

func TestHasPermission(t *testing.T) {
	auth := NewDefaultAuthorizer()

	perms := []Permission{
		NewPermission("task", "read"),
		NewPermission("task", "write"),
	}
	role := NewRole("developer", perms)

	tests := []struct {
		name       string
		permission Permission
		want       bool
	}{
		{"has task:read", NewPermission("task", "read"), true},
		{"has task:write", NewPermission("task", "write"), true},
		{"lacks task:delete", NewPermission("task", "delete"), false},
		{"lacks workspace:read", NewPermission("workspace", "read"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.HasPermission(role, tt.permission)
			if got != tt.want {
				t.Errorf("HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmptyRole(t *testing.T) {
	auth := NewDefaultAuthorizer()
	ctx := context.Background()

	emptyRole := NewRole("guest", []Permission{})

	err := auth.Authorize(ctx, emptyRole, "task", "read")
	if err == nil {
		t.Error("empty role should be denied all permissions")
	}

	perm := NewPermission("task", "read")
	if auth.HasPermission(emptyRole, perm) {
		t.Error("empty role should not have any permissions")
	}
}
