import type { AccessSnapshot } from '@links/access-core';

export type Session = Readonly<{
	id: string;
	userId: string;
	activeTenantId: string;
	expiresAt: string;
}>;

export type AuthState = Readonly<{
	authenticated: boolean;
	session?: Session;
	access?: AccessSnapshot;
}>;

export type AuthFailure = Readonly<{
	code: string;
	message: string;
}>;

export type AuthListener = (state: AuthState) => void;

export type RegisterInput = Readonly<{
	email: string;
	password: string;
	displayName: string;
	tenantName: string;
}>;

export type LoginInput = Readonly<{
	email: string;
	password: string;
}>;

export type AcceptInviteInput = Readonly<{
	inviteToken: string;
	displayName: string;
	password: string;
}>;

export type AuthResult = Readonly<{
	session?: Session;
	access?: AccessSnapshot;
}>;
