<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import logo from '#lib/assets/tusk-arc-icon-dark.svg';
	import { ENV_LABEL, envRailClass, type EnvKind } from '#lib/env.js';

	const kinds: EnvKind[] = ['local', 'dev', 'staging', 'prod'];
	let health = $state<'unknown' | 'ok' | 'down'>('unknown');

	async function check() {
		try {
			const r = await fetch('/api/v1/health');
			health = r.ok ? 'ok' : 'down';
		} catch {
			health = 'down';
		}
	}
</script>

<svelte:head><title>Tuskline</title></svelte:head>

<main
	class="mx-auto flex min-h-screen max-w-xl flex-col items-center justify-center gap-6 p-8 text-center"
>
	<img src={logo} alt="Tuskline" class="size-16" />
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Tuskline</h1>
		<p class="mt-1 text-muted-foreground">
			Pre-alpha scaffold. Port the screens from <code class="font-mono">prototype/</code> (see docs/design.md).
		</p>
	</div>
	<div class="flex gap-2">
		{#each kinds as k (k)}
			<span class="flex items-center gap-1.5 text-sm"
				><span class="h-3 w-1 rounded {envRailClass(k)}"></span>{ENV_LABEL[k]}</span
			>
		{/each}
	</div>
	<div class="flex items-center gap-3">
		<Button onclick={check}>Check API health</Button>
		<Badge variant={health === 'ok' ? 'default' : 'secondary'}>API: {health}</Badge>
	</div>
</main>
