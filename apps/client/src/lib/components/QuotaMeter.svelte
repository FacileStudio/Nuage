<script lang="ts">
	import type { QuotaResponse } from '$lib/backend';
	import { formatSize } from '$lib/format';
	import { nuage } from '$lib/icons';

	let {
		quota,
		collapsed = false
	}: {
		quota: QuotaResponse;
		collapsed?: boolean;
	} = $props();

	const hasLimit = $derived(quota.storage_limit > 0);
	const percent = $derived(Math.min(Math.max(quota.percentage, 0), 100));

	/*
	 * The bar is the one place in this app where chroma carries meaning rather than decoration,
	 * so it goes through the status tokens (CHARTE §2) — accent while there is room, warning as
	 * the ceiling comes into view, danger once it is a problem. It used to reach for stock
	 * Tailwind red and amber, which are fixed values that do not follow the theme into dark
	 * mode. (Named without the literal class strings on purpose: Tailwind's scanner reads
	 * comments too, and would emit the very utilities this file is documenting the removal of.)
	 */
	const fill = $derived(
		percent >= 90 ? 'bg-fc-danger' : percent >= 80 ? 'bg-fc-warning' : 'bg-fc-accent'
	);
</script>

<div
	class="flex shrink-0 flex-col gap-1.5 rounded-fc-md bg-fc-component p-3"
	role="group"
	aria-label="Storage usage"
>
	{#if collapsed}
		<!-- The rail is 44px of content wide here, so the number is all that fits — and the
		     number is the part you glance at anyway. -->
		<span class="text-center text-fc-xs font-medium text-fc-fg-muted">
			{hasLimit ? `${Math.round(percent)}%` : formatSize(quota.storage_used)}
		</span>
	{:else if hasLimit}
		<div class="flex items-center justify-between gap-2 text-fc-xs text-fc-fg-muted">
			<span class="truncate">{formatSize(quota.storage_used)} of {formatSize(quota.storage_limit)}</span>
			<span class="shrink-0 tabular-nums">{Math.round(percent)}%</span>
		</div>
		<div
			class="h-1.5 w-full overflow-hidden rounded-fc-pill bg-fc-surface"
			role="progressbar"
			aria-valuenow={Math.round(percent)}
			aria-valuemin={0}
			aria-valuemax={100}
			aria-label="Storage used"
		>
			<div
				class="h-full rounded-fc-pill transition-[width] duration-300 ease-[var(--ease-fc)] {fill}"
				style="width: {percent}%"
			></div>
		</div>
	{:else}
		<div class="flex items-center gap-1.5 text-fc-xs text-fc-fg-muted">
			<iconify-icon icon={nuage.brand} width="14" height="14" class="block shrink-0"></iconify-icon>
			<span class="truncate">{formatSize(quota.storage_used)} used</span>
		</div>
	{/if}
</div>
