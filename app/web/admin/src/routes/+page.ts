import type { PageLoad } from './$types';
import { createBackendClient } from '@links/rpc-client';

export const load: PageLoad = async () => {
	const baseUrl = (import.meta.env.PUBLIC_API_URL || 'http://localhost:4000').replace(/\/+$/, '');
	const client = createBackendClient(baseUrl);

	try {
		const health = await client.getHealth({});
		return {
			health: {
				status: health.status,
				db: health.db
			}
		};
	} catch {
		return {
			health: {
				status: 'error',
				db: 'unknown'
			}
		};
	}
};
