package domain

import "testing"

func TestPermissionsForProductRole(t *testing.T) {
	policy := NewAuthorizationPolicy()

	viewer := policy.PermissionsForProductRole(ProductPlanner, ProductRoleViewer)
	if !contains(viewer, PermissionPlannerTaskRead) {
		t.Fatalf("viewer permissions = %v, want planner.task.read", viewer)
	}
	if contains(viewer, PermissionPlannerTaskCreate) {
		t.Fatalf("viewer permissions = %v, did not want create", viewer)
	}

	member := policy.PermissionsForProductRole(ProductPlanner, ProductRoleMember)
	if !contains(member, PermissionPlannerTaskCreate) {
		t.Fatalf("member permissions = %v, want planner.task.create", member)
	}
	if contains(member, PermissionPlannerTaskDelete) {
		t.Fatalf("member permissions = %v, did not want delete", member)
	}

	admin := policy.PermissionsForProductRole(ProductPlanner, ProductRoleAdmin)
	if !contains(admin, PermissionPlannerTaskDelete) {
		t.Fatalf("admin permissions = %v, want planner.task.delete", admin)
	}
}

func TestTenantRoleAllowsOwnerEverything(t *testing.T) {
	policy := NewAuthorizationPolicy()

	if !policy.TenantRoleAllows(TenantRoleOwner, TenantRoleAdmin) {
		t.Fatal("owner should satisfy admin-only checks")
	}
	if policy.TenantRoleAllows(TenantRoleMember, TenantRoleAdmin) {
		t.Fatal("member should not satisfy admin-only checks")
	}
}

func contains(values []Permission, candidate Permission) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
