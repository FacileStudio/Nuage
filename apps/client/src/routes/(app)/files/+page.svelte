<script lang="ts">
	import { getContext, onMount, untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		Button,
		ConfirmModal,
		EmptyState,
		Field,
		Input,
		Modal,
		Spinner,
		UploadProgress,
		toast
	} from '@facile/muse';
	import { backend, type Folder, type NuageFile } from '$lib/backend';
	import { pushUndo } from '$lib/undo.svelte';
	import { getSpaceStore } from '$lib/space.svelte';
	import { icons, nuage } from '$lib/icons';
	import { isPreviewable, toEntries, type BrowserEntry } from '$lib/files/entries';
	import { DragMove, isInternalDrag } from '$lib/files/dnd.svelte';
	import { Selection, itemKey } from '$lib/files/selection.svelte';
	import { UploadQueue } from '$lib/files/uploads.svelte';
	import Breadcrumbs from '$lib/components/files/Breadcrumbs.svelte';
	import ContextMenu, { type MenuItem } from '$lib/components/files/ContextMenu.svelte';
	import FileGrid from '$lib/components/files/FileGrid.svelte';
	import FilePreview from '$lib/components/files/FilePreview.svelte';
	import FileTable from '$lib/components/files/FileTable.svelte';
	import ShareModal from '$lib/components/files/ShareModal.svelte';

	const app = getContext<{ token: string; refreshQuota: () => void }>('app');
	const spaceStore = getSpaceStore();

	let files = $state<NuageFile[]>([]);
	let folders = $state<Folder[]>([]);
	let breadcrumbs = $state<{ id: number | null; name: string }[]>([{ id: null, name: 'Files' }]);
	let currentFolderId = $state<number | null>(null);
	let viewMode = $state<'grid' | 'list'>('list');
	let loading = $state(true);
	let searchQuery = $state('');
	let searchTimer: ReturnType<typeof setTimeout> | null = null;
	let isMac = $state(false);

	const entries = $derived(toEntries(folders, files));
	const selection = new Selection(() => entries.map((e) => ({ type: e.type, id: e.id })));
	const dnd = new DragMove();
	const uploads = new UploadQueue();

	let fileInput = $state<HTMLInputElement | null>(null);
	let dropDepth = $state(0);
	const showDropOverlay = $derived(dropDepth > 0);

	let menu = $state<{ x: number; y: number; entry: BrowserEntry | null } | null>(null);
	let renaming = $state<BrowserEntry | null>(null);
	let renameValue = $state('');
	let previewFile = $state<NuageFile | null>(null);
	let shareTarget = $state<{ type: 'file' | 'folder'; id: number; name: string } | null>(null);
	let newFolderOpen = $state(false);
	let newFolderName = $state('');
	let pendingDelete = $state<BrowserEntry[] | null>(null);

	const folderFromUrl = $derived.by(() => {
		const raw = page.url.searchParams.get('folder');
		if (!raw) return null;
		const parsed = Number(raw);
		return Number.isFinite(parsed) ? parsed : null;
	});

	/*
	 * Reloads when the folder or the active space changes, and on nothing else. The body is
	 * untracked on purpose: `loadContents` reads `searchQuery` before its first await, so
	 * without this the effect took a dependency on it and re-fired a request on every
	 * keystroke — running alongside, and largely defeating, the 300ms debounce below.
	 */
	$effect(() => {
		const folder = folderFromUrl;
		void spaceStore.id;
		untrack(() => {
			currentFolderId = folder;
			void loadContents();
			void loadBreadcrumbs();
		});
	});

	onMount(() => {
		isMac = navigator.platform.includes('Mac');
		document.addEventListener('click', closeMenu);
		document.addEventListener('keydown', onGlobalKeydown);
		return () => {
			document.removeEventListener('click', closeMenu);
			document.removeEventListener('keydown', onGlobalKeydown);
		};
	});

	async function loadContents() {
		loading = true;
		selection.clear();
		try {
			const spaceId = spaceStore.id ?? undefined;
			const [fileRes, folderRes] = await Promise.all([
				backend.listFiles(app.token, {
					folder_id: currentFolderId ?? undefined,
					search: searchQuery || undefined,
					space_id: spaceId
				}),
				backend.listFolders(
					app.token,
					currentFolderId != null
						? { parent_id: currentFolderId, space_id: spaceId }
						: { space_id: spaceId }
				)
			]);
			files = fileRes.files ?? [];
			/* A search spans folders, so the folder column would be listing siblings that have
			   nothing to do with the query. */
			folders = searchQuery ? [] : (folderRes.folders ?? []);
		} catch {
			files = [];
			folders = [];
			toast.danger('Could not load this folder.');
		}
		loading = false;
	}

	async function loadBreadcrumbs() {
		if (currentFolderId == null) {
			breadcrumbs = [{ id: null, name: 'Files' }];
			return;
		}
		const trail: { id: number | null; name: string }[] = [];
		let folderId: number | null = currentFolderId;
		while (folderId != null) {
			try {
				const res = await backend.getFolder(app.token, folderId);
				trail.unshift({ id: res.folder.id, name: res.folder.name });
				folderId = res.folder.parent_id;
			} catch {
				break;
			}
		}
		breadcrumbs = [{ id: null, name: 'Files' }, ...trail];
	}

	const navigateTo = (id: number | null) => goto(id == null ? '/files' : `/files?folder=${id}`);

	function onSearchInput() {
		if (searchTimer) clearTimeout(searchTimer);
		searchTimer = setTimeout(loadContents, 300);
	}

	function openEntry(entry: BrowserEntry) {
		if (entry.type === 'folder') {
			navigateTo(entry.id);
			return;
		}
		const file = entry.raw as NuageFile;
		if (isPreviewable(file.mime_type)) previewFile = file;
		else window.open(backend.downloadUrl(file.id), '_blank');
	}

	/* Only reachable in select mode — the row and tile hand a plain click to `onOpen`, and the
	   checkbox does not exist outside it. So a click toggles and shift extends; there is no
	   "replace the selection" case to handle. */
	function onSelect(entry: BrowserEntry, index: number, e: MouseEvent) {
		if (e.shiftKey) {
			selection.extendTo(entry.type, entry.id, index, isMac ? e.metaKey : e.ctrlKey);
		} else {
			selection.toggle(entry.type, entry.id, index);
		}
	}

	function openMenu(e: MouseEvent, entry: BrowserEntry | null) {
		e.preventDefault();
		e.stopPropagation();
		if (entry && !selection.has(entry.type, entry.id)) selection.clear();
		menu = { x: e.clientX, y: e.clientY, entry };
	}

	const closeMenu = () => (menu = null);

	async function uploadFrom(list: FileList | null) {
		const chosen = [...(list ?? [])];
		if (chosen.length === 0) return;

		await uploads.run(chosen, {
			token: app.token,
			folderId: currentFolderId,
			spaceId: spaceStore.id
		});

		const failed = uploads.failed.length;
		if (failed > 0) toast.danger(`${failed} of ${chosen.length} uploads failed.`);
		else toast.success(chosen.length === 1 ? 'Uploaded.' : `Uploaded ${chosen.length} files.`);

		await loadContents();
		app.refreshQuota();
		/* Held until the listing has refreshed, so the rows do not vanish before the files
		   they describe appear underneath. Failures stay up — they are the report. */
		if (failed === 0) uploads.reset();
	}

	async function createFolder() {
		const name = newFolderName.trim();
		if (!name) return;
		try {
			await backend.createFolder(app.token, {
				name,
				parent_id: currentFolderId,
				space_id: spaceStore.id
			});
			newFolderName = '';
			newFolderOpen = false;
			await loadContents();
		} catch {
			toast.danger('Could not create that folder.');
		}
	}

	function startRename(entry: BrowserEntry) {
		renaming = entry;
		renameValue = entry.name;
	}

	const cancelRename = () => {
		renaming = null;
		renameValue = '';
	};

	async function submitRename() {
		const target = renaming;
		const next = renameValue.trim();
		cancelRename();
		if (!target || !next || next === target.name) return;

		const { type, id } = target;
		const previous = target.name;
		try {
			if (type === 'file') await backend.updateFile(app.token, id, { name: next });
			else await backend.updateFolder(app.token, id, { name: next });

			pushUndo({
				label: `Renamed to ${next}`,
				async execute() {
					if (type === 'file') await backend.updateFile(app.token, id, { name: previous });
					else await backend.updateFolder(app.token, id, { name: previous });
					await loadContents();
				}
			});
			await loadContents();
		} catch {
			toast.danger('Could not rename that.');
		}
	}

	function onRenameKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			void submitRename();
		}
		if (e.key === 'Escape') cancelRename();
	}

	async function moveInto(targetFolderId: number | null) {
		const moving = dnd.item;
		dnd.end();
		if (!moving || targetFolderId === currentFolderId) return;

		const { type, id } = moving;
		const origin = currentFolderId;
		try {
			if (type === 'file') await backend.updateFile(app.token, id, { folder_id: targetFolderId });
			else await backend.updateFolder(app.token, id, { parent_id: targetFolderId });

			pushUndo({
				label: 'Item moved',
				async execute() {
					if (type === 'file') await backend.updateFile(app.token, id, { folder_id: origin });
					else await backend.updateFolder(app.token, id, { parent_id: origin });
					await loadContents();
				}
			});
			await loadContents();
		} catch {
			toast.danger('Could not move that.');
		}
	}

	function requestDelete(entry: BrowserEntry | null) {
		if (selection.count > 1) {
			const keys = new Set(selection.keys);
			pendingDelete = entries.filter((e) => keys.has(itemKey(e.type, e.id)));
		} else if (entry) {
			pendingDelete = [entry];
		}
	}

	async function confirmDelete() {
		const targets = pendingDelete ?? [];
		if (targets.length === 0) return;

		const results = await Promise.allSettled(
			targets.map((t) =>
				t.type === 'file'
					? backend.deleteFile(app.token, t.id)
					: backend.deleteFolder(app.token, t.id)
			)
		);

		/* Only the ones that actually went to the trash are offered back — an undo that tries
		   to restore a file which was never deleted just produces a second error. */
		const deleted = targets.filter((_, i) => results[i].status === 'fulfilled');
		const failed = targets.length - deleted.length;
		if (failed > 0) toast.danger(`${failed} ${failed === 1 ? 'item' : 'items'} could not be deleted.`);

		if (deleted.length > 0) {
			pushUndo({
				label:
					deleted.length === 1
						? `“${deleted[0].name}” moved to trash`
						: `${deleted.length} items moved to trash`,
				async execute() {
					await Promise.allSettled(
						deleted.map((t) => backend.restoreItem(app.token, t.type, t.id))
					);
					await loadContents();
					app.refreshQuota();
				}
			});
		}

		pendingDelete = null;
		selection.exit();
		await loadContents();
		app.refreshQuota();
	}

	function onGlobalKeydown(e: KeyboardEvent) {
		if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
		if (previewFile || newFolderOpen || pendingDelete || shareTarget) return;

		const mod = isMac ? e.metaKey : e.ctrlKey;
		if (mod && e.key === 'a' && selection.mode) {
			e.preventDefault();
			selection.selectAll();
		} else if (e.key === 'Escape') {
			if (menu) closeMenu();
			else if (selection.mode) selection.exit();
		} else if ((e.key === 'Delete' || e.key === 'Backspace') && selection.count > 0) {
			e.preventDefault();
			requestDelete(null);
		}
	}

	const menuItems = $derived.by((): MenuItem[] => {
		if (!menu) return [];
		const { entry } = menu;

		if (!entry) {
			return [
				{ label: 'Upload files', icon: icons.upload, onSelect: () => fileInput?.click() },
				{ label: 'New folder', icon: nuage.newFolder, onSelect: () => (newFolderOpen = true) }
			];
		}

		const many = selection.count > 1;
		const items: MenuItem[] = [];

		if (!many && entry.type === 'file') {
			items.push({
				label: 'Download',
				icon: icons.download,
				onSelect: () => {
					const a = document.createElement('a');
					a.href = backend.downloadUrl(entry.id);
					a.download = entry.name;
					a.click();
				}
			});
		}
		if (!many) {
			items.push({ label: 'Rename', icon: nuage.rename, onSelect: () => startRename(entry) });
			items.push({
				label: 'Share',
				icon: nuage.share,
				onSelect: () => (shareTarget = { type: entry.type, id: entry.id, name: entry.name })
			});
		}
		items.push({
			label: many ? `Move ${selection.count} items to trash` : 'Move to trash',
			icon: icons.remove,
			tone: 'danger',
			onSelect: () => requestDelete(entry)
		});
		return items;
	});

	const thumbnailUrl = (entry: BrowserEntry) => backend.downloadUrl(entry.id);
	const renamingKey = $derived(renaming ? itemKey(renaming.type, renaming.id) : null);
</script>

<svelte:head>
	<title>Files — Nuage</title>
</svelte:head>

<!-- The page is a drop surface for uploads. It carries no role on purpose: it is not a widget,
     and the keyboard path to the same action is the Upload button a few lines down. The
     previous version declared `role="application"`, which tells a screen reader to hand every
     keystroke to the page and suppress its own navigation — a much worse lie than no role. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="relative flex min-h-full flex-col"
	ondrop={(e) => {
		if (isInternalDrag(e)) return;
		e.preventDefault();
		dropDepth = 0;
		void uploadFrom(e.dataTransfer?.files ?? null);
	}}
	ondragover={(e) => !isInternalDrag(e) && e.preventDefault()}
	ondragenter={(e) => !isInternalDrag(e) && (dropDepth += 1)}
	ondragleave={(e) => !isInternalDrag(e) && (dropDepth -= 1)}
>
	{#if showDropOverlay}
		<div
			class="pointer-events-none absolute inset-2 z-30 flex items-center justify-center rounded-fc-lg border-2 border-dashed border-fc-accent bg-fc-accent/5"
		>
			<div class="flex flex-col items-center gap-2 text-fc-fg">
				<iconify-icon icon={icons.upload} width="32" height="32" class="block"></iconify-icon>
				<p class="text-fc-sm font-medium">Drop to upload</p>
			</div>
		</div>
	{/if}

	<div class="flex flex-col gap-4 px-4 py-6 md:px-8">
		<div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
			<Breadcrumbs crumbs={breadcrumbs} {dnd} onNavigate={navigateTo} onDrop={moveInto} />

			<div class="flex items-center gap-2">
				<div class="relative flex-1 lg:flex-none">
					<iconify-icon
						icon={icons.search}
						width="16"
						height="16"
						class="pointer-events-none absolute top-1/2 left-3 block -translate-y-1/2 text-fc-fg-muted"
					></iconify-icon>
					<input
						type="search"
						placeholder="Search files"
						aria-label="Search files"
						bind:value={searchQuery}
						oninput={onSearchInput}
						class="h-9 w-full rounded-fc-md border border-fc-border bg-fc-bg pr-3 pl-9 text-fc-sm text-fc-fg placeholder:text-fc-fg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring lg:w-56"
					/>
				</div>

				<div class="flex overflow-hidden rounded-fc-md border border-fc-border">
					{#each [{ mode: 'list', icon: nuage.list, label: 'List view' }, { mode: 'grid', icon: nuage.grid, label: 'Grid view' }] as option (option.mode)}
						<button
							type="button"
							aria-label={option.label}
							aria-pressed={viewMode === option.mode}
							onclick={() => (viewMode = option.mode as 'grid' | 'list')}
							class="flex size-9 items-center justify-center transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-fc-ring {viewMode ===
							option.mode
								? 'bg-fc-accent text-fc-accent-fg'
								: 'text-fc-fg-muted hover:bg-fc-surface hover:text-fc-fg'}"
						>
							<iconify-icon icon={option.icon} width="16" height="16" class="block"></iconify-icon>
						</button>
					{/each}
				</div>
			</div>
		</div>

		<div class="flex flex-wrap items-center gap-2">
			{#if selection.mode}
				<Button variant="ghost" icon={icons.close} onclick={() => selection.exit()}>Done</Button>
				<span class="text-fc-sm text-fc-fg-muted">{selection.count} selected</span>
				<div class="flex-1"></div>
				<Button
					variant="outline"
					onclick={() => (selection.all ? selection.clear() : selection.selectAll())}
				>
					{selection.all ? 'Deselect all' : 'Select all'}
				</Button>
				<Button
					variant="danger"
					icon={icons.remove}
					disabled={selection.count === 0}
					onclick={() => requestDelete(null)}
				>
					Move to trash
				</Button>
			{:else}
				<Button icon={icons.upload} onclick={() => fileInput?.click()}>Upload</Button>
				<Button variant="outline" icon={nuage.newFolder} onclick={() => (newFolderOpen = true)}>
					New folder
				</Button>
				<Button variant="ghost" icon={icons.check} onclick={() => selection.enter()}>Select</Button>
			{/if}
		</div>

		{#if uploads.items.length > 0}
			<UploadProgress items={uploads.items} onCancel={(id) => uploads.remove(id)} />
		{/if}
	</div>

	<!-- Right-click on empty space offers Upload / New folder. Both are reachable from the
	     toolbar, so this is a shortcut rather than the only route. -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="flex-1 px-4 pb-6 md:px-8" oncontextmenu={(e) => openMenu(e, null)}>
		{#if loading}
			<div class="flex h-64 items-center justify-center"><Spinner /></div>
		{:else if entries.length === 0}
			<EmptyState
				icon={searchQuery ? icons.search : nuage.folderOpen}
				title={searchQuery ? 'Nothing matches that' : 'This folder is empty'}
				description={searchQuery
					? 'Try a shorter term — search looks at file names only.'
					: 'Drop files anywhere on this page, or use the buttons above.'}
			>
				{#if !searchQuery}
					<Button icon={icons.upload} onclick={() => fileInput?.click()}>Upload files</Button>
				{/if}
			</EmptyState>
		{:else if viewMode === 'grid'}
			<FileGrid
				{entries}
				{folders}
				{selection}
				{dnd}
				{thumbnailUrl}
				{renamingKey}
				bind:renameValue
				onOpen={openEntry}
				{onSelect}
				onMenu={openMenu}
				onDrop={moveInto}
				{onRenameKeydown}
				onRenameCancel={cancelRename}
			/>
		{:else}
			<FileTable
				{entries}
				{folders}
				{selection}
				{dnd}
				{renamingKey}
				bind:renameValue
				onOpen={openEntry}
				{onSelect}
				onMenu={openMenu}
				onDrop={moveInto}
				{onRenameKeydown}
				onRenameCancel={cancelRename}
			/>
		{/if}
	</div>
</div>

<input
	bind:this={fileInput}
	type="file"
	multiple
	class="hidden"
	onchange={(e) => {
		const input = e.currentTarget;
		void uploadFrom(input.files).then(() => (input.value = ''));
	}}
/>

{#if menu}
	<ContextMenu
		x={menu.x}
		y={menu.y}
		items={menuItems}
		heading={menu.entry && selection.count <= 1
			? { title: menu.entry.name, detail: menu.entry.type === 'folder' ? 'Folder' : undefined }
			: undefined}
		onClose={closeMenu}
	/>
{/if}

{#if previewFile}
	<FilePreview
		file={previewFile}
		url={backend.downloadUrl(previewFile.id)}
		onClose={() => (previewFile = null)}
	/>
{/if}

{#if shareTarget}
	<ShareModal
		token={app.token}
		spaceId={spaceStore.id}
		target={shareTarget}
		onClose={() => (shareTarget = null)}
	/>
{/if}

<Modal bind:open={newFolderOpen} title="New folder" showClose>
	<form
		onsubmit={(e) => {
			e.preventDefault();
			void createFolder();
		}}
	>
		<Field label="Name">
			<Input bind:value={newFolderName} placeholder="Invoices" autocomplete="off" />
		</Field>
	</form>

	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (newFolderOpen = false)}>Cancel</Button>
			<Button icon={icons.plus} disabled={!newFolderName.trim()} onclick={createFolder}>
				Create
			</Button>
		</div>
	{/snippet}
</Modal>

<ConfirmModal
	open={pendingDelete !== null}
	title="Move to trash?"
	description={pendingDelete?.length === 1
		? `“${pendingDelete[0].name}” goes to the trash. You can restore it from there, and nothing is removed from the server yet.`
		: `${pendingDelete?.length ?? 0} items go to the trash. You can restore them from there, and nothing is removed from the server yet.`}
	confirmLabel="Move to trash"
	tone="danger"
	onConfirm={confirmDelete}
	onCancel={() => (pendingDelete = null)}
/>
