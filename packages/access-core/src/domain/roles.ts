export type TenantRole = 'owner' | 'admin' | 'billing_admin' | 'member' | 'viewer';
export type ProductRole = 'admin' | 'member' | 'viewer';

export const tenantRank: Readonly<Record<TenantRole, number>> = {
	viewer: 0,
	member: 1,
	billing_admin: 2,
	admin: 3,
	owner: 4
};

export const productRank: Readonly<Record<ProductRole, number>> = {
	viewer: 0,
	member: 1,
	admin: 2
};

export function isTenantRole(value: string): value is TenantRole {
	return value in tenantRank;
}

export function isProductRole(value: string): value is ProductRole {
	return value in productRank;
}
