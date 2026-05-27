<script lang="ts">
	import { getUndo, undoLast, dismissUndo } from '$lib/undo.svelte';

	const undo = getUndo();
</script>

{#if undo.current}
	<div role="status" class="fixed bottom-6 left-1/2 z-50 -translate-x-1/2 flex items-center gap-3 rounded-lg border border-border bg-background px-4 py-2.5 shadow-xl animate-in fade-in slide-in-from-bottom-2 duration-200">
		<iconify-icon icon="solar:trash-bin-2-linear" width="16" class="text-muted-foreground shrink-0"></iconify-icon>
		<span class="text-sm">{undo.current.label}</span>
		<button
			class="ml-1 rounded-md bg-primary px-2.5 py-1 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
			onclick={undoLast}
		>
			Undo
		</button>
		<button
			class="flex h-5 w-5 items-center justify-center rounded-md transition-colors hover:bg-muted"
			onclick={dismissUndo}
			aria-label="Dismiss"
		>
			<iconify-icon icon="solar:close-circle-linear" width="14" class="text-muted-foreground"></iconify-icon>
		</button>
	</div>
{/if}
