<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { backend, type NuageFile, type Folder, type Share } from '$lib/backend';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { pushUndo } from '$lib/undo.svelte';

	const app = getContext<{ token: string; user: { id: string; email: string; name: string } | null; refreshQuota: () => void }>('app');

	let files = $state<NuageFile[]>([]);
	let folders = $state<Folder[]>([]);
	let breadcrumbs = $state<{ id: number | null; name: string }[]>([{ id: null, name: 'Files' }]);
	let currentFolderId = $state<number | null>(null);
	let viewMode = $state<'grid' | 'list'>('list');
	let loading = $state(true);
	let searchQuery = $state('');
	let searchTimeout = $state<ReturnType<typeof setTimeout> | null>(null);

	const CHUNK_SIZE = 5 * 1024 * 1024;
	const CHUNKED_THRESHOLD = 10 * 1024 * 1024;

	type FileUploadState = {
		name: string;
		size: number;
		loaded: number;
		chunksTotal: number;
		chunksDone: number;
		status: 'pending' | 'uploading' | 'done' | 'error';
	};

	let dragCounter = $state(0);
	let dragging = $derived(dragCounter > 0);
	let uploading = $state(false);
	let uploadQueue = $state<FileUploadState[]>([]);
	let currentUploadIndex = $state(0);
	let currentFile = $derived(uploadQueue[currentUploadIndex] as FileUploadState | undefined);
	let overallDone = $derived(uploadQueue.filter(f => f.status === 'done').length);
	let overallPercent = $derived.by(() => {
		const totalBytes = uploadQueue.reduce((s, f) => s + f.size, 0);
		if (totalBytes === 0) return 0;
		const loadedBytes = uploadQueue.reduce((s, f) => s + f.loaded, 0);
		return Math.round((loadedBytes / totalBytes) * 100);
	});
	let currentFilePercent = $derived.by(() => {
		if (!currentFile || currentFile.size === 0) return 0;
		return Math.round((currentFile.loaded / currentFile.size) * 100);
	});

	let contextMenu = $state<{ x: number; y: number; type: 'file' | 'folder'; item: NuageFile | Folder } | null>(null);
	let bgContextMenu = $state<{ x: number; y: number } | null>(null);
	let fileInputRef = $state<HTMLInputElement | null>(null);
	let renameTarget = $state<{ type: 'file' | 'folder'; item: NuageFile | Folder } | null>(null);
	let renameValue = $state('');

	let previewFile = $state<NuageFile | null>(null);
	let pdfPageNum = $state(1);
	let pdfTotalPages = $state(0);
	let pdfScale = $state(1.0);
	let pdfDoc = $state<any>(null);
	let pdfCanvas = $state<HTMLCanvasElement | null>(null);
	let pdfFitScale = $state(1.0);

	let selectedKeys = $state<string[]>([]);
	let lastClickedIndex = $state(-1);
	let showDeleteConfirm = $state(false);
	let bulkDeleting = $state(false);
	let deleteTargetKeys = $state<string[]>([]);
	let showSingleDeleteConfirm = $state(false);
	let singleDeleteTarget = $state<{ type: 'file' | 'folder'; item: NuageFile | Folder } | null>(null);
	let singleDeleting = $state(false);
	let selectMode = $state(false);
	let isMac = $state(false);
	let selectedMap = $derived.by(() => {
		const m: Record<string, true> = {};
		for (const k of selectedKeys) m[k] = true;
		return m;
	});
	let selectionCount = $derived(selectedKeys.length);
	let allItemsList = $derived([
		...folders.map(f => ({ type: 'folder' as const, id: f.id })),
		...files.map(f => ({ type: 'file' as const, id: f.id }))
	]);

	async function loadPdf(url: string) {
		pdfPageNum = 1;
		pdfScale = 1.0;
		pdfTotalPages = 0;
		pdfDoc = null;
		const pdfjsLib = await import('pdfjs-dist');
		pdfjsLib.GlobalWorkerOptions.workerSrc = '/pdf.worker.min.mjs';
		const doc = await pdfjsLib.getDocument(url).promise;
		pdfDoc = doc;
		pdfTotalPages = doc.numPages;
		const firstPage = await doc.getPage(1);
		const naturalViewport = firstPage.getViewport({ scale: 1.0 });
		const targetWidth = Math.min(window.innerWidth * 0.85 - 64, 900);
		pdfFitScale = targetWidth / naturalViewport.width;
		pdfScale = pdfFitScale;
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
	function pdfZoomIn() { pdfScale = Math.min(pdfScale + 0.25, 4); renderPdfPage(); }
	function pdfZoomOut() { pdfScale = Math.max(pdfScale - 0.25, 0.25); renderPdfPage(); }
	function pdfFitToWidth() { pdfScale = pdfFitScale; renderPdfPage(); }

	let dndItem = $state<{ type: 'file' | 'folder'; id: number } | null>(null);
	let dndTargetFolderId = $state<number | null | 'root'>(null);
	let dndCounter = $state<Record<string, number>>({});

	function isDndInternal(e: DragEvent): boolean {
		return e.dataTransfer?.types.includes('application/x-nuage-move') ?? false;
	}

	function handleItemDragStart(e: DragEvent, type: 'file' | 'folder', item: NuageFile | Folder) {
		if (selectMode) { e.preventDefault(); return; }
		dndItem = { type, id: item.id };
		e.dataTransfer!.effectAllowed = 'move';
		e.dataTransfer!.setData('application/x-nuage-move', JSON.stringify({ type, id: item.id }));
		e.dataTransfer!.setData('text/plain', item.name);
	}

	function handleItemDragEnd() {
		dndItem = null;
		dndTargetFolderId = null;
		dndCounter = {};
	}

	function isDescendantDrop(targetFolderId: number): boolean {
		if (!dndItem || dndItem.type !== 'folder') return false;
		if (dndItem.id === targetFolderId) return true;
		let fid: number | null = targetFolderId;
		const visited = new Set<number>();
		while (fid != null) {
			if (visited.has(fid)) break;
			visited.add(fid);
			if (fid === dndItem.id) return true;
			const f = folders.find(fo => fo.id === fid);
			if (f) fid = f.parent_id;
			else break;
		}
		return false;
	}

	function handleFolderDragOver(e: DragEvent, folderId: number) {
		if (!isDndInternal(e)) return;
		if (dndItem && dndItem.type === 'folder' && isDescendantDrop(folderId)) return;
		if (dndItem && dndItem.type === 'folder' && dndItem.id === folderId) return;
		e.preventDefault();
		e.dataTransfer!.dropEffect = 'move';
	}

	function handleFolderDragEnter(e: DragEvent, folderId: number) {
		if (!isDndInternal(e)) return;
		e.preventDefault();
		const key = `folder:${folderId}`;
		dndCounter = { ...dndCounter, [key]: (dndCounter[key] ?? 0) + 1 };
		if (dndItem && dndItem.type === 'folder' && (dndItem.id === folderId || isDescendantDrop(folderId))) return;
		dndTargetFolderId = folderId;
	}

	function handleFolderDragLeave(e: DragEvent, folderId: number) {
		if (!isDndInternal(e)) return;
		const key = `folder:${folderId}`;
		const next = (dndCounter[key] ?? 0) - 1;
		dndCounter = { ...dndCounter, [key]: next };
		if (next <= 0 && dndTargetFolderId === folderId) {
			dndTargetFolderId = null;
		}
	}

	async function handleFolderDrop(e: DragEvent, folderId: number) {
		if (!isDndInternal(e)) return;
		e.preventDefault();
		e.stopPropagation();
		dndTargetFolderId = null;
		dndCounter = {};
		if (!dndItem) return;
		if (dndItem.type === 'folder' && (dndItem.id === folderId || isDescendantDrop(folderId))) return;
		const { type, id } = dndItem;
		const oldFolderId = currentFolderId;
		try {
			if (type === 'file') {
				await backend.updateFile(app.token, id, { folder_id: folderId });
			} else {
				await backend.updateFolder(app.token, id, { parent_id: folderId });
			}
			pushUndo({
				label: 'Item moved',
				async execute() {
					if (type === 'file') await backend.updateFile(app.token, id, { folder_id: oldFolderId });
					else await backend.updateFolder(app.token, id, { parent_id: oldFolderId });
					await loadContents();
				}
			});
			await loadContents();
		} catch {}
		dndItem = null;
	}

	function handleBreadcrumbDragOver(e: DragEvent, folderId: number | null) {
		if (!isDndInternal(e)) return;
		if (dndItem && dndItem.type === 'folder' && folderId !== null && isDescendantDrop(folderId)) return;
		e.preventDefault();
		e.dataTransfer!.dropEffect = 'move';
	}

	function handleBreadcrumbDragEnter(e: DragEvent, folderId: number | null) {
		if (!isDndInternal(e)) return;
		e.preventDefault();
		const key = folderId === null ? 'root' : `bc:${folderId}`;
		dndCounter = { ...dndCounter, [key]: (dndCounter[key] ?? 0) + 1 };
		if (dndItem && dndItem.type === 'folder' && folderId !== null && isDescendantDrop(folderId)) return;
		dndTargetFolderId = folderId === null ? 'root' : folderId;
	}

	function handleBreadcrumbDragLeave(e: DragEvent, folderId: number | null) {
		if (!isDndInternal(e)) return;
		const key = folderId === null ? 'root' : `bc:${folderId}`;
		const next = (dndCounter[key] ?? 0) - 1;
		dndCounter = { ...dndCounter, [key]: next };
		const target = folderId === null ? 'root' : folderId;
		if (next <= 0 && dndTargetFolderId === target) {
			dndTargetFolderId = null;
		}
	}

	async function handleBreadcrumbDrop(e: DragEvent, folderId: number | null) {
		if (!isDndInternal(e)) return;
		e.preventDefault();
		e.stopPropagation();
		dndTargetFolderId = null;
		dndCounter = {};
		if (!dndItem) return;
		if (dndItem.type === 'folder' && folderId !== null && isDescendantDrop(folderId)) return;
		const currentParent = currentFolderId;
		const targetId = folderId;
		if (targetId === currentParent) { dndItem = null; return; }
		const { type, id } = dndItem;
		try {
			if (type === 'file') {
				await backend.updateFile(app.token, id, { folder_id: targetId });
			} else {
				await backend.updateFolder(app.token, id, { parent_id: targetId });
			}
			pushUndo({
				label: 'Item moved',
				async execute() {
					if (type === 'file') await backend.updateFile(app.token, id, { folder_id: currentParent });
					else await backend.updateFolder(app.token, id, { parent_id: currentParent });
					await loadContents();
				}
			});
			await loadContents();
		} catch {}
		dndItem = null;
	}


	let shareTarget = $state<{ type: 'file' | 'folder'; item: NuageFile | Folder } | null>(null);
	let shareLoading = $state(false);
	let existingShare = $state<Share | null>(null);
	let shareCopied = $state(false);
	let shareExpiration = $state('none');
	let shareCopiedTimeout = $state<ReturnType<typeof setTimeout> | null>(null);

	function startShare() {
		if (!contextMenu) return;
		shareTarget = { type: contextMenu.type, item: contextMenu.item };
		contextMenu = null;
		existingShare = null;
		shareLoading = true;
		shareExpiration = 'none';
		shareCopied = false;
		loadExistingShare();
	}

	async function loadExistingShare() {
		if (!shareTarget) return;
		try {
			const res = await backend.listMyShares(app.token);
			const match = res.shares.find((s) => {
				if (shareTarget!.type === 'file') return s.file_id === (shareTarget!.item as NuageFile).id;
				return s.folder_id === (shareTarget!.item as Folder).id;
			});
			existingShare = match ?? null;
			if (match?.expires_at) {
				shareExpiration = 'custom';
			}
		} catch {}
		shareLoading = false;
	}

	async function createShareLink() {
		if (!shareTarget) return;
		shareLoading = true;
		try {
			const data: { file_id?: number; folder_id?: number; permission?: string; expires_at?: string } = {};
			if (shareTarget.type === 'file') data.file_id = (shareTarget.item as NuageFile).id;
			else data.folder_id = (shareTarget.item as Folder).id;

			if (shareExpiration !== 'none') {
				const now = new Date();
				if (shareExpiration === '1d') now.setDate(now.getDate() + 1);
				else if (shareExpiration === '7d') now.setDate(now.getDate() + 7);
				else if (shareExpiration === '30d') now.setDate(now.getDate() + 30);
				data.expires_at = now.toISOString();
			}

			existingShare = await backend.createShare(app.token, data);
		} catch {}
		shareLoading = false;
	}

	async function removeShareLink() {
		if (!existingShare) return;
		shareLoading = true;
		try {
			await backend.deleteShare(app.token, existingShare.id);
			existingShare = null;
			shareExpiration = 'none';
		} catch {}
		shareLoading = false;
	}

	function copyShareLink() {
		if (!existingShare) return;
		const url = `${window.location.origin}/s/${existingShare.token}`;
		navigator.clipboard.writeText(url);
		shareCopied = true;
		if (shareCopiedTimeout) clearTimeout(shareCopiedTimeout);
		shareCopiedTimeout = setTimeout(() => { shareCopied = false; }, 2000);
	}

	function closeShareDialog() {
		shareTarget = null;
		existingShare = null;
		shareCopied = false;
		shareExpiration = 'none';
		if (shareCopiedTimeout) clearTimeout(shareCopiedTimeout);
	}

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
		isMac = navigator.platform.includes('Mac');
		document.addEventListener('click', closeContextMenu);
		document.addEventListener('keydown', handleGlobalKeydown);
		return () => {
			document.removeEventListener('click', closeContextMenu);
			document.removeEventListener('keydown', handleGlobalKeydown);
		};
	});

	async function loadContents() {
		loading = true;
		selectedKeys = [];
		lastClickedIndex = -1;
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
			goto(`/files?folder=${folderId}`);
		} else {
			goto('/files');
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
		if (isDndInternal(e)) return;
		e.preventDefault();
		dragCounter = 0;
		const droppedFiles = e.dataTransfer?.files;
		if (!droppedFiles?.length) return;
		await uploadFiles(droppedFiles);
	}

	function handleDragOver(e: DragEvent) {
		if (isDndInternal(e)) return;
		e.preventDefault();
	}

	function handleDragEnter(e: DragEvent) {
		if (isDndInternal(e)) return;
		e.preventDefault();
		dragCounter++;
	}

	function handleDragLeave(e: DragEvent) {
		if (isDndInternal(e)) return;
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
		uploadQueue = Array.from(fileList).map(f => ({
			name: f.name,
			size: f.size,
			loaded: 0,
			chunksTotal: f.size > CHUNKED_THRESHOLD ? Math.ceil(f.size / CHUNK_SIZE) : 0,
			chunksDone: 0,
			status: 'pending' as const
		}));
		currentUploadIndex = 0;

		for (let i = 0; i < fileList.length; i++) {
			const file = fileList[i];
			currentUploadIndex = i;
			uploadQueue[i].status = 'uploading';

			try {
				if (file.size > CHUNKED_THRESHOLD) {
					await uploadChunked(file, i);
				} else {
					await uploadSimple(file, i);
				}
				uploadQueue[i].loaded = file.size;
				uploadQueue[i].status = 'done';
			} catch {
				uploadQueue[i].status = 'error';
			}
		}

		uploading = false;
		uploadQueue = [];
		currentUploadIndex = 0;
		await loadContents();
		app.refreshQuota();
	}

	async function uploadSimple(file: File, index: number) {
		const formData = new FormData();
		formData.set('file', file);
		if (currentFolderId != null) formData.set('folder_id', String(currentFolderId));
		await backend.uploadFileWithProgress(app.token, formData, (loaded) => {
			uploadQueue[index].loaded = loaded;
		});
	}

	async function uploadChunked(file: File, index: number) {
		const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
		uploadQueue[index].chunksTotal = totalChunks;

		const session = await backend.initUpload(app.token, {
			file_name: file.name,
			mime_type: file.type || 'application/octet-stream',
			total_size: file.size,
			folder_id: currentFolderId
		});

		for (let part = 0; part < totalChunks; part++) {
			const start = part * CHUNK_SIZE;
			const end = Math.min(start + CHUNK_SIZE, file.size);
			const blob = file.slice(start, end);
			await backend.uploadChunk(app.token, session.session_id, part + 1, blob);
			uploadQueue[index].chunksDone = part + 1;
			uploadQueue[index].loaded = end;
		}

		await backend.completeUpload(app.token, session.session_id);
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
		e.stopPropagation();
		bgContextMenu = null;
		if (!isSelected(type, item.id)) {
			selectedKeys = [];
			lastClickedIndex = -1;
		}
		contextMenu = { x: e.clientX, y: e.clientY, type, item };
	}

	function openBgContextMenu(e: MouseEvent) {
		e.preventDefault();
		contextMenu = null;
		bgContextMenu = { x: e.clientX, y: e.clientY };
	}

	function closeContextMenu() {
		contextMenu = null;
		bgContextMenu = null;
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
		const { type, item } = renameTarget;
		const oldName = item.name;
		const newName = renameValue.trim();
		if (oldName === newName) { renameTarget = null; renameValue = ''; return; }
		try {
			if (type === 'file') {
				await backend.updateFile(app.token, (item as NuageFile).id, { name: newName });
			} else {
				await backend.updateFolder(app.token, (item as Folder).id, { name: newName });
			}
			const itemId = item.id;
			pushUndo({
				label: `Renamed to ${newName}`,
				async execute() {
					if (type === 'file') await backend.updateFile(app.token, itemId, { name: oldName });
					else await backend.updateFolder(app.token, itemId, { name: oldName });
					await loadContents();
				}
			});
		} catch {}
		renameTarget = null;
		renameValue = '';
		await loadContents();
	}

	function cancelRename() {
		renameTarget = null;
		renameValue = '';
	}

	function openBulkDeleteDialog() {
		deleteTargetKeys = [...selectedKeys];
		showDeleteConfirm = true;
	}

	function deleteItem() {
		if (!contextMenu) return;
		const { type, item } = contextMenu;
		contextMenu = null;
		if (selectionCount > 1) {
			openBulkDeleteDialog();
			return;
		}
		singleDeleteTarget = { type, item };
		showSingleDeleteConfirm = true;
	}

	async function doSingleDelete() {
		if (!singleDeleteTarget) return;
		const { type, item } = singleDeleteTarget;
		singleDeleting = true;
		try {
			if (type === 'file') {
				await backend.deleteFile(app.token, (item as NuageFile).id);
			} else {
				await backend.deleteFolder(app.token, (item as Folder).id);
			}
			const itemId = item.id;
			const itemName = item.name;
			pushUndo({
				label: `${itemName} moved to trash`,
				async execute() {
					await backend.restoreItem(app.token, type, itemId);
					await loadContents();
					app.refreshQuota();
				}
			});
		} catch {}
		singleDeleting = false;
		showSingleDeleteConfirm = false;
		singleDeleteTarget = null;
		selectedKeys = [];
		lastClickedIndex = -1;
		await loadContents();
		app.refreshQuota();
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

	function itemKey(type: 'file' | 'folder', id: number): string {
		return `${type}:${id}`;
	}

	function isSelected(type: 'file' | 'folder', id: number): boolean {
		return !!selectedMap[itemKey(type, id)];
	}

	function toggleSelect(type: 'file' | 'folder', id: number, index: number, e: MouseEvent) {
		const key = itemKey(type, id);
		const modKey = isMac ? e.metaKey : e.ctrlKey;
		if (e.shiftKey && lastClickedIndex >= 0) {
			const items = allItemsList;
			const start = Math.min(lastClickedIndex, index);
			const end = Math.max(lastClickedIndex, index);
			const base = modKey ? [...selectedKeys] : [];
			const existing = new Set(base);
			for (let i = start; i <= end; i++) {
				const k = itemKey(items[i].type, items[i].id);
				if (!existing.has(k)) base.push(k);
			}
			selectedKeys = base;
		} else if (modKey) {
			if (selectedKeys.includes(key)) {
				selectedKeys = selectedKeys.filter(k => k !== key);
			} else {
				selectedKeys = [...selectedKeys, key];
			}
			lastClickedIndex = index;
		} else {
			selectedKeys = [key];
			lastClickedIndex = index;
		}
	}

	function handleCheckboxClick(e: MouseEvent, type: 'file' | 'folder', id: number, index: number) {
		e.stopPropagation();
		const key = itemKey(type, id);
		if (e.shiftKey && lastClickedIndex >= 0) {
			const items = allItemsList;
			const start = Math.min(lastClickedIndex, index);
			const end = Math.max(lastClickedIndex, index);
			const next = [...selectedKeys];
			const existing = new Set(next);
			for (let i = start; i <= end; i++) {
				const k = itemKey(items[i].type, items[i].id);
				if (!existing.has(k)) next.push(k);
			}
			selectedKeys = next;
		} else if (selectedKeys.includes(key)) {
			selectedKeys = selectedKeys.filter(k => k !== key);
		} else {
			selectedKeys = [...selectedKeys, key];
		}
		lastClickedIndex = index;
	}

	function handleItemClick(e: MouseEvent, type: 'file' | 'folder', item: NuageFile | Folder, index: number) {
		if (!selectMode) {
			if (type === 'folder') openFolder(item as Folder);
			else openPreview(item as NuageFile);
			return;
		}
		const modKey = isMac ? e.metaKey : e.ctrlKey;
		if (modKey || e.shiftKey) {
			e.preventDefault();
			e.stopPropagation();
			toggleSelect(type, item.id, index, e);
			return;
		}
		const key = itemKey(type, item.id);
		if (selectedKeys.includes(key)) {
			selectedKeys = selectedKeys.filter(k => k !== key);
		} else {
			selectedKeys = [...selectedKeys, key];
		}
		lastClickedIndex = index;
	}

	function handleItemDblClick(e: MouseEvent, type: 'file' | 'folder', item: NuageFile | Folder) {
		if (!selectMode) return;
		e.preventDefault();
		e.stopPropagation();
		exitSelectMode();
		if (type === 'folder') openFolder(item as Folder);
		else openPreview(item as NuageFile);
	}

	function selectAll() {
		selectedKeys = allItemsList.map(i => itemKey(i.type, i.id));
	}

	function clearSelection() {
		selectedKeys = [];
		lastClickedIndex = -1;
	}

	function enterSelectMode() {
		selectMode = true;
	}

	function exitSelectMode() {
		selectMode = false;
		selectedKeys = [];
		lastClickedIndex = -1;
	}

	function handleGlobalKeydown(e: KeyboardEvent) {
		if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
		if (previewFile || showNewFolderDialog || showDeleteConfirm || shareTarget) return;
		const modKey = isMac ? e.metaKey : e.ctrlKey;
		if (modKey && e.key === 'a' && selectMode) {
			e.preventDefault();
			selectAll();
		} else if (e.key === 'Escape') {
			if (selectMode) exitSelectMode();
			else closeContextMenu();
		} else if ((e.key === 'Delete' || e.key === 'Backspace') && selectMode && selectionCount > 0) {
			e.preventDefault();
			openBulkDeleteDialog();
		}
	}

	async function bulkDelete() {
		const keys = [...deleteTargetKeys];
		if (keys.length === 0) return;
		bulkDeleting = true;
		const parsed = keys.map(k => {
			const [type, idStr] = k.split(':');
			return { type: type as 'file' | 'folder', id: Number(idStr) };
		});
		const promises: Promise<unknown>[] = [];
		for (const { type, id } of parsed) {
			if (type === 'file') promises.push(backend.deleteFile(app.token, id));
			else promises.push(backend.deleteFolder(app.token, id));
		}
		await Promise.allSettled(promises);
		const count = parsed.length;
		pushUndo({
			label: `${count} ${count === 1 ? 'item' : 'items'} moved to trash`,
			async execute() {
				const restores = parsed.map(({ type, id }) => backend.restoreItem(app.token, type, id));
				await Promise.allSettled(restores);
				await loadContents();
				app.refreshQuota();
			}
		});
		deleteTargetKeys = [];
		selectedKeys = [];
		lastClickedIndex = -1;
		bulkDeleting = false;
		showDeleteConfirm = false;
		selectMode = false;
		await loadContents();
		app.refreshQuota();
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
						<span
							class="font-medium truncate rounded px-1 py-0.5 transition-colors {dndItem && dndTargetFolderId === (crumb.id === null ? 'root' : crumb.id) ? 'bg-primary/15 text-primary ring-1 ring-primary/40' : ''}"
							role="listitem"
							ondragover={(e) => handleBreadcrumbDragOver(e, crumb.id)}
							ondragenter={(e) => handleBreadcrumbDragEnter(e, crumb.id)}
							ondragleave={(e) => handleBreadcrumbDragLeave(e, crumb.id)}
							ondrop={(e) => handleBreadcrumbDrop(e, crumb.id)}
						>{crumb.name}</span>
					{:else}
						<button
							class="truncate text-muted-foreground hover:text-foreground transition-colors rounded px-1 py-0.5 {dndItem && dndTargetFolderId === (crumb.id === null ? 'root' : crumb.id) ? 'bg-primary/15 !text-primary ring-1 ring-primary/40' : ''}"
							onclick={() => navigateToFolder(crumb.id)}
							ondragover={(e) => handleBreadcrumbDragOver(e, crumb.id)}
							ondragenter={(e) => handleBreadcrumbDragEnter(e, crumb.id)}
							ondragleave={(e) => handleBreadcrumbDragLeave(e, crumb.id)}
							ondrop={(e) => handleBreadcrumbDrop(e, crumb.id)}
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
			{#if selectMode}
				<button
					class="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
					onclick={exitSelectMode}
				>
					Cancel
				</button>
				<span class="text-sm text-muted-foreground">{selectionCount} selected</span>
				<div class="flex-1"></div>
				<button
					class="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
					onclick={() => selectionCount === allItemsList.length ? clearSelection() : selectAll()}
				>
					{selectionCount === allItemsList.length ? 'Deselect all' : 'Select all'}
				</button>
			{:else}
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

				<button
					class="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
					onclick={enterSelectMode}
				>
					<iconify-icon icon="solar:check-circle-linear" width="16"></iconify-icon>
					Select
				</button>
			{/if}
		</div>
	</div>

	{#if uploading && uploadQueue.length > 0}
		<div class="border-b border-border bg-muted/50 px-4 py-3 md:px-8">
			<div class="flex flex-col gap-2">
				{#if uploadQueue.length > 1}
					<div class="flex items-center justify-between text-xs text-muted-foreground">
						<span>Overall: {overallDone}/{uploadQueue.length} files</span>
						<span>{overallPercent}%</span>
					</div>
					<div class="h-1.5 w-full overflow-hidden rounded-full bg-border">
						<div
							class="h-full rounded-full bg-primary transition-[width] duration-200"
							style="width: {overallPercent}%"
						></div>
					</div>
				{/if}
				{#if currentFile && currentFile.status === 'uploading'}
					<div class="flex items-center gap-3">
						<div class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-primary border-t-transparent shrink-0"></div>
						<div class="flex-1 min-w-0">
							<div class="flex items-center justify-between gap-2">
								<span class="truncate text-sm font-medium">{currentFile.name}</span>
								<span class="shrink-0 text-xs tabular-nums text-muted-foreground">{currentFilePercent}%</span>
							</div>
							<div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-border">
								<div
									class="h-full rounded-full bg-primary transition-[width] duration-200"
									style="width: {currentFilePercent}%"
								></div>
							</div>
						</div>
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<div class="flex-1 overflow-auto px-4 py-4 md:px-8 md:py-6" oncontextmenu={openBgContextMenu}>
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
							{#each folders as folder, i}
								<button
									class="group relative flex flex-col items-center gap-2 rounded-lg border p-4 text-center transition-colors {dndTargetFolderId === folder.id ? 'border-primary bg-primary/10 ring-2 ring-primary/50 scale-[1.02]' : selectedMap[`folder:${folder.id}`] ? 'border-primary bg-primary/5 ring-1 ring-primary/30' : 'border-border hover:bg-muted'} {dndItem?.id === folder.id && dndItem?.type === 'folder' ? 'opacity-40' : ''}"
									onclick={(e) => handleItemClick(e, 'folder', folder, i)}
									ondblclick={(e) => handleItemDblClick(e, 'folder', folder)}
									oncontextmenu={(e) => openContextMenu(e, 'folder', folder)}
									draggable={!selectMode}
									ondragstart={(e) => handleItemDragStart(e, 'folder', folder)}
									ondragend={handleItemDragEnd}
									ondragover={(e) => handleFolderDragOver(e, folder.id)}
									ondragenter={(e) => handleFolderDragEnter(e, folder.id)}
									ondragleave={(e) => handleFolderDragLeave(e, folder.id)}
									ondrop={(e) => handleFolderDrop(e, folder.id)}
								>
									<div
										class="absolute top-2 left-2 z-10 flex h-5 w-5 cursor-pointer items-center justify-center rounded-full border-2 transition-all duration-200 {selectMode ? (selectedMap[`folder:${folder.id}`] ? 'opacity-100 scale-100 border-primary bg-primary text-primary-foreground' : 'opacity-100 scale-100 border-muted-foreground/40 bg-white/80') : 'opacity-0 scale-75 pointer-events-none'}"
										onclick={(e) => handleCheckboxClick(e, 'folder', folder.id, i)}
										role="checkbox"
										aria-checked={!!selectedMap[`folder:${folder.id}`]}
									>
										{#if selectedMap[`folder:${folder.id}`]}
											<iconify-icon icon="mdi:check" width="14"></iconify-icon>
										{/if}
									</div>
									{#if renameTarget?.type === 'folder' && (renameTarget.item as Folder).id === folder.id}
										<iconify-icon icon="solar:folder-linear" width="36" class="text-amber-500"></iconify-icon>
										<input
											type="text"
											bind:value={renameValue}
											onkeydown={handleRenameKeydown}
											onblur={cancelRename}
											onclick={(e) => e.stopPropagation()}
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
							{#each files as file, j}
								{@const fileIdx = folders.length + j}
								<button
									class="group relative flex flex-col items-center gap-2 rounded-lg border p-4 text-center transition-colors {selectedMap[`file:${file.id}`] ? 'border-primary bg-primary/5 ring-1 ring-primary/30' : 'border-border hover:bg-muted'} {dndItem?.id === file.id && dndItem?.type === 'file' ? 'opacity-40' : ''}"
									onclick={(e) => handleItemClick(e, 'file', file, fileIdx)}
									ondblclick={(e) => handleItemDblClick(e, 'file', file)}
									oncontextmenu={(e) => openContextMenu(e, 'file', file)}
									draggable={!selectMode}
									ondragstart={(e) => handleItemDragStart(e, 'file', file)}
									ondragend={handleItemDragEnd}
								>
									<div
										class="absolute top-2 left-2 z-10 flex h-5 w-5 cursor-pointer items-center justify-center rounded-full border-2 transition-all duration-200 {selectMode ? (selectedMap[`file:${file.id}`] ? 'opacity-100 scale-100 border-primary bg-primary text-primary-foreground' : 'opacity-100 scale-100 border-muted-foreground/40 bg-white/80') : 'opacity-0 scale-75 pointer-events-none'}"
										onclick={(e) => handleCheckboxClick(e, 'file', file.id, fileIdx)}
										role="checkbox"
										aria-checked={!!selectedMap[`file:${file.id}`]}
									>
										{#if selectedMap[`file:${file.id}`]}
											<iconify-icon icon="mdi:check" width="14"></iconify-icon>
										{/if}
									</div>
									{#if renameTarget?.type === 'file' && (renameTarget.item as NuageFile).id === file.id}
										<iconify-icon icon={fileIcon(file.mime_type)} width="36" class={fileIconColor(file.mime_type)}></iconify-icon>
										<input
											type="text"
											bind:value={renameValue}
											onkeydown={handleRenameKeydown}
											onblur={cancelRename}
											onclick={(e) => e.stopPropagation()}
											class="w-full rounded border border-input bg-background px-1.5 py-0.5 text-xs text-center focus:outline-none focus:ring-1 focus:ring-ring"
											autofocus
										/>
									{:else}
										{#if file.mime_type.startsWith('image/')}
											<div class="w-full aspect-[4/3] overflow-hidden rounded bg-muted/30">
												<img
													src={backend.downloadUrl(app.token, file.id)}
													alt={file.name}
													class="h-full w-full object-cover"
													loading="lazy"
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
								<th class="pb-3 pr-4">
									<div class="flex items-center">
										<div class="shrink-0 overflow-hidden transition-all duration-200 {selectMode ? 'w-8 opacity-100' : 'w-0 opacity-0'}">
											<div
												class="flex h-5 w-5 cursor-pointer items-center justify-center rounded-full border-2 transition-colors {selectionCount > 0 ? 'border-primary bg-primary text-primary-foreground' : 'border-muted-foreground/40 bg-background'}"
												onclick={() => selectionCount === allItemsList.length ? clearSelection() : selectAll()}
												role="checkbox"
												aria-checked={selectionCount > 0 && selectionCount === allItemsList.length}
												aria-label="Select all"
											>
												{#if selectionCount > 0 && selectionCount === allItemsList.length}
													<iconify-icon icon="mdi:check" width="14"></iconify-icon>
												{:else if selectionCount > 0}
													<iconify-icon icon="mdi:minus" width="14"></iconify-icon>
												{/if}
											</div>
										</div>
										Name
									</div>
								</th>
								<th class="hidden pb-3 pr-4 sm:table-cell">Size</th>
								<th class="hidden pb-3 pr-4 md:table-cell">Modified</th>
								<th class="pb-3 w-10"></th>
							</tr>
						</thead>
						<tbody>
							{#each folders as folder, i}
								<tr
									class="group cursor-pointer border-b border-border/50 transition-colors {dndTargetFolderId === folder.id ? 'bg-primary/10 outline outline-2 outline-primary/50' : selectedMap[`folder:${folder.id}`] ? 'bg-primary/5' : 'hover:bg-muted/50'} {dndItem?.id === folder.id && dndItem?.type === 'folder' ? 'opacity-40' : ''}"
									onclick={(e) => handleItemClick(e, 'folder', folder, i)}
									ondblclick={(e) => handleItemDblClick(e, 'folder', folder)}
									oncontextmenu={(e) => openContextMenu(e, 'folder', folder)}
									draggable={!selectMode}
									ondragstart={(e) => handleItemDragStart(e, 'folder', folder)}
									ondragend={handleItemDragEnd}
									ondragover={(e) => handleFolderDragOver(e, folder.id)}
									ondragenter={(e) => handleFolderDragEnter(e, folder.id)}
									ondragleave={(e) => handleFolderDragLeave(e, folder.id)}
									ondrop={(e) => handleFolderDrop(e, folder.id)}
								>
									<td class="py-2.5 pr-4">
										<div class="flex items-center">
											<div class="shrink-0 overflow-hidden transition-all duration-200 {selectMode ? 'w-8 opacity-100' : 'w-0 opacity-0'}" onclick={(e) => e.stopPropagation()}>
												<div
													class="flex h-5 w-5 cursor-pointer items-center justify-center rounded-full border-2 transition-colors {selectedMap[`folder:${folder.id}`] ? 'border-primary bg-primary text-primary-foreground' : 'border-muted-foreground/40 bg-background'}"
													onclick={(e) => handleCheckboxClick(e, 'folder', folder.id, i)}
													role="checkbox"
													aria-checked={!!selectedMap[`folder:${folder.id}`]}
												>
													{#if selectedMap[`folder:${folder.id}`]}
														<iconify-icon icon="mdi:check" width="14"></iconify-icon>
													{/if}
												</div>
											</div>
											<div class="flex items-center gap-3 min-w-0">
												<iconify-icon icon="solar:folder-linear" width="20" class="text-amber-500 shrink-0"></iconify-icon>
												{#if renameTarget?.type === 'folder' && (renameTarget.item as Folder).id === folder.id}
													<input
														type="text"
														bind:value={renameValue}
														onkeydown={handleRenameKeydown}
														onblur={cancelRename}
														onclick={(e) => e.stopPropagation()}
														class="rounded border border-input bg-background px-1.5 py-0.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
														autofocus
													/>
												{:else}
													<span class="truncate font-medium">{folder.name}</span>
												{/if}
											</div>
										</div>
									</td>
									<td class="hidden py-2.5 pr-4 text-muted-foreground sm:table-cell">—</td>
									<td class="hidden py-2.5 pr-4 text-muted-foreground md:table-cell">{formatDate(folder.created_at)}</td>
									<td class="py-2.5">
										<button
											class="flex h-7 w-7 items-center justify-center rounded-md opacity-0 transition-opacity group-hover:opacity-100 hover:bg-muted"
											onclick={(e) => { e.stopPropagation(); openContextMenu(e, 'folder', folder); }}
											aria-label="More options"
										>
											<iconify-icon icon="solar:menu-dots-linear" width="16"></iconify-icon>
										</button>
									</td>
								</tr>
							{/each}
							{#each files as file, j}
								{@const fileIdx = folders.length + j}
								<tr
									class="group cursor-pointer border-b border-border/50 transition-colors {selectedMap[`file:${file.id}`] ? 'bg-primary/5' : 'hover:bg-muted/50'} {dndItem?.id === file.id && dndItem?.type === 'file' ? 'opacity-40' : ''}"
									onclick={(e) => handleItemClick(e, 'file', file, fileIdx)}
									ondblclick={(e) => handleItemDblClick(e, 'file', file)}
									oncontextmenu={(e) => openContextMenu(e, 'file', file)}
									draggable={!selectMode}
									ondragstart={(e) => handleItemDragStart(e, 'file', file)}
									ondragend={handleItemDragEnd}
								>
									<td class="py-2.5 pr-4">
										<div class="flex items-center">
											<div class="shrink-0 overflow-hidden transition-all duration-200 {selectMode ? 'w-8 opacity-100' : 'w-0 opacity-0'}" onclick={(e) => e.stopPropagation()}>
												<div
													class="flex h-5 w-5 cursor-pointer items-center justify-center rounded-full border-2 transition-colors {selectedMap[`file:${file.id}`] ? 'border-primary bg-primary text-primary-foreground' : 'border-muted-foreground/40 bg-background'}"
													onclick={(e) => handleCheckboxClick(e, 'file', file.id, fileIdx)}
													role="checkbox"
													aria-checked={!!selectedMap[`file:${file.id}`]}
												>
													{#if selectedMap[`file:${file.id}`]}
														<iconify-icon icon="mdi:check" width="14"></iconify-icon>
													{/if}
												</div>
											</div>
											<div class="flex items-center gap-3 min-w-0">
												<iconify-icon icon={fileIcon(file.mime_type)} width="20" class="{fileIconColor(file.mime_type)} shrink-0"></iconify-icon>
												{#if renameTarget?.type === 'file' && (renameTarget.item as NuageFile).id === file.id}
													<input
														type="text"
														bind:value={renameValue}
														onkeydown={handleRenameKeydown}
														onblur={cancelRename}
														onclick={(e) => e.stopPropagation()}
														class="rounded border border-input bg-background px-1.5 py-0.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
														autofocus
													/>
												{:else}
													<span class="truncate font-medium">{file.name}</span>
												{/if}
											</div>
										</div>
									</td>
									<td class="hidden py-2.5 pr-4 text-muted-foreground sm:table-cell">{formatSize(file.size)}</td>
									<td class="hidden py-2.5 pr-4 text-muted-foreground md:table-cell">{formatDate(file.updated_at)}</td>
									<td class="py-2.5">
										<button
											class="flex h-7 w-7 items-center justify-center rounded-md opacity-0 transition-opacity group-hover:opacity-100 hover:bg-muted"
											onclick={(e) => { e.stopPropagation(); openContextMenu(e, 'file', file); }}
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
			onclick={(e) => e.stopPropagation()}
		>
			{#if contextMenu.type === 'folder' && selectionCount <= 1}
				<div class="px-3 py-2">
					<div class="text-sm font-medium truncate max-w-[200px]">{(contextMenu.item as Folder).name}</div>
					<div class="text-xs text-muted-foreground">{formatSize((contextMenu.item as Folder).size)}</div>
				</div>
				<div class="my-1 h-px bg-border"></div>
			{/if}
			{#if contextMenu.type === 'file' && selectionCount <= 1}
				<button
					class="flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-muted"
					onclick={downloadItem}
				>
					<iconify-icon icon="solar:download-linear" width="16" class="text-muted-foreground"></iconify-icon>
					Download
				</button>
			{/if}
			{#if selectionCount <= 1}
				<button
					class="flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-muted"
					onclick={startRename}
				>
					<iconify-icon icon="solar:pen-linear" width="16" class="text-muted-foreground"></iconify-icon>
					Rename
				</button>
			{/if}
			{#if selectionCount <= 1}
				<button
					class="flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-muted"
					onclick={startShare}
				>
					<iconify-icon icon="solar:share-linear" width="16" class="text-muted-foreground"></iconify-icon>
					Share
				</button>
			{/if}
			<div class="my-1 h-px bg-border"></div>
			<button
				class="flex w-full items-center gap-2 px-3 py-2 text-sm text-destructive transition-colors hover:bg-destructive/10"
				onclick={deleteItem}
			>
				<iconify-icon icon="solar:trash-bin-2-linear" width="16"></iconify-icon>
				{selectionCount > 1 ? `Delete ${selectionCount} items` : 'Delete'}
			</button>
		</div>
	{/if}

	{#if bgContextMenu}
		<div
			class="fixed z-50 min-w-[160px] rounded-md border border-border bg-background py-1 shadow-lg"
			style="left: {bgContextMenu.x}px; top: {bgContextMenu.y}px;"
			onclick={(e) => e.stopPropagation()}
		>
			<button
				class="flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-muted"
				onclick={() => { bgContextMenu = null; fileInputRef?.click(); }}
			>
				<iconify-icon icon="solar:upload-linear" width="16" class="text-muted-foreground"></iconify-icon>
				Upload
			</button>
			<button
				class="flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-muted"
				onclick={() => { bgContextMenu = null; showNewFolderDialog = true; }}
			>
				<iconify-icon icon="solar:add-folder-linear" width="16" class="text-muted-foreground"></iconify-icon>
				New folder
			</button>
		</div>
	{/if}

	<input type="file" multiple class="hidden" bind:this={fileInputRef} onchange={handleFileInput} />

	{#if previewFile}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" role="dialog">
			<button class="absolute inset-0" onclick={closePreview} aria-label="Close preview"></button>
			<div class="relative z-10 flex max-h-[95vh] w-[90vw] max-w-5xl flex-col rounded-lg bg-background p-4 shadow-xl">
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
				<div class="flex-1 overflow-auto flex flex-col items-center min-h-0">
					{#if previewFile.mime_type.startsWith('image/')}
						<img
							src={backend.downloadUrl(app.token, previewFile.id)}
							alt={previewFile.name}
							class="max-h-[75vh] max-w-full rounded object-contain"
						/>
					{:else if previewFile.mime_type === 'application/pdf'}
						<div class="flex flex-1 flex-col items-center gap-3 min-h-0 w-full">
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
								<span class="min-w-[3rem] text-center text-xs tabular-nums">{Math.round(pdfScale * 100)}%</span>
								<button onclick={pdfZoomIn} aria-label="Zoom in" class="flex h-7 w-7 items-center justify-center rounded transition-colors hover:bg-background">
									<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
								</button>
								<div class="mx-1 h-4 w-px bg-border"></div>
								<button onclick={pdfFitToWidth} aria-label="Fit to width" class="flex h-7 w-7 items-center justify-center rounded transition-colors hover:bg-background" title="Fit to width">
									<iconify-icon icon="solar:full-screen-linear" width="16"></iconify-icon>
								</button>
							</div>
							<div class="flex-1 overflow-auto rounded border border-border bg-white shadow-sm">
								<canvas bind:this={pdfCanvas} class="block mx-auto"></canvas>
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
						<audio controls src={backend.downloadUrl(app.token, previewFile.id)} class="w-full max-w-80">
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

	{#if selectionCount > 0}
		<div class="fixed bottom-6 left-1/2 z-40 flex -translate-x-1/2 items-center gap-3 rounded-lg border border-border bg-background px-4 py-2.5 shadow-xl">
			<span class="text-sm font-medium">{selectionCount} selected</span>
			<div class="h-4 w-px bg-border"></div>
			<button
				class="inline-flex items-center gap-1.5 rounded-md bg-red-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-red-700"
				onclick={openBulkDeleteDialog}
			>
				<iconify-icon icon="solar:trash-bin-2-linear" width="14"></iconify-icon>
				Delete
			</button>
			<div class="h-4 w-px bg-border"></div>
			<span class="hidden text-xs text-muted-foreground sm:inline">
				{isMac ? '⌘' : 'Ctrl+'}A all · {isMac ? '⌫' : 'Del'} delete · Esc exit
			</span>
			<button
				class="flex h-6 w-6 items-center justify-center rounded-md transition-colors hover:bg-muted"
				onclick={exitSelectMode}
				aria-label="Exit select mode"
			>
				<iconify-icon icon="solar:close-circle-linear" width="16" class="text-muted-foreground"></iconify-icon>
			</button>
		</div>
	{/if}

	<ConfirmDialog
		bind:open={showDeleteConfirm}
		title="Move to trash?"
		message="{deleteTargetKeys.length} {deleteTargetKeys.length === 1 ? 'item' : 'items'} will be moved to trash. You can restore them from the Trash page."
		confirmLabel="Move to trash"
		loading={bulkDeleting}
		onconfirm={bulkDelete}
	/>

	<ConfirmDialog
		bind:open={showSingleDeleteConfirm}
		title="Move to trash?"
		message="{singleDeleteTarget?.item.name ?? 'This item'} will be moved to trash. You can restore it from the Trash page."
		confirmLabel="Move to trash"
		loading={singleDeleting}
		onconfirm={doSingleDelete}
	/>

	{#if shareTarget}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" role="dialog">
			<button class="absolute inset-0" onclick={closeShareDialog} aria-label="Close dialog"></button>
			<div class="relative z-10 w-full max-w-md rounded-lg border border-border bg-background p-6 shadow-xl">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2 min-w-0">
						<iconify-icon icon="solar:share-linear" width="20" class="text-muted-foreground shrink-0"></iconify-icon>
						<h3 class="truncate text-lg font-semibold">Share "{shareTarget.item.name}"</h3>
					</div>
					<button
						class="flex h-8 w-8 items-center justify-center rounded-md transition-colors hover:bg-muted shrink-0"
						onclick={closeShareDialog}
						aria-label="Close"
					>
						<iconify-icon icon="solar:close-circle-linear" width="18"></iconify-icon>
					</button>
				</div>

				{#if shareLoading && !existingShare}
					<div class="mt-6 flex items-center justify-center py-8">
						<div class="h-5 w-5 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
					</div>
				{:else if !existingShare}
					<div class="mt-6 flex flex-col items-center gap-4 py-4">
						<div class="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
							<iconify-icon icon="solar:link-linear" width="24" class="text-muted-foreground"></iconify-icon>
						</div>
						<p class="text-sm text-muted-foreground">No public link exists for this {shareTarget.type}.</p>

						<div class="flex w-full items-center gap-2">
							<label for="share-expiration" class="text-sm font-medium shrink-0">Expires</label>
							<select
								id="share-expiration"
								bind:value={shareExpiration}
								class="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							>
								<option value="none">No expiration</option>
								<option value="1d">1 day</option>
								<option value="7d">7 days</option>
								<option value="30d">30 days</option>
							</select>
						</div>

						<button
							class="inline-flex h-9 w-full items-center justify-center gap-2 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
							onclick={createShareLink}
							disabled={shareLoading}
						>
							{#if shareLoading}
								<div class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent"></div>
							{:else}
								<iconify-icon icon="solar:link-linear" width="16"></iconify-icon>
							{/if}
							Create public link
						</button>
					</div>
				{:else}
					<div class="mt-5 flex flex-col gap-4">
						<div>
							<label for="share-url" class="mb-1.5 block text-sm font-medium">Public link</label>
							<div class="flex gap-2">
								<input
									id="share-url"
									type="text"
									readonly
									value="{window.location.origin}/s/{existingShare.token}"
									class="flex h-9 w-full rounded-md border border-input bg-muted/50 px-3 text-sm text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
								/>
								<button
									class="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-md border border-border bg-background px-3 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
									onclick={copyShareLink}
								>
									{#if shareCopied}
										<iconify-icon icon="solar:check-circle-linear" width="16" class="text-emerald-600"></iconify-icon>
										Copied!
									{:else}
										<iconify-icon icon="solar:copy-linear" width="16"></iconify-icon>
										Copy
									{/if}
								</button>
							</div>
						</div>

						{#if existingShare.expires_at}
							<p class="text-xs text-muted-foreground">
								Expires {new Date(existingShare.expires_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' })}
							</p>
						{:else}
							<p class="text-xs text-muted-foreground">This link does not expire.</p>
						{/if}

						<div class="border-t border-border pt-4">
							<button
								class="inline-flex h-9 w-full items-center justify-center gap-2 rounded-md border border-destructive/30 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
								onclick={removeShareLink}
								disabled={shareLoading}
							>
								{#if shareLoading}
									<div class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-destructive border-t-transparent"></div>
								{:else}
									<iconify-icon icon="solar:trash-bin-2-linear" width="16"></iconify-icon>
								{/if}
								Remove link
							</button>
						</div>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>
