<script lang="ts">
	import { can as canAccess, hasProduct, type AccessSnapshot, type ProductKey } from '@links/access-core';

	type Props = {
		access?: AccessSnapshot | null;
		can?: string;
		product?: ProductKey | string;
		children?: import('svelte').Snippet;
		fallback?: import('svelte').Snippet;
	};

	let { access = null, can = undefined, product = undefined, children, fallback }: Props = $props();

	let allowed = $derived(
		Boolean(access) &&
			(!can || canAccess(access, can)) &&
			(!product || hasProduct(access, product))
	);
</script>

{#if allowed}
	{@render children?.()}
{:else}
	{@render fallback?.()}
{/if}
