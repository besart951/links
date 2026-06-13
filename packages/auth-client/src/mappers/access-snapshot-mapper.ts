import type { AccessSnapshot } from '@links/access-core';
import type {
	AccessSnapshot as ProtoAccessSnapshot,
	AccessUser as ProtoAccessUser,
	ActiveTenant as ProtoActiveTenant,
	ProductAccess as ProtoProductAccess,
	Session as ProtoSession,
	TenantSummary as ProtoTenantSummary
} from '@links/rpc-client';
import type { Session } from '../domain/auth.js';

export class AccessSnapshotProtoMapper {
	toSession(proto: ProtoSession | undefined): Session | undefined {
		if (!proto) return undefined;
		return Object.freeze({
			id: proto.id,
			userId: proto.userId,
			activeTenantId: proto.activeTenantId,
			expiresAt: proto.expiresAt
		});
	}

	toAccessSnapshot(proto: ProtoAccessSnapshot | undefined): AccessSnapshot | undefined {
		if (!proto) return undefined;
		return Object.freeze({
			user: this.toUser(proto.user),
			activeTenant: this.toActiveTenant(proto.activeTenant),
			tenants: proto.tenants.map((tenant) => this.toTenantSummary(tenant)),
			products: proto.products.map((product) => this.toProductAccess(product))
		});
	}

	private toUser(proto: ProtoAccessUser | undefined) {
		if (!proto) return undefined;
		return Object.freeze({
			id: proto.id,
			email: proto.email,
			displayName: proto.displayName,
			platformRole: proto.platformRole
		});
	}

	private toActiveTenant(proto: ProtoActiveTenant | undefined) {
		if (!proto) return undefined;
		return Object.freeze({
			id: proto.id,
			name: proto.name,
			slug: proto.slug,
			role: proto.role
		});
	}

	private toTenantSummary(proto: ProtoTenantSummary) {
		return Object.freeze({
			id: proto.id,
			name: proto.name,
			slug: proto.slug,
			role: proto.role,
			status: proto.status
		});
	}

	private toProductAccess(proto: ProtoProductAccess) {
		return Object.freeze({
			key: proto.key,
			name: proto.name,
			status: proto.status,
			assigned: proto.assigned,
			role: proto.role,
			permissions: [...proto.permissions],
			seatsTotal: proto.seatsTotal,
			seatsUsed: proto.seatsUsed
		});
	}
}

export const accessSnapshotProtoMapper = new AccessSnapshotProtoMapper();
