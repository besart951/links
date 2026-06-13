import type { AccessSnapshot, Permission } from './domain/access-snapshot.js';
import type { ProductKey } from './domain/product.js';
import type { ProductRole, TenantRole } from './domain/roles.js';
import { defaultAccessPolicy } from './policies/access-policy.js';

export * from './domain/access-snapshot.js';
export * from './domain/product.js';
export * from './domain/roles.js';
export * from './policies/access-policy.js';

export function hasProduct(access: AccessSnapshot | null | undefined, productKey: ProductKey | string): boolean {
	return defaultAccessPolicy.hasProduct(access, productKey);
}

export function can(access: AccessSnapshot | null | undefined, permission: Permission): boolean {
	return defaultAccessPolicy.can(access, permission);
}

export function isTenantOwner(access: AccessSnapshot | null | undefined): boolean {
	return defaultAccessPolicy.isTenantOwner(access);
}

export function hasTenantRole(
	access: AccessSnapshot | null | undefined,
	roles: TenantRole | TenantRole[]
): boolean {
	return defaultAccessPolicy.hasTenantRole(access, roles);
}

export function roleAtLeast(role: TenantRole, minimum: TenantRole): boolean;
export function roleAtLeast(role: ProductRole, minimum: ProductRole): boolean;
export function roleAtLeast(role: string, minimum: string): boolean {
	return defaultAccessPolicy.roleAtLeast(role, minimum);
}
