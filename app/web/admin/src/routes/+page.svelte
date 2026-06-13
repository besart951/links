<script lang="ts">
	import { onMount } from 'svelte';
	import Access from '@links/ui-svelte/access';
	import type { ProductAssignment, ProductLicense, TenantMember } from '@links/rpc-client';
	import {
		AdminDashboardController,
		emptyAdminDashboardData,
		productOptions,
		type AcceptInviteForm,
		type AssignmentForm,
		type InviteForm,
		type LoginForm,
		type RegisterForm
	} from '../features/admin/dashboard';

	const baseUrl = (import.meta.env.PUBLIC_API_URL || 'http://localhost:4000').replace(/\/+$/, '');
	const dashboard = new AdminDashboardController(
		baseUrl,
		(state) => (authState = state),
		(message) => (authError = message)
	);

	let authState = $state(dashboard.currentAuthState);
	let authError = $state<string | null>(null);
	let busy = $state(false);
	let mode = $state<'login' | 'register' | 'invite'>('register');

	let registerForm = $state<RegisterForm>({
		email: '',
		password: '',
		displayName: '',
		tenantName: ''
	});
	let loginForm = $state<LoginForm>({ email: '', password: '' });
	let acceptInviteForm = $state<AcceptInviteForm>({ inviteToken: '', displayName: '', password: '' });

	let members = $state<TenantMember[]>([]);
	let licenses = $state<ProductLicense[]>([]);
	let assignments = $state<ProductAssignment[]>([]);
	let inviteForm = $state<InviteForm>({ email: '', displayName: '', tenantRole: 'member' });
	let latestInviteToken = $state('');
	let assignmentForm = $state<AssignmentForm>({ userId: '', productKey: 'planner_link', role: 'member' });
	let seatsByProduct = $state<Record<string, number>>({});

	onMount(() => {
		const unmount = dashboard.mount();
		void boot();
		return unmount;
	});

	async function boot() {
		applyAdminData(await dashboard.boot());
	}

	async function run(operation: () => Promise<void>) {
		busy = true;
		authError = null;
		try {
			await operation();
		} catch (error) {
			authError = error instanceof Error ? error.message : 'Unknown error';
		} finally {
			busy = false;
		}
	}

	async function register() {
		await run(async () => {
			applyAdminData(await dashboard.register(registerForm));
		});
	}

	async function login() {
		await run(async () => {
			applyAdminData(await dashboard.login(loginForm));
		});
	}

	async function acceptInvite() {
		await run(async () => {
			applyAdminData(await dashboard.acceptInvite(acceptInviteForm));
		});
	}

	async function logout() {
		await run(async () => {
			applyAdminData(await dashboard.logout());
		});
	}

	async function saveSeats(productKey: string) {
		await run(async () => {
			applyAdminData(await dashboard.saveSeats(productKey, Number(seatsByProduct[productKey] ?? 0)));
		});
	}

	async function inviteMember() {
		await run(async () => {
			const result = await dashboard.inviteMember(inviteForm);
			latestInviteToken = result.inviteToken;
			inviteForm = { email: '', displayName: '', tenantRole: 'member' };
			applyAdminData(result.data);
		});
	}

	async function assignProduct() {
		await run(async () => {
			applyAdminData(await dashboard.assignProduct(assignmentForm));
		});
	}

	async function removeAssignment(userId: string, productKey: string) {
		await run(async () => {
			applyAdminData(await dashboard.removeAssignment(userId, productKey));
		});
	}

	async function switchTenant(tenantId: string) {
		await run(async () => {
			applyAdminData(await dashboard.switchTenant(tenantId));
		});
	}

	function applyAdminData(data = emptyAdminDashboardData()) {
		members = data.members;
		licenses = data.licenses;
		assignments = data.assignments;
		seatsByProduct = data.seatsByProduct;
	}
</script>

<svelte:head>
	<title>Links Admin</title>
</svelte:head>

<main class="admin-shell">
	<header class="topbar">
		<div>
			<p class="eyebrow">CodeLinks Platform</p>
			<h1>Admin Panel</h1>
		</div>
		{#if authState.authenticated}
			<div class="account">
				<span>{authState.access?.user?.email}</span>
				<button type="button" class="ghost-button" onclick={logout} disabled={busy}>Logout</button>
			</div>
		{/if}
	</header>

	{#if authError}
		<p class="error">{authError}</p>
	{/if}

	{#if !authState.authenticated}
		<section class="auth-grid">
			<div class="panel auth-panel">
				<div class="segments">
					<button class:active={mode === 'register'} type="button" onclick={() => (mode = 'register')}>Register</button>
					<button class:active={mode === 'login'} type="button" onclick={() => (mode = 'login')}>Login</button>
					<button class:active={mode === 'invite'} type="button" onclick={() => (mode = 'invite')}>Accept invite</button>
				</div>

				{#if mode === 'register'}
					<form onsubmit={(event) => { event.preventDefault(); void register(); }}>
						<label>Email <input bind:value={registerForm.email} type="email" required /></label>
						<label>Password <input bind:value={registerForm.password} type="password" minlength="8" required /></label>
						<label>Name <input bind:value={registerForm.displayName} required /></label>
						<label>Tenant <input bind:value={registerForm.tenantName} required /></label>
						<button type="submit" disabled={busy}>Create tenant</button>
					</form>
				{:else if mode === 'login'}
					<form onsubmit={(event) => { event.preventDefault(); void login(); }}>
						<label>Email <input bind:value={loginForm.email} type="email" required /></label>
						<label>Password <input bind:value={loginForm.password} type="password" required /></label>
						<button type="submit" disabled={busy}>Login</button>
					</form>
				{:else}
					<form onsubmit={(event) => { event.preventDefault(); void acceptInvite(); }}>
						<label>Invite token <input bind:value={acceptInviteForm.inviteToken} required /></label>
						<label>Name <input bind:value={acceptInviteForm.displayName} required /></label>
						<label>Password <input bind:value={acceptInviteForm.password} type="password" minlength="8" required /></label>
						<button type="submit" disabled={busy}>Accept invite</button>
					</form>
				{/if}
			</div>
		</section>
	{:else}
		<section class="overview">
			<div class="panel">
				<p class="eyebrow">Active Tenant</p>
				<h2>{authState.access?.activeTenant?.name}</h2>
				<p>{authState.access?.activeTenant?.role}</p>
				<select
					aria-label="Tenant wechseln"
					value={authState.access?.activeTenant?.id}
					onchange={(event) => void switchTenant(event.currentTarget.value)}
				>
					{#each authState.access?.tenants ?? [] as tenant}
						<option value={tenant.id}>{tenant.name} · {tenant.role}</option>
					{/each}
				</select>
			</div>
			<div class="panel product-links">
				<p class="eyebrow">Product demos</p>
				<div class="link-row">
					<Access access={authState.access} product="planner_link">
						<a href="/planner">Open PlannerLink</a>
					</Access>
					<Access access={authState.access} product="finance_link">
						<a href="/finance">Open FinanceLink</a>
					</Access>
				</div>
			</div>
		</section>

		<section class="workspace">
			<div class="panel">
				<h2>Licenses</h2>
				<div class="table">
					{#each licenses as license}
						<div class="row">
							<span>{license.productName}</span>
							<span>{license.seatsUsed}/{license.seatsTotal} used</span>
							<input
								type="number"
								min="0"
								max="10"
								bind:value={seatsByProduct[license.productKey]}
								aria-label={`${license.productName} seats`}
							/>
							<button type="button" onclick={() => void saveSeats(license.productKey)} disabled={busy}>Save</button>
						</div>
					{/each}
				</div>
			</div>

			<div class="panel">
				<h2>Members</h2>
				<form class="inline-form" onsubmit={(event) => { event.preventDefault(); void inviteMember(); }}>
					<input bind:value={inviteForm.email} type="email" placeholder="email@company.com" required />
					<input bind:value={inviteForm.displayName} placeholder="Display name" />
					<select bind:value={inviteForm.tenantRole}>
						<option value="member">Member</option>
						<option value="admin">Admin</option>
						<option value="billing_admin">Billing admin</option>
						<option value="viewer">Viewer</option>
					</select>
					<button type="submit" disabled={busy}>Invite</button>
				</form>
				{#if latestInviteToken}
					<p class="token">Invite token: {latestInviteToken}</p>
				{/if}
				<div class="table">
					{#each members as member}
						<div class="row">
							<span>{member.email}</span>
							<span>{member.tenantRole}</span>
							<span>{member.status}</span>
						</div>
					{/each}
				</div>
			</div>

			<div class="panel">
				<h2>Product assignments</h2>
				<form class="inline-form" onsubmit={(event) => { event.preventDefault(); void assignProduct(); }}>
					<select bind:value={assignmentForm.userId} required>
						<option value="" disabled>User</option>
						{#each members as member}
							<option value={member.userId}>{member.email}</option>
						{/each}
					</select>
					<select bind:value={assignmentForm.productKey}>
						{#each productOptions as product}
							<option value={product.key}>{product.label}</option>
						{/each}
					</select>
					<select bind:value={assignmentForm.role}>
						<option value="viewer">Viewer</option>
						<option value="member">Member</option>
						<option value="admin">Admin</option>
					</select>
					<button type="submit" disabled={busy}>Assign</button>
				</form>
				<div class="table">
					{#each assignments as assignment}
						<div class="row">
							<span>{assignment.email}</span>
							<span>{assignment.productName}</span>
							<span>{assignment.role}</span>
							<span>{assignment.status}</span>
							<button
								type="button"
								class="ghost-button"
								onclick={() => void removeAssignment(assignment.userId, assignment.productKey)}
								disabled={busy || assignment.status !== 'active'}
							>
								Remove
							</button>
						</div>
					{/each}
				</div>
			</div>
		</section>
	{/if}
</main>

<style>
	.admin-shell {
		min-height: 100vh;
		background: #f7f7f4;
		color: #171717;
		padding: 24px;
	}

	.topbar,
	.overview,
	.workspace,
	.auth-grid {
		width: min(1180px, 100%);
		margin: 0 auto;
	}

	.topbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 24px;
	}

	h1,
	h2,
	p {
		margin: 0;
	}

	h1 {
		font-size: 30px;
	}

	h2 {
		font-size: 18px;
		margin-bottom: 14px;
	}

	.eyebrow {
		color: #626257;
		font-size: 12px;
		font-weight: 700;
		letter-spacing: 0;
		text-transform: uppercase;
	}

	.account,
	.link-row,
	.inline-form,
	.row {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.overview {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 16px;
		margin-bottom: 16px;
	}

	.workspace {
		display: grid;
		grid-template-columns: 1fr;
		gap: 16px;
	}

	.auth-grid {
		display: grid;
		place-items: start center;
	}

	.panel {
		background: #ffffff;
		border: 1px solid #d9d8cf;
		border-radius: 8px;
		padding: 18px;
		box-shadow: 0 1px 2px rgb(0 0 0 / 0.04);
	}

	.auth-panel {
		width: min(460px, 100%);
	}

	.segments {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 4px;
		margin-bottom: 16px;
		background: #ecebe3;
		padding: 4px;
		border-radius: 8px;
	}

	.segments button,
	button,
	a {
		border: 1px solid #22251f;
		border-radius: 6px;
		background: #22251f;
		color: #fff;
		font: inherit;
		font-size: 14px;
		padding: 8px 12px;
		text-decoration: none;
		cursor: pointer;
	}

	.segments button {
		background: transparent;
		color: #22251f;
		border-color: transparent;
	}

	.segments button.active {
		background: #fff;
		border-color: #d9d8cf;
	}

	button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.ghost-button {
		background: transparent;
		color: #22251f;
		border-color: #c8c6bb;
	}

	form {
		display: grid;
		gap: 12px;
	}

	label {
		display: grid;
		gap: 6px;
		font-size: 13px;
		color: #45453d;
	}

	input,
	select {
		min-width: 0;
		border: 1px solid #c8c6bb;
		border-radius: 6px;
		background: #fff;
		color: #171717;
		font: inherit;
		padding: 8px 10px;
	}

	.table {
		display: grid;
		gap: 8px;
	}

	.row {
		display: grid;
		grid-template-columns: minmax(140px, 1fr) repeat(4, minmax(88px, auto));
		padding: 10px 0;
		border-top: 1px solid #eeeeea;
	}

	.row input {
		width: 88px;
	}

	.inline-form {
		display: flex;
		flex-wrap: wrap;
		margin-bottom: 12px;
	}

	.error,
	.token {
		width: min(1180px, 100%);
		margin: 0 auto 16px;
		padding: 10px 12px;
		border-radius: 6px;
		background: #fff3f0;
		color: #8f2f16;
		border: 1px solid #f0c9bd;
	}

	.token {
		background: #f3f7ff;
		color: #184270;
		border-color: #bfd3f0;
		word-break: break-all;
	}

	@media (max-width: 760px) {
		.admin-shell {
			padding: 16px;
		}

		.topbar,
		.overview {
			grid-template-columns: 1fr;
			display: grid;
			gap: 12px;
		}

		.account,
		.inline-form {
			align-items: stretch;
			flex-direction: column;
		}

		.row {
			grid-template-columns: 1fr;
			gap: 6px;
		}
	}
</style>
