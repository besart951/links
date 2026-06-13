import type { LinksClientOptions } from '@links/rpc-client';
import { AuthApplicationService } from './application/auth-application-service.js';
import { ProtoAuthRepository } from './infrastructure/proto-auth-repository.js';

export class LinksAuthClient extends AuthApplicationService {
	private readonly protoRepository: ProtoAuthRepository;

	constructor(baseUrl: string, options?: LinksClientOptions) {
		const repository = new ProtoAuthRepository(baseUrl, options);
		super(repository);
		this.protoRepository = repository;
	}

	linksClients() {
		return this.protoRepository.linksClients();
	}
}

export function createAuthClient(baseUrl: string, options?: LinksClientOptions) {
	return new LinksAuthClient(baseUrl, options);
}

export * from './application/auth-application-service.js';
export * from './domain/auth.js';
export * from './infrastructure/proto-auth-repository.js';
export * from './mappers/access-snapshot-mapper.js';
export * from './mappers/auth-error-mapper.js';
