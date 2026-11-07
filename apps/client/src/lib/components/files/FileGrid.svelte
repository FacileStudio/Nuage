<script lang="ts">
	import { Card } from '@facile/muse';
	import { SvelteMap } from 'svelte/reactivity';
	import type { Folder } from '$lib/backend';
	import type { BrowserEntry } from '$lib/files/entries';
	import type { DragMove } from '$lib/files/dnd.svelte';
	import type { Selection } from '$lib/files/selection.svelte';
	import { formatSize, relativeTime } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';
	import { nuage } from '$lib/icons';
	import SelectToggle from './SelectToggle.svelte';

	let {
		entries,
		folders,
		selection,
		dnd,
		thumbnail,
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
		thumbnail: (entry: BrowserEntry) => { src: string; srcset?: string };
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

	const THUMBNAIL_SIZES =
		'(min-width: 1536px) 12vw, (min-width: 1024px) 18vw, (min-width: 640px) 28vw, 44vw';
	const MAX_ATTEMPTS = 2;
	const RETRY_DELAY_MS = 1500;
	const failures = new SvelteMap<number, number>();

	const failuresOf = (id: number) => failures.get(id) ?? 0;

	function retryLater(id: number) {
		setTimeout(() => failures.set(id, failuresOf(id) + 1), RETRY_DELAY_MS);
	}

	const OVERLAY =
		'[@media(pointer:coarse)]:absolute [@media(pointer:coarse)]:top-3 [@media(pointer:coarse)]:right-3 [@media(pointer:coarse)]:z-10 [@media(pointer:coarse)]:flex [@media(pointer:coarse)]:size-11 [@media(pointer:coarse)]:items-center [@media(pointer:coarse)]:justify-center [@media(pointer:coarse)]:rounded-fc-pill [@media(pointer:coarse)]:bg-fc-component/90 [@media(pointer:coarse)]:shadow-md [@media(pointer:coarse)]:backdrop-blur';

	const key = (entry: BrowserEntry) => `${entry.type}:${entry.id}`;
	const isDragged = (entry: BrowserEntry) =>
		dnd.item?.type === entry.type && dnd.item?.id === entry.id;
	const folderEntries = $derived(entries.filter((e) => e.type === 'folder'));
	const fileEntries = $derived(entries.filter((e) => e.type === 'file'));

	function dropHandlers(entry: BrowserEntry) {
		if (entry.type !== 'folder') return {};
		const legal = () => dnd.canDropOn(entry.id, folders);
		return {
			ondragover: (e: DragEvent) => {
				if (!legal()) return;
				e.preventDefault();
				e.dataTransfer!.dropEffect = 'move';
			},
			ondragenter: (e: DragEvent) => {
				if (!legal()) return;
				e.preventDefault();
				dnd.enter(key(entry), entry.id);
			},
			ondragleave: () => dnd.leave(key(entry), entry.id),
			ondrop: (e: DragEvent) => {
				if (!legal()) return;
				e.preventDefault();
				e.stopPropagation();
				onDrop(entry.id);
			}
		};
	}

	function tileClass(entry: BrowserEntry, selected: boolean) {
		const state =
			entry.type === 'folder' && dnd.target === entry.id
				? 'bg-fc-surface outline-2 outline-fc-accent'
				: selected
					? 'bg-fc-surface outline-2 outline-fc-accent'
					: 'hover:bg-fc-surface';
		const offscreen =
			entry.type === 'folder'
				? '[content-visibility:auto] [contain-intrinsic-size:auto_3.5rem]'
				: '[content-visibility:auto] [contain-intrinsic-size:auto_14rem]';
		return `group relative flex flex-col p-0 transition-colors ${offscreen} ${state} ${isDragged(entry) ? 'opacity-40' : ''}`;
	}

	function menuFromButton(e: MouseEvent, entry: BrowserEntry) {
		e.stopPropagation();
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		onMenu(new MouseEvent('contextmenu', { clientX: rect.left, clientY: rect.bottom + 4 }), entry);
	}
</script>

{#snippet name(entry: BrowserEntry)}
	{#if renamingKey === key(entry)}
		<!-- svelte-ignore a11y_autofocus -->
		<input
			bind:value={renameValue}
			onkeydown={onRenameKeydown}
			onblur={onRenameCancel}
			onclick={(e) => e.stopPropagation()}
			aria-label={t('files.newNameFor', { name: entry.name })}
			autofocus
			class="h-8 w-full min-w-0 rounded-fc-sm bg-fc-field px-2 text-fc-sm text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
		/>
	{:else}
		<span class="block truncate text-fc-sm font-medium text-fc-fg" title={entry.name}>
			{entry.name}
		</span>
	{/if}
{/snippet}

{#snippet actions(entry: BrowserEntry)}
	<span class={entry.type === 'file' ? OVERLAY : ''}>
	{#if selection.mode}
		<SelectToggle
			checked={selection.has(entry.type, entry.id)}
			label={t('files.selectItem', { name: entry.name })}
			onToggle={(e) => {
				e.stopPropagation();
				onSelect(entry, entries.indexOf(entry), e);
			}}
		/>
	{:else}
		<button
			type="button"
			aria-label={t('files.actionsFor', { name: entry.name })}
			onclick={(e) => menuFromButton(e, entry)}
			class="inline-flex size-8 [@media(pointer:coarse)]:size-11 shrink-0 items-center justify-center rounded-fc-pill text-fc-fg-muted transition hover:bg-fc-bg hover:text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-hover:opacity-100 [@media(hover:hover)]:focus-visible:opacity-100"
		>
			<iconify-icon icon={nuage.more} width="16" height="16" class="block"></iconify-icon>
		</button>
	{/if}
	</span>
{/snippet}

{#snippet tile(entry: BrowserEntry)}
	{@const index = entries.indexOf(entry)}
	{@const selected = selection.has(entry.type, entry.id)}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<Card
		class={tileClass(entry, selected)}
		draggable={!selection.mode && renamingKey !== key(entry)}
		ondragstart={(e: DragEvent) => onDragStart(e, entry)}
		ondragend={() => dnd.end()}
		oncontextmenu={(e: MouseEvent) => onMenu(e, entry)}
		{...dropHandlers(entry)}
	>
		{#if entry.type === 'file'}
			<button
				type="button"
				aria-label={t('files.open', { name: entry.name })}
				onclick={(e) => (selection.mode ? onSelect(entry, index, e) : onOpen(entry))}
				class="m-2 mb-0 block overflow-hidden rounded-fc-sm bg-fc-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
			>
				{#if entry.isImage && failuresOf(entry.id) < MAX_ATTEMPTS}
					{@const preview = thumbnail(entry)}
					{@const retry = failuresOf(entry.id)}
					<img
						src={retry ? `${preview.src}&r=${retry}` : preview.src}
						srcset={retry ? undefined : preview.srcset}
						sizes={THUMBNAIL_SIZES}
						alt=""
						loading="lazy"
						decoding="async"
						draggable="false"
						onerror={() => retryLater(entry.id)}
						class="aspect-[4/3] w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
					/>
				{:else}
					<span class="flex aspect-[4/3] w-full items-center justify-center text-fc-fg-muted">
						<iconify-icon icon={entry.icon} width="40" height="40" class="block"></iconify-icon>
					</span>
				{/if}
			</button>
		{/if}
		<div class="flex min-h-11 min-w-0 items-center gap-2.5 px-3 py-2.5">
			{#if entry.type === 'folder'}
				<button
					type="button"
					aria-label={t('files.open', { name: entry.name })}
					onclick={(e) => (selection.mode ? onSelect(entry, index, e) : onOpen(entry))}
					class="absolute inset-0 rounded-fc-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
				></button>
			{/if}
			<iconify-icon
				icon={entry.icon}
				width="20"
				height="20"
				class="pointer-events-none relative block shrink-0 text-fc-fg-muted {entry.type === 'file' ? 'max-sm:hidden' : ''}"
			></iconify-icon>
			<div class="pointer-events-none relative min-w-0 flex-1 [&_input]:pointer-events-auto">
				{@render name(entry)}
				{#if entry.type === 'file'}
					<span class="block truncate text-fc-xs text-fc-fg-muted">
						{formatSize(entry.size ?? 0)} · {relativeTime(entry.date)}
					</span>
				{/if}
			</div>
			<div class="relative {entry.type === 'file' ? '[@media(pointer:coarse)]:contents' : ''}">
				{@render actions(entry)}
			</div>
		</div>
	</Card>
{/snippet}

<div class="flex flex-col gap-6">
	{#if folderEntries.length > 0}
		<section class="flex flex-col gap-3" aria-label={t('files.folders')}>
			<h2 class="text-fc-xs font-medium tracking-wide text-fc-fg-muted uppercase">{t('files.folders')}</h2>
			<div class="grid grid-cols-1 gap-4 min-[480px]:grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6">
				{#each folderEntries as entry (key(entry))}
					{@render tile(entry)}
				{/each}
			</div>
		</section>
	{/if}
	{#if fileEntries.length > 0}
		<section class="flex flex-col gap-3" aria-label={t('files.files')}>
			<h2 class="text-fc-xs font-medium tracking-wide text-fc-fg-muted uppercase">{t('files.files')}</h2>
			<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6">
				{#each fileEntries as entry (key(entry))}
					{@render tile(entry)}
				{/each}
			</div>
		</section>
	{/if}
</div>
