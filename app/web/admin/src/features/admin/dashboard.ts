import type { AuthFailure, AuthState } from '@links/auth-client';
import type { ProductAssignment, ProductLicense, TenantMember } from '@links/rpc-client';
import { createLinksAuthStore } from '@links/ui-svelte/auth-store';

export type AdminDashboardData = Readonly<{
	members: TenantMember[];
	licenses: ProductLicense[];
	assignments: ProductAssignment[];
	seatsByProduct: Record<string, number>;
}>;

export type RegisterForm = {
	email: string;
	password: string;
	displayName: string;
	tenantName: string;
};

export type LoginForm = {
	email: string;
	password: string;
};

export type AcceptInviteForm = {
	inviteToken: string;
	displayName: string;
	password: string;
};

export type InviteForm = {
	email: string;
	displayName: string;
	tenantRole: string;
};

export type AssignmentForm = {
	userId: string;
	productKey: string;
	role: string;
};

export const productOptions = [
	{ key: 'planner_link', label: 'PlannerLink' },
	{ key: 'finance_link', label: 'FinanceLink' },
	{ key: 'infra_link', label: 'InfraLink' },
	{ key: 'loka_link', label: 'LokaLink' }
] as const;

export function emptyAdminDashboardData(): AdminDashboardData {
	return Object.freeze({
		members: [],
		licenses: [],
		assignments: [],
		seatsByProduct: {}
	});
}

export class AdminDashboardController {
	private readonly auth: ReturnType<typeof createLinksAuthStore>;
	private readonly clients: ReturnType<ReturnType<typeof createLinksAuthStore>['client']['linksClients']>;

	constructor(
		private readonly baseUrl: string,
		private readonly onAuthState: (state: AuthState) => void,
		private readonly onAuthError: (message: string | null) => void
	) {
		this.auth = createLinksAuthStore(this.baseUrl);
		this.clients = this.auth.client.linksClients();
	}

	get currentAuthState(): AuthState {
		return this.auth.client.current;
	}

	mount(): () => void {
		const unsubscribeState = this.auth.state.subscribe(this.onAuthState);
		const unsubscribeError = this.auth.error.subscribe((error: AuthFailure | null) => {
			this.onAuthError(error ? `${error.code}: ${error.message}` : null);
		});
		return () => {
			unsubscribeState();
			unsubscribeError();
		};
	}

	async boot(): Promise<AdminDashboardData> {
		await this.auth.refreshSession();
		if (!this.auth.client.current.authenticated) {
			return emptyAdminDashboardData();
		}
		return this.refreshAdminData();
	}

	async register(input: RegisterForm): Promise<AdminDashboardData> {
		await this.auth.register(input);
		return this.refreshAdminData();
	}

	async login(input: LoginForm): Promise<AdminDashboardData> {
		await this.auth.login(input);
		return this.refreshAdminData();
	}

	async acceptInvite(input: AcceptInviteForm): Promise<AdminDashboardData> {
		await this.auth.client.acceptInvite(input);
		return this.refreshAdminData();
	}

	async logout(): Promise<AdminDashboardData> {
		await this.auth.logout();
		return emptyAdminDashboardData();
	}

	async refreshAdminData(): Promise<AdminDashboardData> {
		const [licenseResponse, memberResponse, assignmentResponse] = await Promise.all([
			this.clients.licensing.getTenantLicenses({}),
			this.clients.memberships.listTenantMembers({}),
			this.clients.assignments.listProductAssignments({})
		]);
		return Object.freeze({
			licenses: licenseResponse.licenses,
			members: memberResponse.members,
			assignments: assignmentResponse.assignments,
			seatsByProduct: Object.fromEntries(
				licenseResponse.licenses.map((license) => [license.productKey, license.seatsTotal])
			)
		});
	}

	async saveSeats(productKey: string, seatsTotal: number): Promise<AdminDashboardData> {
		await this.clients.licensing.setMyTenantProductSeats({ productKey, seatsTotal });
		await this.auth.refreshSession();
		return this.refreshAdminData();
	}

	async inviteMember(input: InviteForm): Promise<Readonly<{ data: AdminDashboardData; inviteToken: string }>> {
		const response = await this.clients.memberships.inviteTenantMember(input);
		return Object.freeze({
			data: await this.refreshAdminData(),
			inviteToken: response.inviteToken
		});
	}

	async assignProduct(input: AssignmentForm): Promise<AdminDashboardData> {
		await this.clients.assignments.assignUserToProduct(input);
		await this.auth.refreshSession();
		return this.refreshAdminData();
	}

	async removeAssignment(userId: string, productKey: string): Promise<AdminDashboardData> {
		await this.clients.assignments.removeUserFromProduct({ userId, productKey });
		await this.auth.refreshSession();
		return this.refreshAdminData();
	}

	async switchTenant(tenantId: string): Promise<AdminDashboardData> {
		await this.auth.switchTenant(tenantId);
		return this.refreshAdminData();
	}
}
