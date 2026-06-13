<script lang="ts">
	import { onMount } from 'svelte';
	import { createLinksClients, type DemoFeatureResponse } from '@links/rpc-client';

	const baseUrl = (import.meta.env.PUBLIC_API_URL || 'http://localhost:4000').replace(/\/+$/, '');
	const clients = createLinksClients(baseUrl);
	let demo = $state<DemoFeatureResponse | null>(null);
	let error = $state('');

	onMount(async () => {
		try {
			demo = await clients.demo.getPlannerDemo({});
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'PlannerLink access denied';
		}
	});
</script>

<main class="demo-page">
	<a href="/">Back to admin</a>
	<section>
		<h1>{demo?.title ?? 'PlannerLink'}</h1>
		<p>{demo?.message ?? 'Loading PlannerLink access...'}</p>
		{#if error}
			<p class="error">{error}</p>
		{/if}
	</section>
</main>

<style>
	.demo-page {
		min-height: 100vh;
		background: #f5f7f6;
		color: #17201b;
		padding: 24px;
	}

	a {
		color: #22543d;
	}

	section {
		margin-top: 24px;
		max-width: 720px;
	}

	h1 {
		font-size: 32px;
		margin-bottom: 8px;
	}

	.error {
		margin-top: 16px;
		color: #8f2f16;
	}
</style>
