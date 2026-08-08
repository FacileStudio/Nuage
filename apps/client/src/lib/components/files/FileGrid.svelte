<script lang="ts">
	import type { Folder } from '$lib/backend';
	import type { BrowserEntry } from '$lib/files/entries';
	import type { DragMove } from '$lib/files/dnd.svelte';
	import type { Selection } from '$lib/files/selection.svelte';
	import { formatSize } from '$lib/format';
	import SelectToggle from './SelectToggle.svelte';

	let {
		entries,
		folders,
		selection,
		dnd,
		thumbnailUrl,
		renamingKey = null,
		renameValue = $bindable(''),
		onOpen,
		onSelect,
		onMenu,
		onDrop,
		onRenameKeydown,
		onRenameCancel
	}: {
		entries: BrowserEntry[];
		folders: Folder[];
		selection: Selection;
		dnd: DragMove;
		thumbnailUrl: (entry: BrowserEntry) => string;
		renamingKey?: string | null;
		renameValue?: string;
		onOpen: (entry: BrowserEntry) => void;
		onSelect: (entry: BrowserEntry, index: number, e: MouseEvent) => void;
		onMenu: (e: MouseEvent, entry: BrowserEntry) => void;
		onDrop: (target: number) => void;
		onRenameKeydown: (e: KeyboardEvent) => void;
		onRenameCancel: () => void;
	} = $props();

	const key = (entry: BrowserEntry) => `${entry.type}:${entry.id}`;
	const isDragged = (entry: BrowserEntry) =>
		dnd.item?.type === entry.type && dnd.item?.id === entry.id;
</script>

<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-5">
	{#each entries as entry, i (key(entry))}
		{@const selected = selection.has(entry.type, entry.id)}
		{@const dropTarget = entry.type === 'folder' && dnd.target === entry.id}
		<!-- The tile is a drop surface, not a widget: everything you can *do* to the entry is on
		     the button inside it or in the context menu. Drag-to-move has no keyboard path here
		     and never did; that is a gap to close with a Move command, not with a tabindex on a
		     div that would then be a focus stop announcing nothing. -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="relative rounded-fc-md transition-colors {dropTarget
				? 'bg-fc-accent/10 outline-2 outline-fc-accent/50'
				: selected
					? 'bg-fc-surface'
					: 'bg-fc-component hover:bg-fc-surface'} {isDragged(entry) ? 'opacity-40' : ''}"
			draggable={!selection.mode}
			ondragstart={(e) => dnd.start(e, entry.type, entry.id, entry.name)}
			ondragend={() => dnd.end()}
			ondragover={(e) => {
				if (entry.type !== 'folder' || !dnd.canDropOn(entry.id, folders)) return;
				e.preventDefault();
				e.dataTransfer!.dropEffect = 'move';
			}}
			ondragenter={(e) => {
				if (entry.type !== 'folder' || !dnd.canDropOn(entry.id, folders)) return;
				e.preventDefault();
				dnd.enter(key(entry), entry.id);
			}}
			ondragleave={() => entry.type === 'folder' && dnd.leave(key(entry), entry.id)}
			ondrop={(e) => {
				if (entry.type !== 'folder' || !dnd.canDropOn(entry.id, folders)) return;
				e.preventDefault();
				e.stopPropagation();
				onDrop(entry.id);
			}}
			oncontextmenu={(e) => onMenu(e, entry)}
		>
			{#if selection.mode}
				<div class="absolute top-2 left-2 z-10">
					<SelectToggle
						checked={selected}
						label="Select {entry.name}"
						onToggle={(e) => {
							e.stopPropagation();
							onSelect(entry, i, e);
						}}
					/>
				</div>
			{/if}

			<button
				type="button"
				class="flex w-full flex-col items-center gap-2 rounded-fc-md p-4 text-center focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-fc-ring"
				onclick={(e) => (selection.mode ? onSelect(entry, i, e) : onOpen(entry))}
			>
				{#if entry.isImage}
					<img
						src={thumbnailUrl(entry)}
						alt=""
						loading="lazy"
						class="aspect-[4/3] w-full rounded-fc-sm bg-fc-surface object-cover"
					/>
				{:else}
					<span
						class="flex aspect-[4/3] w-full items-center justify-center rounded-fc-sm bg-fc-surface text-fc-fg-muted"
					>
						<iconify-icon icon={entry.icon} width="32" height="32" class="block"></iconify-icon>
					</span>
				{/if}

				{#if renamingKey === key(entry)}
					<!-- svelte-ignore a11y_autofocus -->
					<input
						bind:value={renameValue}
						onkeydown={onRenameKeydown}
						onblur={onRenameCancel}
						onclick={(e) => e.stopPropagation()}
						aria-label="New name for {entry.name}"
						autofocus
						class="h-8 w-full rounded-fc-sm border border-fc-border bg-fc-bg px-2 text-center text-fc-xs text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
					/>
				{:else}
					<span class="w-full truncate text-fc-xs font-medium text-fc-fg">{entry.name}</span>
					<span class="text-fc-xs text-fc-fg-muted">
						{entry.size === null ? 'Folder' : formatSize(entry.size)}
					</span>
				{/if}
			</button>
		</div>
	{/each}
</div>
