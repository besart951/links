import type { AccessSnapshot, Permission } from '../domain/access-snapshot.js';
import type { ProductKey } from '../domain/product.js';
import { isProductRole, isTenantRole, productRank, tenantRank, type ProductRole, type TenantRole } from '../domain/roles.js';

export class AccessPolicy {
	hasProduct(access: AccessSnapshot | null | undefined, productKey: ProductKey | string): boolean {
		return Boolean(
			access?.products.some(
				(product) => product.key === productKey && product.assigned && product.status === 'active'
			)
		);
	}

	can(access: AccessSnapshot | null | undefined, permission: Permission): boolean {
		return Boolean(access?.products.some((product) => product.permissions.includes(permission)));
	}

	isTenantOwner(access: AccessSnapshot | null | undefined): boolean {
		return access?.activeTenant?.role === 'owner';
	}

	hasTenantRole(
		access: AccessSnapshot | null | undefined,
		roles: TenantRole | TenantRole[]
	): boolean {
		const current = access?.activeTenant?.role;
		if (!current || !isTenantRole(current)) return false;
		const allowed = Array.isArray(roles) ? roles : [roles];
		return current === 'owner' || allowed.includes(current);
	}

	roleAtLeast(role: TenantRole, minimum: TenantRole): boolean;
	roleAtLeast(role: ProductRole, minimum: ProductRole): boolean;
	roleAtLeast(role: string, minimum: string): boolean;
	roleAtLeast(role: string, minimum: string): boolean {
		if (isTenantRole(role) && isTenantRole(minimum)) {
			return tenantRank[role] >= tenantRank[minimum];
		}
		if (isProductRole(role) && isProductRole(minimum)) {
			return productRank[role] >= productRank[minimum];
		}
		return false;
	}
}

export const defaultAccessPolicy = new AccessPolicy();
