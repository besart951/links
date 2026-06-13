import { createAuthClient, normalizeAuthError, type AuthFailure, type AuthState } from '@links/auth-client';
import { writable } from 'svelte/store';

export function createLinksAuthStore(baseUrl: string) {
	const client = createAuthClient(baseUrl);
	const state = writable<AuthState>({ authenticated: false });
	const error = writable<AuthFailure | null>(null);

	client.subscribe((next) => state.set(next));

	async function run<T>(operation: () => Promise<T>): Promise<T | undefined> {
		try {
			error.set(null);
			return await operation();
		} catch (cause) {
			error.set(normalizeAuthError(cause));
			return undefined;
		}
	}

	return {
		client,
		state,
		error,
		refreshSession: () => run(() => client.refreshSession()),
		register: (input: { email: string; password: string; displayName: string; tenantName: string }) =>
			run(() => client.register(input)),
		login: (input: { email: string; password: string }) => run(() => client.login(input)),
		logout: () => run(() => client.logout()),
		switchTenant: (tenantId: string) => run(() => client.switchTenant(tenantId))
	};
}
