import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import {
	AssignmentService,
	AuthService,
	BackendService,
	DemoService,
	EntitlementService,
	LicensingService,
	MembershipService,
	TenantService
} from './gen/proto/links/backend/v1/backend_pb.js';

export * from './gen/proto/links/backend/v1/backend_pb.js';

type TransportFetch = typeof fetch;

export type LinksClientOptions = {
	fetch?: TransportFetch;
};

export function createLinksTransport(baseUrl: string, options: LinksClientOptions = {}) {
	const transportFetch: TransportFetch =
		options.fetch ??
		((input, init) =>
			fetch(input, {
				...init,
				credentials: 'include'
			}));
	return createConnectTransport({ baseUrl, fetch: transportFetch });
}

export function createBackendClient(baseUrl: string, options?: LinksClientOptions) {
	const transport = createLinksTransport(baseUrl, options);
	return createClient(BackendService, transport);
}

export function createLinksClients(baseUrl: string, options?: LinksClientOptions) {
	const transport = createLinksTransport(baseUrl, options);
	return {
		backend: createClient(BackendService, transport),
		auth: createClient(AuthService, transport),
		tenants: createClient(TenantService, transport),
		entitlements: createClient(EntitlementService, transport),
		memberships: createClient(MembershipService, transport),
		licensing: createClient(LicensingService, transport),
		assignments: createClient(AssignmentService, transport),
		demo: createClient(DemoService, transport)
	};
}
