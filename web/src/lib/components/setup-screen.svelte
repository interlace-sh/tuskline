<script lang="ts">
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import Unplug from '@lucide/svelte/icons/unplug';
	import { shell } from '#lib/shell.svelte.js';

	const btn =
		'inline-flex h-8 items-center gap-1.5 rounded-md border border-transparent bg-brand px-2.5 text-[12.5px] font-medium text-white';
</script>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-4">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Setup</h1>
		<p class="mt-1 max-w-xl text-sm text-tl-fg2">
			Connect a database you already run, or review its configuration. Deploying a new cluster is a
			later milestone.
		</p>
	</div>
	<div class="grid gap-3 sm:grid-cols-2">
		<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
			<h2 class="flex items-center gap-2 text-[14px] font-semibold">
				<Unplug size={16} class="text-brand" /> Add connection
			</h2>
			<p class="mt-1 text-[12.5px] leading-relaxed text-tl-fg2">
				One form. Create probes the server and saves it. A typed password is not stored.
			</p>
			<button class="{btn} mt-3" onclick={() => shell.go('connect')}>Add connection</button>
		</section>
		<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
			<h2 class="flex items-center gap-2 text-[14px] font-semibold">
				<SlidersHorizontal size={16} class="text-brand" /> Config advisor
			</h2>
			<p class="mt-1 text-[12.5px] leading-relaxed text-tl-fg2">
				A preview of settings to change, with the reason and whether each one needs a reload or a
				restart.
			</p>
			<button class="{btn} mt-3" onclick={() => shell.go('advisor')}>Open config advisor</button>
		</section>
	</div>
</div>
