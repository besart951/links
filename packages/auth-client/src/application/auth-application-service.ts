import type { AccessSnapshot } from '@links/access-core';
import type {
	AcceptInviteInput,
	AuthListener,
	AuthResult,
	AuthState,
	LoginInput,
	RegisterInput
} from '../domain/auth.js';

export type AuthRepository = {
	register(input: RegisterInput): Promise<AuthResult>;
	login(input: LoginInput): Promise<AuthResult>;
	acceptInvite(input: AcceptInviteInput): Promise<AuthResult>;
	logout(): Promise<void>;
	getSession(): Promise<AuthState>;
	switchTenant(tenantId: string): Promise<AccessSnapshot | undefined>;
};

export class AuthApplicationService {
	private listeners = new Set<AuthListener>();
	private state: AuthState = Object.freeze({ authenticated: false });

	constructor(private readonly authRepository: AuthRepository) {}

	get current(): AuthState {
		return this.state;
	}

	subscribe(listener: AuthListener): () => void {
		this.listeners.add(listener);
		listener(this.state);
		return () => this.listeners.delete(listener);
	}

	async register(input: RegisterInput): Promise<AuthState> {
		const response = await this.authRepository.register(input);
		this.setAuthenticated(response);
		return this.state;
	}

	async login(input: LoginInput): Promise<AuthState> {
		const response = await this.authRepository.login(input);
		this.setAuthenticated(response);
		return this.state;
	}

	async acceptInvite(input: AcceptInviteInput): Promise<AuthState> {
		const response = await this.authRepository.acceptInvite(input);
		this.setAuthenticated(response);
		return this.state;
	}

	async logout(): Promise<void> {
		await this.authRepository.logout();
		this.setState({ authenticated: false });
	}

	async refreshSession(): Promise<AuthState> {
		this.setState(await this.authRepository.getSession());
		return this.state;
	}

	async switchTenant(tenantId: string): Promise<AuthState> {
		const access = await this.authRepository.switchTenant(tenantId);
		this.setState({ ...this.state, access });
		return this.state;
	}

	private setAuthenticated(response: AuthResult): void {
		this.setState({
			authenticated: Boolean(response.session && response.access),
			session: response.session,
			access: response.access
		});
	}

	private setState(state: AuthState): void {
		this.state = Object.freeze({ ...state });
		for (const listener of this.listeners) {
			listener(this.state);
		}
	}
}
