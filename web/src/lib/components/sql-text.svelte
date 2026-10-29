<script lang="ts">
	let { sql, class: className = '' }: { sql: string; class?: string } = $props();

	const keyword =
		/\b(SELECT|FROM|WHERE|JOIN|ON|AND|OR|UPDATE|SET|INSERT|INTO|VALUES|DELETE|WITH|AS|GROUP|BY|ORDER|LIMIT|DESC|ASC|ANY|COUNT|SUM|NOW|INTERVAL|ALTER|TABLE|ADD|COLUMN|CREATE|INDEX|CONCURRENTLY|CONSTRAINT|FOREIGN|KEY|REFERENCES|NOT|NULL|LIKE|INCLUDING|ALL|PARTITION|RANGE|TYPE|ENUM|DROP|EXPLAIN|ANALYZE|BUFFERS|BEGIN|COMMIT|ROLLBACK|LOWER|IS|IN|CASE|WHEN|THEN|ELSE|END|DISTINCT|HAVING|LEFT|INNER|OUTER|RETURNING)\b/gi;

	const parts = $derived(sql.split(/(--[^\n]*|'[^']*'|\$\d+|\b\d+(?:\.\d+)?\b)/g));

	function words(part: string): { text: string; keyword: boolean }[] {
		return part.split(keyword).map((text, index) => ({ text, keyword: index % 2 === 1 }));
	}
</script>

<code class="font-mono {className}">
	{#each parts as part, index (index)}
		{#if part.startsWith('--')}
			<span class="text-tl-fg3 italic">{part}</span>
		{:else if part.startsWith("'")}
			<span style:color="var(--c2)">{part}</span>
		{:else if part.startsWith('$')}
			<span style:color="var(--c4)">{part}</span>
		{:else if /^\d/.test(part)}
			<span style:color="var(--c3)">{part}</span>
		{:else}
			{#each words(part) as word, wordIndex (`${index}-${wordIndex}`)}
				{#if word.keyword}
					<span class="font-medium text-tl-accent">{word.text}</span>
				{:else}
					{word.text}
				{/if}
			{/each}
		{/if}
	{/each}
</code>
