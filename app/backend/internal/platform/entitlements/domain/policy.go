package domain

type ProductKey string
type TenantRole string
type ProductRole string
type Permission string

const (
	ProductPlanner ProductKey = "planner_link"
	ProductFinance ProductKey = "finance_link"
	ProductInfra   ProductKey = "infra_link"
	ProductLoka    ProductKey = "loka_link"

	TenantRoleOwner        TenantRole = "owner"
	TenantRoleAdmin        TenantRole = "admin"
	TenantRoleBillingAdmin TenantRole = "billing_admin"
	TenantRoleMember       TenantRole = "member"
	TenantRoleViewer       TenantRole = "viewer"

	ProductRoleAdmin  ProductRole = "admin"
	ProductRoleMember ProductRole = "member"
	ProductRoleViewer ProductRole = "viewer"

	PermissionPlannerTaskRead     Permission = "planner.task.read"
	PermissionPlannerTaskCreate   Permission = "planner.task.create"
	PermissionPlannerTaskUpdate   Permission = "planner.task.update"
	PermissionPlannerTaskDelete   Permission = "planner.task.delete"
	PermissionPlannerMemberManage Permission = "planner.member.manage"

	PermissionFinanceInvoiceRead   Permission = "finance.invoice.read"
	PermissionFinanceInvoiceCreate Permission = "finance.invoice.create"
	PermissionFinanceReportRead    Permission = "finance.report.read"
	PermissionFinanceMemberManage  Permission = "finance.member.manage"
)

type AuthorizationPolicy struct{}

func NewAuthorizationPolicy() AuthorizationPolicy {
	return AuthorizationPolicy{}
}

var productPermissions = map[ProductKey]map[ProductRole][]Permission{
	ProductPlanner: {
		ProductRoleViewer: {PermissionPlannerTaskRead},
		ProductRoleMember: {PermissionPlannerTaskRead, PermissionPlannerTaskCreate, PermissionPlannerTaskUpdate},
		ProductRoleAdmin:  {PermissionPlannerTaskRead, PermissionPlannerTaskCreate, PermissionPlannerTaskUpdate, PermissionPlannerTaskDelete, PermissionPlannerMemberManage},
	},
	ProductFinance: {
		ProductRoleViewer: {PermissionFinanceInvoiceRead, PermissionFinanceReportRead},
		ProductRoleMember: {PermissionFinanceInvoiceRead, PermissionFinanceInvoiceCreate, PermissionFinanceReportRead},
		ProductRoleAdmin:  {PermissionFinanceInvoiceRead, PermissionFinanceInvoiceCreate, PermissionFinanceReportRead, PermissionFinanceMemberManage},
	},
	ProductInfra: {
		ProductRoleViewer: {"infra.asset.read"},
		ProductRoleMember: {"infra.asset.read", "infra.asset.update"},
		ProductRoleAdmin:  {"infra.asset.read", "infra.asset.update", "infra.member.manage"},
	},
	ProductLoka: {
		ProductRoleViewer: {"loka.place.read"},
		ProductRoleMember: {"loka.place.read", "loka.place.update"},
		ProductRoleAdmin:  {"loka.place.read", "loka.place.update", "loka.member.manage"},
	},
}

func (AuthorizationPolicy) PermissionsForProductRole(productKey ProductKey, role ProductRole) []Permission {
	if byRole, ok := productPermissions[productKey]; ok {
		if permissions, ok := byRole[role]; ok {
			return append([]Permission(nil), permissions...)
		}
	}
	return nil
}

func (AuthorizationPolicy) ValidProductRole(role ProductRole) bool {
	return role == ProductRoleAdmin || role == ProductRoleMember || role == ProductRoleViewer
}

func (AuthorizationPolicy) ValidTenantRole(role TenantRole) bool {
	switch role {
	case TenantRoleOwner, TenantRoleAdmin, TenantRoleBillingAdmin, TenantRoleMember, TenantRoleViewer:
		return true
	default:
		return false
	}
}

func (AuthorizationPolicy) TenantRoleAllows(role TenantRole, allowed ...TenantRole) bool {
	if role == TenantRoleOwner {
		return true
	}
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}
