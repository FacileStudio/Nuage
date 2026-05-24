<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { backend, type NuageFile, type Folder } from '$lib/backend';

	const app = getContext<{ token: string; user: { id: string; email: string; name: string } | null }>('app');

	let files = $state<NuageFile[]>([]);
	let folders = $state<Folder[]>([]);
	let breadcrumbs = $state<{ id: number | null; name: string }[]>([{ id: null, name: 'Files' }]);
	let currentFolderId = $state<number | null>(null);
	let viewMode = $state<'grid' | 'list'>('grid');
	let loading = $state(true);
	let searchQuery = $state('');
	let searchTimeout = $state<ReturnType<typeof setTimeout> | null>(null);

	let dragCounter = $state(0);
	let dragging = $derived(dragCounter > 0);
	let uploading = $state(false);
	let uploadProgress = $state('');

	let contextMenu = $state<{ x: number; y: number; type: 'file' | 'folder'; item: NuageFile | Folder } | null>(null);
	let renameTarget = $state<{ type: 'file' | 'folder'; item: NuageFile | Folder } | null>(null);
	let renameValue = $state('');

	let previewFile = $state<NuageFile | null>(null);
	let pdfPageNum = $state(1);
	let pdfTotalPages = $state(0);
	let pdfScale = $state(1.0);
	let pdfDoc = $state<any>(null);
	let pdfCanvas = $state<HTMLCanvasElement | null>(null);

	async function loadPdf(url: string) {
		pdfPageNum = 1;
		pdfScale = 1.0;
		pdfTotalPages = 0;
		pdfDoc = null;
		const pdfjsLib = await import('pdfjs-dist');
		pdfjsLib.GlobalWorkerOptions.workerSrc = `https://cdnjs.cloudflare.com/ajax/libs/pdf.js/${pdfjsLib.version}/pdf.worker.min.mjs`;
		const doc = await pdfjsLib.getDocument(url).promise;
		pdfDoc = doc;
		pdfTotalPages = doc.numPages;
		await renderPdfPage();
	}

	async function renderPdfPage() {
		if (!pdfDoc || !pdfCanvas) return;
		const page = await pdfDoc.getPage(pdfPageNum);
		const viewport = page.getViewport({ scale: pdfScale });
		pdfCanvas.width = viewport.width;
		pdfCanvas.height = viewport.height;
		const ctx = pdfCanvas.getContext('2d')!;
		ctx.fillStyle = '#ffffff';
		ctx.fillRect(0, 0, viewport.width, viewport.height);
		await page.render({ canvasContext: ctx, viewport }).promise;
	}

	function pdfPrev() {
		if (pdfPageNum > 1) { pdfPageNum--; renderPdfPage(); }
	}
	function pdfNext() {
		if (pdfPageNum < pdfTotalPages) { pdfPageNum++; renderPdfPage(); }
	}
	function pdfZoomIn() { pdfScale = Math.min(pdfScale + 0.25, 3); renderPdfPage(); }
	function pdfZoomOut() { pdfScale = Math.max(pdfScale - 0.25, 0.5); renderPdfPage(); }

	let showNewFolderDialog = $state(false);
	let newFolderName = $state('');

	let currentFolderIdFromUrl = $derived.by(() => {
		const raw = page.url.searchParams.get('folder');
		if (!raw) return null;
		const parsed = Number(raw);
		return Number.isFinite(parsed) ? parsed : null;
	});

	$effect(() => {
		const urlFolderId = currentFolderIdFromUrl;
		currentFolderId = urlFolderId;
		loadContents();
		loadBreadcrumbs();
	});

	onMount(() => {
		document.addEventListener('click', closeContextMenu);
		return () => document.removeEventListener('click', closeContextMenu);
	});

	async function loadContents() {
		loading = true;
		try {
			const [fileRes, folderRes] = await Promise.all([
				backend.listFiles(app.token, {
					folder_id: currentFolderId ?? undefined,
					search: searchQuery || undefined
				}),
				backend.listFolders(app.token, currentFolderId != null ? { parent_id: currentFolderId } : undefined)
			]);
			files = fileRes.files ?? [];
			folders = searchQuery ? [] : (folderRes.folders ?? []);
		} catch {
			files = [];
			folders = [];
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
		trail.unshift({ id: null, name: 'Files' });
		breadcrumbs = trail;
	}

	function navigateToFolder(folderId: number | null) {
		if (folderId != null) {
			goto(`/drive?folder=${folderId}`);
		} else {
			goto('/drive');
		}
	}

	function openFolder(folder: Folder) {
		navigateToFolder(folder.id);
	}

	function handleSearch() {
		if (searchTimeout) clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			loadContents();
		}, 300);
	}

	async function handleFileDrop(e: DragEvent) {
		e.preventDefault();
		dragCounter = 0;
		const droppedFiles = e.dataTransfer?.files;
		if (!droppedFiles?.length) return;
		await uploadFiles(droppedFiles);
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
	}

	function handleDragEnter(e: DragEvent) {
		e.preventDefault();
		dragCounter++;
	}

	function handleDragLeave() {
		dragCounter--;
	}

	async function handleFileInput(e: Event) {
		const input = e.target as HTMLInputElement;
		if (!input.files?.length) return;
		await uploadFiles(input.files);
		input.value = '';
	}

	async function uploadFiles(fileList: FileList) {
		uploading = true;
		const total = fileList.length;
		let done = 0;
		for (const file of fileList) {
			uploadProgress = `Uploading ${done + 1}/${total}: ${file.name}`;
			const formData = new FormData();
			formData.set('file', file);
			if (currentFolderId != null) formData.set('folder_id', String(currentFolderId));
			try {
				await backend.uploadFile(app.token, formData);
			} catch {}
			done++;
		}
		uploading = false;
		uploadProgress = '';
		await loadContents();
	}

	async function createFolder() {
		if (!newFolderName.trim()) return;
		try {
			await backend.createFolder(app.token, {
				name: newFolderName.trim(),
				parent_id: currentFolderId
			});
			newFolderName = '';
			showNewFolderDialog = false;
			await loadContents();
		} catch {}
	}

	function openContextMenu(e: MouseEvent, type: 'file' | 'folder', item: NuageFile | Folder) {
		e.preventDefault();
		contextMenu = { x: e.clientX, y: e.clientY, type, item };
	}

	function closeContextMenu() {
		contextMenu = null;
	}

	async function downloadItem() {
		if (!contextMenu || contextMenu.type !== 'file') return;
		const file = contextMenu.item as NuageFile;
		const url = backend.downloadUrl(app.token, file.id);
		const a = document.createElement('a');
		a.href = url;
		a.download = file.name;
		a.click();
		contextMenu = null;
	}

	function startRename() {
		if (!contextMenu) return;
		renameTarget = { type: contextMenu.type, item: contextMenu.item };
		renameValue = contextMenu.item.name;
		contextMenu = null;
	}

	async function submitRename() {
		if (!renameTarget || !renameValue.trim()) return;
		try {
			if (renameTarget.type === 'file') {
				await backend.updateFile(app.token, (renameTarget.item as NuageFile).id, { name: renameValue.trim() });
			} else {
				await backend.updateFolder(app.token, (renameTarget.item as Folder).id, { name: renameValue.trim() });
			}
		} catch {}
		renameTarget = null;
		renameValue = '';
		await loadContents();
	}

	function cancelRename() {
		renameTarget = null;
		renameValue = '';
	}

	async function deleteItem() {
		if (!contextMenu) return;
		const { type, item } = contextMenu;
		contextMenu = null;
		try {
			if (type === 'file') {
				await backend.deleteFile(app.token, (item as NuageFile).id);
			} else {
				await backend.deleteFolder(app.token, (item as Folder).id);
			}
		} catch {}
		await loadContents();
	}

	function openPreview(file: NuageFile) {
		if (isPreviewable(file.mime_type)) {
			previewFile = file;
			if (file.mime_type === 'application/pdf') {
				const url = backend.downloadUrl(app.token, file.id);
				setTimeout(() => loadPdf(url), 0);
			}
		} else {
			const url = backend.downloadUrl(app.token, file.id);
			window.open(url, '_blank');
		}
	}

	function closePreview() {
		previewFile = null;
		pdfDoc = null;
	}

	function isPreviewable(mime: string) {
		return mime.startsWith('image/') || mime === 'application/pdf' || mime.startsWith('video/') || mime.startsWith('audio/');
	}

	function fileIcon(mime: string): string {
		if (mime.startsWith('image/')) return 'solar:gallery-linear';
		if (mime.startsWith('video/')) return 'solar:videocamera-record-linear';
		if (mime.startsWith('audio/')) return 'solar:music-note-2-linear';
		if (mime === 'application/pdf') return 'solar:document-linear';
		if (mime.includes('zip') || mime.includes('archive') || mime.includes('compressed')) return 'solar:zip-file-linear';
		if (mime.includes('spreadsheet') || mime.includes('excel') || mime.includes('csv')) return 'solar:chart-square-linear';
		if (mime.includes('presentation') || mime.includes('powerpoint')) return 'solar:presentation-graph-linear';
		if (mime.includes('word') || mime.includes('document') || mime.startsWith('text/')) return 'solar:document-text-linear';
		return 'solar:file-linear';
	}

	function fileIconColor(mime: string): string {
		if (mime.startsWith('image/')) return 'text-emerald-600';
		if (mime.startsWith('video/')) return 'text-purple-600';
		if (mime.startsWith('audio/')) return 'text-pink-600';
		if (mime === 'application/pdf') return 'text-red-600';
		if (mime.includes('zip') || mime.includes('archive')) return 'text-amber-600';
		if (mime.includes('spreadsheet') || mime.includes('excel')) return 'text-green-600';
		return 'text-blue-600';
	}

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(1024));
		return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function handleRenameKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') submitRename();
		if (e.key === 'Escape') cancelRename();
	}

	function handleNewFolderKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') createFolder();
		if (e.key === 'Escape') { showNewFolderDialog = false; newFolderName = ''; }
	}
</script>

<svelte:head>
	<title>Files — Nuage</title>
</svelte:head>

<div
	class="relative flex h-full flex-col"
	ondrop={handleFileDrop}
	ondragover={handleDragOver}
	ondragenter={handleDragEnter}
	ondragleave={handleDragLeave}
	role="application"
>
	{#if dragging}
		<div class="pointer-events-none absolute inset-0 z-50 flex items-center justify-center rounded-lg border-2 border-dashed border-foreground/30 bg-foreground/5">
			<div class="text-center">
				<iconify-icon icon="solar:upload-linear" width="48" class="text-foreground/50"></iconify-icon>
				<p class="mt-2 text-sm font-medium text-foreground/70">Drop files to upload</p>
			</div>
		</div>
	{/if}

	<div class="flex flex-col gap-4 border-b border-border px-4 py-4 md:px-8 md:py-5">
		<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<nav class="flex items-center gap-1.5 text-sm min-w-0">
				{#each breadcrumbs as crumb, i}
					{#if i > 0}
						<iconify-icon icon="solar:alt-arrow-right-linear" width="14" class="text-muted-foreground shrink-0"></iconify-icon>
					{/if}
					{#if i === breadcrumbs.length - 1}
						<span class="font-medium truncate">{crumb.name}</span>
					{:else}
						<button
							class="truncate text-muted-foreground hover:text-foreground transition-colors"
							onclick={() => navigateToFolder(crumb.id)}
						>
							{crumb.name}
						</button>
					{/if}
				{/each}
			</nav>

			<div class="flex items-center gap-2">
				<div class="relative flex-1 sm:flex-none">
					<iconify-icon icon="solar:magnifer-linear" width="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"></iconify-icon>
					<input
						type="text"
						placeholder="Search files..."
						bind:value={searchQuery}
						oninput={handleSearch}
						class="h-9 w-full rounded-md border border-input bg-background pl-9 pr-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:w-56"
					/>
				</div>

				<div class="flex rounded-md border border-border">
					<button
						class="flex h-9 w-9 items-center justify-center transition-colors {viewMode === 'grid' ? 'bg-foreground text-background' : 'text-muted-foreground hover:text-foreground'} rounded-l-md"
						onclick={() => viewMode = 'grid'}
						aria-label="Grid view"
					>
						<iconify-icon icon="solar:widget-linear" width="16"></iconify-icon>
					</button>
					<button
						class="flex h-9 w-9 items-center justify-center transition-colors {viewMode === 'list' ? 'bg-foreground text-background' : 'text-muted-foreground hover:text-foreground'} rounded-r-md"
						onclick={() => viewMode = 'list'}
						aria-label="List view"
					>
						<iconify-icon icon="solar:list-linear" width="16"></iconify-icon>
					</button>
				</div>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<label
				class="inline-flex h-9 cursor-pointer items-center gap-2 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
			>
				<iconify-icon icon="solar:upload-linear" width="16"></iconify-icon>
				Upload
				<input type="file" multiple class="hidden" onchange={handleFileInput} />
			</label>

			<button
				class="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
				onclick={() => showNewFolderDialog = true}
			>
				<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
				New folder
			</button>
		</div>
	</div>

	{#if uploading}
		<div class="border-b border-border bg-muted/50 px-4 py-2.5 md:px-8">
			<div class="flex items-center gap-3">
				<div class="h-4 w-4 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
				<span class="text-sm text-muted-foreground">{uploadProgress}</span>
			</div>
		</div>
	{/if}

	<div class="flex-1 overflow-auto px-4 py-4 md:px-8 md:py-6">
		{#if loading}
			<div class="flex h-64 items-center justify-center">
				<div class="h-6 w-6 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
			</div>
		{:else if folders.length === 0 && files.length === 0}
			<div class="flex h-64 flex-col items-center justify-center text-center">
				<iconify-icon icon="solar:cloud-bold-duotone" width="48" class="text-muted-foreground/40"></iconify-icon>
				<p class="mt-4 text-sm font-medium text-muted-foreground">
					{searchQuery ? 'No files match your search' : 'This folder is empty'}
				</p>
				<p class="mt-1 text-xs text-muted-foreground/70">
					{searchQuery ? 'Try a different search term' : 'Upload files or create a folder to get started'}
				</p>
			</div>
		{:else}
			{#if viewMode === 'grid'}
				{#if folders.length > 0}
					<div class="mb-6">
						<p class="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">Folders</p>
						<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
							{#each folders as folder}
								<button
									class="group flex flex-col items-center gap-2 rounded-lg border border-border p-4 text-center transition-colors hover:bg-muted"
									onclick={() => openFolder(folder)}
									oncontextmenu={(e) => openContextMenu(e, 'folder', folder)}
								>
									{#if renameTarget?.type === 'folder' && (renameTarget.item as Folder).id === folder.id}
										<iconify-icon icon="solar:folder-linear" width="36" class="text-amber-500"></iconify-icon>
										<input
											type="text"
											bind:value={renameValue}
											onkeydown={handleRenameKeydown}
											onblur={cancelRename}
											class="w-full rounded border border-input bg-background px-1.5 py-0.5 text-xs text-center focus:outline-none focus:ring-1 focus:ring-ring"
											autofocus
										/>
									{:else}
										<iconify-icon icon="solar:folder-linear" width="36" class="text-amber-500"></iconify-icon>
										<span class="w-full truncate text-xs font-medium">{folder.name}</span>
									{/if}
								</button>
							{/each}
						</div>
					</div>
				{/if}

				{#if files.length > 0}
					<div>
						{#if folders.length > 0}
							<p class="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">Files</p>
						{/if}
						<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
							{#each files as file}
								<button
									class="group flex flex-col items-center gap-2 rounded-lg border border-border p-4 text-center transition-colors hover:bg-muted"
									onclick={() => openPreview(file)}
									oncontextmenu={(e) => openContextMenu(e, 'file', file)}
								>
									{#if renameTarget?.type === 'file' && (renameTarget.item as NuageFile).id === file.id}
										<iconify-icon icon={fileIcon(file.mime_type)} width="36" class={fileIconColor(file.mime_type)}></iconify-icon>
										<input
											type="text"
											bind:value={renameValue}
											onkeydown={handleRenameKeydown}
											onblur={cancelRename}
											class="w-full rounded border border-input bg-background px-1.5 py-0.5 text-xs text-center focus:outline-none focus:ring-1 focus:ring-ring"
											autofocus
										/>
									{:else}
										{#if file.mime_type.startsWith('image/')}
											<div class="flex h-16 w-full items-center justify-center overflow-hidden rounded">
												<img
													src={backend.downloadUrl(app.token, file.id)}
													alt={file.name}
													class="h-full w-full object-cover rounded"
												/>
											</div>
										{:else}
											<iconify-icon icon={fileIcon(file.mime_type)} width="36" class={fileIconColor(file.mime_type)}></iconify-icon>
										{/if}
										<span class="w-full truncate text-xs font-medium">{file.name}</span>
										<span class="text-[10px] text-muted-foreground">{formatSize(file.size)}</span>
									{/if}
								</button>
							{/each}
						</div>
					</div>
				{/if}

			{:else}
				<div class="overflow-x-auto">
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b border-border text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
								<th class="pb-3 pr-4">Name</th>
								<th class="hidden pb-3 pr-4 sm:table-cell">Size</th>
								<th class="hidden pb-3 pr-4 md:table-cell">Modified</th>
								<th class="pb-3 w-10"></th>
							</tr>
						</thead>
						<tbody>
							{#each folders as folder}
								<tr
									class="group cursor-pointer border-b border-border/50 transition-colors hover:bg-muted/50"
									oncontextmenu={(e) => openContextMenu(e, 'folder', folder)}
								>
									<td class="py-2.5 pr-4">
										<button class="flex items-center gap-3 text-left" onclick={() => openFolder(folder)}>
											<iconify-icon icon="solar:folder-linear" width="20" class="text-amber-500 shrink-0"></iconify-icon>
											{#if renameTarget?.type === 'folder' && (renameTarget.item as Folder).id === folder.id}
												<input
													type="text"
													bind:value={renameValue}
													onkeydown={handleRenameKeydown}
													onblur={cancelRename}
													class="rounded border border-input bg-background px-1.5 py-0.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
													autofocus
												/>
											{:else}
												<span class="truncate font-medium">{folder.name}</span>
											{/if}
										</button>
									</td>
									<td class="hidden py-2.5 pr-4 text-muted-foreground sm:table-cell">—</td>
									<td class="hidden py-2.5 pr-4 text-muted-foreground md:table-cell">{formatDate(folder.created_at)}</td>
									<td class="py-2.5">
										<button
											class="flex h-7 w-7 items-center justify-center rounded-md opacity-0 transition-opacity group-hover:opacity-100 hover:bg-muted"
											onclick={(e) => openContextMenu(e, 'folder', folder)}
											aria-label="More options"
										>
											<iconify-icon icon="solar:menu-dots-linear" width="16"></iconify-icon>
										</button>
									</td>
								</tr>
							{/each}
							{#each files as file}
								<tr
									class="group cursor-pointer border-b border-border/50 transition-colors hover:bg-muted/50"
									oncontextmenu={(e) => openContextMenu(e, 'file', file)}
								>
									<td class="py-2.5 pr-4">
										<button class="flex items-center gap-3 text-left" onclick={() => openPreview(file)}>
											<iconify-icon icon={fileIcon(file.mime_type)} width="20" class="{fileIconColor(file.mime_type)} shrink-0"></iconify-icon>
											{#if renameTarget?.type === 'file' && (renameTarget.item as NuageFile).id === file.id}
												<input
													type="text"
													bind:value={renameValue}
													onkeydown={handleRenameKeydown}
													onblur={cancelRename}
													class="rounded border border-input bg-background px-1.5 py-0.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
													autofocus
												/>
											{:else}
												<span class="truncate font-medium">{file.name}</span>
											{/if}
										</button>
									</td>
									<td class="hidden py-2.5 pr-4 text-muted-foreground sm:table-cell">{formatSize(file.size)}</td>
									<td class="hidden py-2.5 pr-4 text-muted-foreground md:table-cell">{formatDate(file.updated_at)}</td>
									<td class="py-2.5">
										<button
											class="flex h-7 w-7 items-center justify-center rounded-md opacity-0 transition-opacity group-hover:opacity-100 hover:bg-muted"
											onclick={(e) => openContextMenu(e, 'file', file)}
											aria-label="More options"
										>
											<iconify-icon icon="solar:menu-dots-linear" width="16"></iconify-icon>
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		{/if}
	</div>

	{#if contextMenu}
		<div
			class="fixed z-50 min-w-[160px] rounded-md border border-border bg-background py-1 shadow-lg"
			style="left: {contextMenu.x}px; top: {contextMenu.y}px;"
		>
			{#if contextMenu.type === 'file'}
				<button
					class="flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-muted"
					onclick={downloadItem}
				>
					<iconify-icon icon="solar:download-linear" width="16" class="text-muted-foreground"></iconify-icon>
					Download
				</button>
			{/if}
			<button
				class="flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-muted"
				onclick={startRename}
			>
				<iconify-icon icon="solar:pen-linear" width="16" class="text-muted-foreground"></iconify-icon>
				Rename
			</button>
			<div class="my-1 h-px bg-border"></div>
			<button
				class="flex w-full items-center gap-2 px-3 py-2 text-sm text-destructive transition-colors hover:bg-destructive/10"
				onclick={deleteItem}
			>
				<iconify-icon icon="solar:trash-bin-2-linear" width="16"></iconify-icon>
				Delete
			</button>
		</div>
	{/if}

	{#if previewFile}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" role="dialog">
			<button class="absolute inset-0" onclick={closePreview} aria-label="Close preview"></button>
			<div class="relative z-10 flex max-h-[90vh] max-w-[90vw] flex-col items-center rounded-lg bg-background p-4 shadow-xl">
				<div class="mb-3 flex w-full items-center justify-between">
					<h3 class="truncate text-sm font-medium">{previewFile.name}</h3>
					<div class="flex items-center gap-2">
						<a
							href={backend.downloadUrl(app.token, previewFile.id)}
							download={previewFile.name}
							class="flex h-8 w-8 items-center justify-center rounded-md transition-colors hover:bg-muted"
							aria-label="Download"
						>
							<iconify-icon icon="solar:download-linear" width="16"></iconify-icon>
						</a>
						<button
							class="flex h-8 w-8 items-center justify-center rounded-md transition-colors hover:bg-muted"
							onclick={closePreview}
							aria-label="Close"
						>
							<iconify-icon icon="solar:close-circle-linear" width="18"></iconify-icon>
						</button>
					</div>
				</div>
				<div class="overflow-auto">
					{#if previewFile.mime_type.startsWith('image/')}
						<img
							src={backend.downloadUrl(app.token, previewFile.id)}
							alt={previewFile.name}
							class="max-h-[75vh] max-w-full rounded object-contain"
						/>
					{:else if previewFile.mime_type === 'application/pdf'}
						<div class="flex flex-col items-center gap-3">
							<div class="flex items-center gap-2 rounded-md border border-border bg-muted/50 px-3 py-1.5">
								<button onclick={pdfPrev} disabled={pdfPageNum <= 1} aria-label="Previous page" class="flex h-7 w-7 items-center justify-center rounded transition-colors hover:bg-background disabled:opacity-30">
									<iconify-icon icon="mdi:chevron-left" width="18"></iconify-icon>
								</button>
								<span class="min-w-[4rem] text-center text-xs font-medium tabular-nums">{pdfPageNum} / {pdfTotalPages}</span>
								<button onclick={pdfNext} disabled={pdfPageNum >= pdfTotalPages} aria-label="Next page" class="flex h-7 w-7 items-center justify-center rounded transition-colors hover:bg-background disabled:opacity-30">
									<iconify-icon icon="mdi:chevron-right" width="18"></iconify-icon>
								</button>
								<div class="mx-1 h-4 w-px bg-border"></div>
								<button onclick={pdfZoomOut} aria-label="Zoom out" class="flex h-7 w-7 items-center justify-center rounded transition-colors hover:bg-background">
									<iconify-icon icon="mdi:minus" width="16"></iconify-icon>
								</button>
								<span class="text-xs tabular-nums">{Math.round(pdfScale * 100)}%</span>
								<button onclick={pdfZoomIn} aria-label="Zoom in" class="flex h-7 w-7 items-center justify-center rounded transition-colors hover:bg-background">
									<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
								</button>
							</div>
							<div class="max-h-[70vh] max-w-[80vw] overflow-auto rounded border border-border bg-white">
								<canvas bind:this={pdfCanvas} class="block"></canvas>
							</div>
						</div>
					{:else if previewFile.mime_type.startsWith('video/')}
						<video
							controls
							src={backend.downloadUrl(app.token, previewFile.id)}
							class="max-h-[75vh] max-w-full rounded"
						>
							<track kind="captions" />
						</video>
					{:else if previewFile.mime_type.startsWith('audio/')}
						<audio controls src={backend.downloadUrl(app.token, previewFile.id)} class="w-80">
							<track kind="captions" />
						</audio>
					{/if}
				</div>
			</div>
		</div>
	{/if}

	{#if showNewFolderDialog}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" role="dialog">
			<button class="absolute inset-0" onclick={() => { showNewFolderDialog = false; newFolderName = ''; }} aria-label="Close dialog"></button>
			<div class="relative z-10 w-full max-w-sm rounded-lg border border-border bg-background p-6 shadow-xl">
				<h3 class="text-lg font-semibold">New folder</h3>
				<p class="mt-1 text-sm text-muted-foreground">Enter a name for the new folder.</p>
				<input
					type="text"
					bind:value={newFolderName}
					onkeydown={handleNewFolderKeydown}
					placeholder="Folder name"
					class="mt-4 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
					autofocus
				/>
				<div class="mt-4 flex justify-end gap-2">
					<button
						class="inline-flex h-9 items-center justify-center rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent"
						onclick={() => { showNewFolderDialog = false; newFolderName = ''; }}
					>
						Cancel
					</button>
					<button
						class="inline-flex h-9 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
						disabled={!newFolderName.trim()}
						onclick={createFolder}
					>
						Create
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>
