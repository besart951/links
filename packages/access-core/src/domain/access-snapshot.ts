import type { ProductKey } from './product.js';
import type { ProductRole, TenantRole } from './roles.js';

export type Permission = string;

export type AccessUser = {
	id: string;
	email: string;
	displayName: string;
	platformRole: 'none' | 'support' | 'super_admin' | string;
};

export type ActiveTenant = {
	id: string;
	name: string;
	slug: string;
	role: TenantRole | string;
};

export type TenantSummary = ActiveTenant & {
	status: string;
};

export type ProductAccess = {
	key: ProductKey | string;
	name: string;
	status: string;
	assigned: boolean;
	role: ProductRole | string;
	permissions: Permission[];
	seatsTotal: number;
	seatsUsed: number;
};

export type AccessSnapshot = {
	user?: AccessUser;
	activeTenant?: ActiveTenant;
	tenants: TenantSummary[];
	products: ProductAccess[];
};
