import { Code, ConnectError } from '@connectrpc/connect';
import type { AuthFailure } from '../domain/auth.js';

export function normalizeAuthError(error: unknown): AuthFailure {
	if (error instanceof ConnectError) {
		return Object.freeze({
			code: Code[error.code] ?? 'UNKNOWN',
			message: error.rawMessage || error.message
		});
	}
	if (error instanceof Error) {
		return Object.freeze({ code: 'UNKNOWN', message: error.message });
	}
	return Object.freeze({ code: 'UNKNOWN', message: 'Unknown error' });
}
