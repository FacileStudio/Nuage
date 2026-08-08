<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		actions
	}: {
		title: string;
		description?: string;
		actions?: Snippet;
	} = $props();
</script>

<!-- Title and description are one `gap-1` block, not two siblings of the outer gap: as
     siblings the description floats between heading and actions and belongs to neither
     (CHARTE §4). Same anatomy as muse's own `SettingsSection`. -->
<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
	<div class="flex min-w-0 flex-col gap-1">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">{title}</h1>
		{#if description}
			<p class="text-fc-sm text-fc-fg-muted">{description}</p>
		{/if}
	</div>
	{#if actions}
		<div class="flex shrink-0 flex-wrap items-center gap-2">{@render actions()}</div>
	{/if}
</div>
