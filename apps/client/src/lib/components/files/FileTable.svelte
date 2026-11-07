<script lang="ts">
	import { Table } from '@facile/muse';
	import type { BrowserEntry } from '$lib/files/entries';
	import type { DragMove } from '$lib/files/dnd.svelte';
	import type { Selection } from '$lib/files/selection.svelte';
	import type { Folder } from '$lib/backend';
	import { formatDate, formatSize } from '$lib/format';
	import { nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';
	import SelectToggle from './SelectToggle.svelte';

	let {
		entries,
		folders,
		selection,
		dnd,
		renamingKey = null,
		renameValue = $bindable(''),
		onOpen,
		onSelect,
		onMenu,
		onDragStart,
		onDrop,
		onRenameKeydown,
		onRenameCancel
	}: {
		entries: BrowserEntry[];
		folders: Folder[];
		selection: Selection;
		dnd: DragMove;
		renamingKey?: string | null;
		renameValue?: string;
		onOpen: (entry: BrowserEntry) => void;
		onSelect: (entry: BrowserEntry, index: number, e: MouseEvent) => void;
		onMenu: (e: MouseEvent, entry: BrowserEntry) => void;
		onDragStart: (e: DragEvent, entry: BrowserEntry) => void;
		onDrop: (target: number) => void;
		onRenameKeydown: (e: KeyboardEvent) => void;
		onRenameCancel: () => void;
	} = $props();

	const key = (entry: BrowserEntry) => `${entry.type}:${entry.id}`;
	const isDragged = (entry: BrowserEntry) =>
		dnd.item?.type === entry.type && dnd.item?.id === entry.id;
</script>

<Table>
	<thead>
		<tr>
			<th scope="col">
				<div class="flex items-center gap-3">
					{#if selection.mode}
						<SelectToggle
							checked={selection.all}
							indeterminate={selection.count > 0 && !selection.all}
							label={t('files.selectEverything')}
							onToggle={() => (selection.all ? selection.clear() : selection.selectAll())}
						/>
					{/if}
					{t('common.name')}
				</div>
			</th>
			<th scope="col" class="hidden sm:table-cell">{t('common.size')}</th>
			<th scope="col" class="hidden md:table-cell">{t('common.modified')}</th>
			<th scope="col" class="w-12 text-right"><span class="sr-only">{t('files.actions')}</span></th>
		</tr>
	</thead>
	<tbody>
		{#each entries as entry, i (key(entry))}
			{@const selected = selection.has(entry.type, entry.id)}
			{@const dropTarget = entry.type === 'folder' && dnd.target === entry.id}
			<tr
				class="group transition-colors {dropTarget
					? 'bg-fc-accent/10 outline-2 -outline-offset-2 outline-fc-accent/50'
					: selected
						? 'bg-fc-surface'
						: 'hover:bg-fc-surface'} {isDragged(entry) ? 'opacity-40' : ''}"
				draggable={!selection.mode}
				ondragstart={(e) => onDragStart(e, entry)}
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
				<td>
					<div class="flex items-center gap-3">
						{#if selection.mode}
							<SelectToggle
								checked={selected}
								label={t('files.selectItem', { name: entry.name })}
								onToggle={(e) => {
									e.stopPropagation();
									onSelect(entry, i, e);
								}}
							/>
						{/if}
						<iconify-icon
							icon={entry.icon}
							width="18"
							height="18"
							class="block shrink-0 text-fc-fg-muted"
						></iconify-icon>

						{#if renamingKey === key(entry)}
							<!-- svelte-ignore a11y_autofocus -->
							<input
								bind:value={renameValue}
								onkeydown={onRenameKeydown}
								onblur={onRenameCancel}
								onclick={(e) => e.stopPropagation()}
								aria-label={t('files.newNameFor', { name: entry.name })}
								autofocus
								class="h-8 min-w-0 flex-1 rounded-fc-sm bg-fc-field px-2 text-fc-sm text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
							/>
						{:else}
							<!-- The name is the button, not the row: a click handler on a <tr> is
							     unreachable by keyboard, which is how every row in this table shipped. -->
							<button
								type="button"
								class="min-w-0 flex-1 truncate text-left font-medium [@media(pointer:coarse)]:min-h-11 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
								onclick={(e) => (selection.mode ? onSelect(entry, i, e) : onOpen(entry))}
							>
								{entry.name}
							</button>
						{/if}
					</div>
				</td>
				<td class="hidden text-fc-fg-muted sm:table-cell">
					{entry.size === null ? '—' : formatSize(entry.size)}
				</td>
				<td class="hidden text-fc-fg-muted md:table-cell">{formatDate(entry.date)}</td>
				<td class="text-right">
					<button
						type="button"
						aria-label={t('files.actionsFor', { name: entry.name })}
						onclick={(e) => {
							e.stopPropagation();
							onMenu(e, entry);
						}}
						class="inline-flex size-8 [@media(pointer:coarse)]:size-11 items-center justify-center rounded-fc-md text-fc-fg-muted transition hover:bg-fc-surface hover:text-fc-fg [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-hover:opacity-100 [@media(hover:hover)]:focus-visible:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
					>
						<iconify-icon icon={nuage.more} width="16" height="16" class="block"></iconify-icon>
					</button>
				</td>
			</tr>
		{/each}
	</tbody>
</Table>
