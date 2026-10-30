<script lang="ts">
	import { onMount } from 'svelte';
	import { sql, PostgreSQL } from '@codemirror/lang-sql';
	import { EditorState } from '@codemirror/state';
	import { EditorView, keymap } from '@codemirror/view';
	import { basicSetup } from 'codemirror';

	let {
		sql: initial = '',
		schema = {},
		minHeight = 180,
		onchange,
		onrun,
		onexplain
	}: {
		sql?: string;
		schema?: Record<string, string[]>;
		minHeight?: number;
		onchange?: (sql: string) => void;
		onrun?: () => void;
		onexplain?: () => void;
	} = $props();

	let host: HTMLDivElement | undefined = $state();
	let view: EditorView | undefined;

	const theme = EditorView.theme({
		'&': {
			backgroundColor: 'transparent',
			color: 'var(--tl-fg)',
			fontSize: '13px',
			height: '100%'
		},
		'.cm-scroller': { fontFamily: 'JetBrains Mono, ui-monospace, monospace' },
		'.cm-gutters': {
			backgroundColor: 'transparent',
			color: 'var(--tl-fg3)',
			border: 'none'
		},
		'.cm-activeLine, .cm-activeLineGutter': { backgroundColor: 'transparent' },
		'&.cm-focused': { outline: 'none' },
		'.cm-cursor': { borderLeftColor: 'var(--tl-accent)' }
	});

	onMount(() => {
		if (!host) return;
		view = new EditorView({
			parent: host,
			state: EditorState.create({
				doc: initial,
				extensions: [
					basicSetup,
					sql({ dialect: PostgreSQL, schema }),
					theme,
					keymap.of([
						{
							key: 'Shift-Mod-Enter',
							run: () => {
								onexplain?.();
								return true;
							}
						},
						{
							key: 'Mod-Enter',
							run: () => {
								onrun?.();
								return true;
							}
						}
					]),
					EditorView.updateListener.of((update) => {
						if (update.docChanged) onchange?.(update.state.doc.toString());
					})
				]
			})
		});
		return () => view?.destroy();
	});
</script>

<div bind:this={host} class="h-full" style:min-height="{minHeight}px"></div>
