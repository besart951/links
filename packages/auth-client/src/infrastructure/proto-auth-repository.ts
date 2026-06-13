import { createLinksClients, type LinksClientOptions } from '@links/rpc-client';
import type {
	AcceptInviteInput,
	AuthResult,
	AuthState,
	LoginInput,
	RegisterInput
} from '../domain/auth.js';
import { accessSnapshotProtoMapper, type AccessSnapshotProtoMapper } from '../mappers/access-snapshot-mapper.js';
import type { AuthRepository } from '../application/auth-application-service.js';

export class ProtoAuthRepository implements AuthRepository {
	private readonly clients: ReturnType<typeof createLinksClients>;

	constructor(
		baseUrl: string,
		options?: LinksClientOptions,
		private readonly mapper: AccessSnapshotProtoMapper = accessSnapshotProtoMapper
	) {
		this.clients = createLinksClients(baseUrl, options);
	}

	async register(input: RegisterInput): Promise<AuthResult> {
		const response = await this.clients.auth.register(input);
		return this.toAuthResult(response);
	}

	async login(input: LoginInput): Promise<AuthResult> {
		const response = await this.clients.auth.login(input);
		return this.toAuthResult(response);
	}

	async acceptInvite(input: AcceptInviteInput): Promise<AuthResult> {
		const response = await this.clients.memberships.acceptTenantInvite(input);
		return this.toAuthResult(response);
	}

	async logout(): Promise<void> {
		await this.clients.auth.logout({});
	}

	async getSession(): Promise<AuthState> {
		const response = await this.clients.auth.getSession({});
		return Object.freeze({
			authenticated: response.authenticated,
			session: this.mapper.toSession(response.session),
			access: this.mapper.toAccessSnapshot(response.access)
		});
	}

	async switchTenant(tenantId: string) {
		const response = await this.clients.tenants.switchTenant({ tenantId });
		return this.mapper.toAccessSnapshot(response.access);
	}

	linksClients(): ReturnType<typeof createLinksClients> {
		return this.clients;
	}

	private toAuthResult(response: {
		session?: Parameters<AccessSnapshotProtoMapper['toSession']>[0];
		access?: Parameters<AccessSnapshotProtoMapper['toAccessSnapshot']>[0];
	}): AuthResult {
		return Object.freeze({
			session: this.mapper.toSession(response.session),
			access: this.mapper.toAccessSnapshot(response.access)
		});
	}
}
