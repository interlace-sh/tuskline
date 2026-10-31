<script lang="ts">
	import Binary from '@lucide/svelte/icons/binary';
	import Braces from '@lucide/svelte/icons/braces';
	import Calendar from '@lucide/svelte/icons/calendar';
	import Fingerprint from '@lucide/svelte/icons/fingerprint';
	import Hash from '@lucide/svelte/icons/hash';
	import ToggleLeft from '@lucide/svelte/icons/toggle-left';
	import TypeIcon from '@lucide/svelte/icons/type';
	import {
		fingerprint,
		readProfileCache,
		scheduleProfile,
		writeProfileCache
	} from '#lib/column-profile-cache.js';
	import type { ColumnProfile } from '#lib/column-profile.js';
	import { flyoutAnchor, mountFlyout } from '#lib/flyout.js';

	let {
		columns,
		rows,
		embedded = false
	}: {
		columns: { name: string; type: string }[];
		rows: unknown[][];
		/** Drop the top border when this sits inside another panel. */
		embedded?: boolean;
	} = $props();

	const key = $derived(fingerprint(columns, rows));
	let ready = $state<ColumnProfile[] | undefined>();
	let readyKey = $state('');
	const profiles = $derived(readProfileCache(key) ?? (readyKey === key ? ready : undefined));

	$effect(() => {
		const currentKey = key;
		if (readProfileCache(currentKey)) return;
		let cancelled = false;
		const task = scheduleProfile(columns, rows, (next) => {
			if (cancelled) return;
			writeProfileCache(currentKey, next);
			ready = next;
			readyKey = currentKey;
		});
		return () => {
			cancelled = true;
			task.cancel();
		};
	});

	let selected = $state('');
	let anchor = $state<{ top: number; left: number } | null>(null);
	const current = $derived(profiles?.find((profile) => profile.name === selected));

	function typeIcon(type: string) {
		const declared = type.toLowerCase();
		if (/bool/.test(declared)) return ToggleLeft;
		if (/json/.test(declared)) return Braces;
		if (/uuid/.test(declared)) return Fingerprint;
		if (/bytea|blob/.test(declared)) return Binary;
		if (/int|numeric|float|double|real|decimal|money|serial/.test(declared)) return Hash;
		if (/time|date/.test(declared)) return Calendar;
		return TypeIcon;
	}

	function toggle(profile: ColumnProfile, event: MouseEvent) {
		if (selected === profile.name) {
			selected = '';
			anchor = null;
			return;
		}
		const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
		selected = profile.name;
		anchor = flyoutAnchor(rect, window.innerWidth);
	}
</script>

<aside
	class="flex min-h-0 w-full flex-1 flex-col overflow-auto {embedded
		? ''
		: 'border-t border-tl-line'}"
>
	<div class="px-3 py-2 text-[11px] text-tl-fg3 tabular-nums">
		{rows.length} rows · {columns.length} columns
		{#if !profiles}<span> · summarising…</span>{/if}
	</div>
	{#each profiles ?? [] as profile (profile.name)}
		{@const Icon = typeIcon(profile.type)}
		<button
			class="flex h-7 w-full items-center gap-2 px-2 text-left hover:bg-tl-hover {current?.name ===
			profile.name
				? 'bg-tl-hover'
				: ''}"
			title="{profile.type} · {profile.distinct} distinct · {profile.nullPct}% null"
			aria-expanded={current?.name === profile.name}
			onclick={(event) => toggle(profile, event)}
		>
			<Icon size={12} class="shrink-0 text-tl-fg3" />
			<span class="min-w-0 flex-1 truncate font-mono text-[12px]">{profile.name}</span>
			<span class="shrink-0 text-[10.5px] text-tl-fg3 tabular-nums">{profile.distinct}</span>
			<span class="w-9 shrink-0 text-right text-[10.5px] text-tl-fg3 tabular-nums"
				>{profile.nullPct}%</span
			>
		</button>
	{/each}
</aside>

{#if current && anchor}
	<div
		class="fixed z-50 w-64 overflow-auto rounded-lg border border-tl-line bg-tl-elev p-3"
		use:mountFlyout={anchor}
		style:box-shadow="var(--tl-shadow)"
	>
		<div class="font-mono text-[12px]">{current.name}</div>
		<p class="mt-0.5 text-[10.5px] text-tl-fg3">
			{current.distinct} distinct · {current.nulls} null
		</p>
		{#if current.kind === 'category'}
			<ul class="mt-2 space-y-1.5">
				{#each current.values as value, index (`${index}:${value.label}`)}
					<li>
						<div class="flex items-center justify-between gap-2 text-[11px]">
							<span class="truncate font-mono {value.null ? 'text-tl-fg3 italic' : ''}"
								>{value.label}</span
							>
							<span class="shrink-0 text-tl-fg3 tabular-nums">{value.count} · {value.pct}%</span>
						</div>
						<div class="mt-0.5 h-1 overflow-hidden rounded-full bg-tl-subtle">
							<div
								class="h-full rounded-full {value.null ? 'bg-tl-line2' : 'bg-tl-accent'}"
								style:width="{value.pct}%"
							></div>
						</div>
					</li>
				{/each}
			</ul>
		{:else}
			<dl class="mt-2 space-y-1 text-[11px]">
				<div class="flex justify-between gap-2">
					<dt class="text-tl-fg3">{current.kind === 'time' ? 'earliest' : 'min'}</dt>
					<dd class="font-mono tabular-nums">{current.min ?? '—'}</dd>
				</div>
				<div class="flex justify-between gap-2">
					<dt class="text-tl-fg3">{current.kind === 'time' ? 'latest' : 'max'}</dt>
					<dd class="font-mono tabular-nums">{current.max ?? '—'}</dd>
				</div>
			</dl>
		{/if}
	</div>
{/if}
