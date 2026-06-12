<script lang="ts">
	let { data } = $props();
</script>

<h1>Admin</h1>
<p>Backend status: {data.health.status}</p>
<p>Database status: {data.health.db}</p>
