import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { BackendService } from './gen/proto/links/backend/v1/backend_pb.js';

export * from './gen/proto/links/backend/v1/backend_pb.js';

export function createBackendClient(baseUrl: string) {
	const transport = createConnectTransport({ baseUrl });
	return createClient(BackendService, transport);
}
