<script lang="ts">
	import type { DragMove, DropTarget } from '$lib/files/dnd.svelte';
	import { icons, nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	let {
		crumbs,
		dnd,
		onNavigate,
		onDrop
	}: {
		crumbs: { id: number | null; name: string }[];
		dnd: DragMove;
		onNavigate: (id: number | null) => void;
		onDrop: (id: number | null) => void;
	} = $props();

	/* The root has no folder id, so it needs a sentinel to be distinguishable from
	   "nothing is being hovered". */
	const asTarget = (id: number | null): DropTarget => (id === null ? 'root' : id);

	const isDropping = (id: number | null) => dnd.item !== null && dnd.target === asTarget(id);
	const nameOf = (crumb: { id: number | null; name: string }) =>
		crumb.id === null ? t('nav.files') : crumb.name;

	/*
	 * Dropping onto the crumb you are already in is a no-op, and highlighting it promises a
	 * move that will not happen. Every other crumb is an *ancestor* of the current folder, so
	 * moving into one is always legal — the descendant check the grid and table need does not
	 * apply here, because nothing in the trail can be below the thing being dragged.
	 */
	const accepts = (id: number | null, isLast: boolean) =>
		dnd.item !== null && !isLast && !(dnd.item.type === 'folder' && dnd.item.id === id);

	function dragProps(id: number | null, isLast: boolean, slot = 'crumb') {
		const key = `${slot}:${id ?? 'root'}`;
		return {
			ondragover: (e: DragEvent) => {
				if (!accepts(id, isLast)) return;
				e.preventDefault();
				e.dataTransfer!.dropEffect = 'move';
			},
			ondragenter: (e: DragEvent) => {
				if (!accepts(id, isLast)) return;
				e.preventDefault();
				dnd.enter(key, asTarget(id));
			},
			ondragleave: () => dnd.leave(key, asTarget(id)),
			ondrop: (e: DragEvent) => {
				if (!accepts(id, isLast)) return;
				e.preventDefault();
				e.stopPropagation();
				onDrop(id);
			}
		};
	}
</script>

{#snippet up(parent: { id: number | null; name: string })}
	<button
		type="button"
		aria-label={t('files.upTo', { name: nameOf(parent) })}
		title={t('files.upToDrop', { name: nameOf(parent) })}
		onclick={() => onNavigate(parent.id)}
		class="mr-1 inline-flex size-9 [@media(pointer:coarse)]:size-11 shrink-0 items-center justify-center rounded-fc-pill transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring {isDropping(
			parent.id
		)
			? 'bg-fc-accent text-fc-accent-fg'
			: dnd.item
				? 'bg-fc-accent/15 text-fc-fg'
				: 'bg-fc-surface text-fc-fg-muted hover:text-fc-fg'}"
		{...dragProps(parent.id, false, 'up')}
	>
		<iconify-icon icon={nuage.parent} width="18" height="18" class="block"></iconify-icon>
	</button>
{/snippet}

<nav class="flex min-w-0 flex-wrap items-center gap-1 text-fc-sm" aria-label={t('files.breadcrumb')}>
	{#if crumbs.length > 1}
		{@render up(crumbs[crumbs.length - 2])}
	{/if}
	{#each crumbs as crumb, i (crumb.id ?? 'root')}
		{@const isLast = i === crumbs.length - 1}
		{#if i > 0}
			<iconify-icon
				icon={icons.arrow}
				width="14"
				height="14"
				class="block shrink-0 text-fc-fg-muted"
			></iconify-icon>
		{/if}
		{#if isLast}
			<span
				class="truncate rounded-fc-sm px-1.5 py-0.5 font-medium text-fc-fg"
				aria-current="page"
				{...dragProps(crumb.id, true)}
			>
				{nameOf(crumb)}
			</span>
		{:else}
			<button
				type="button"
				onclick={() => onNavigate(crumb.id)}
				class="truncate rounded-fc-sm px-1.5 py-0.5 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring {isDropping(
					crumb.id
				)
					? 'bg-fc-accent text-fc-accent-fg'
					: 'text-fc-fg-muted hover:text-fc-fg'}"
				{...dragProps(crumb.id, false)}
			>
				{nameOf(crumb)}
			</button>
		{/if}
	{/each}
</nav>
