<script lang="ts">
	let {
		open = $bindable(false),
		title,
		message,
		confirmLabel = 'Delete',
		confirmIcon = 'solar:trash-bin-2-linear',
		loading = false,
		onconfirm
	}: {
		open: boolean;
		title: string;
		message: string;
		confirmLabel?: string;
		confirmIcon?: string;
		loading?: boolean;
		onconfirm: () => void;
	} = $props();
</script>

{#if open}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" role="alertdialog">
		<button class="absolute inset-0" onclick={() => (open = false)} aria-label="Cancel"></button>
		<div class="relative z-10 w-full max-w-sm rounded-lg border border-border bg-background p-6 shadow-xl">
			<h3 class="text-lg font-semibold">{title}</h3>
			<p class="mt-2 text-sm text-muted-foreground">{message}</p>
			<div class="mt-5 flex justify-end gap-2">
				<button
					class="inline-flex h-9 items-center justify-center rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent"
					onclick={() => (open = false)}
					disabled={loading}
				>
					Cancel
				</button>
				<button
					class="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-red-600 px-4 text-sm font-medium text-white transition-colors hover:bg-red-700 disabled:opacity-50"
					onclick={onconfirm}
					disabled={loading}
				>
					{#if loading}
						<div class="h-3 w-3 animate-spin rounded-full border-2 border-white border-t-transparent"></div>
					{:else}
						<iconify-icon icon={confirmIcon} width="16"></iconify-icon>
					{/if}
					{confirmLabel}
				</button>
			</div>
		</div>
	</div>
{/if}
