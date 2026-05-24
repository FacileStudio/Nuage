<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import { backend, type NuageFile, type Folder } from '$lib/backend';

	const app = getContext<{ token: string; user: { id: string; email: string; name: string } | null }>('app');

	let files = $state<NuageFile[]>([]);
	let folders = $state<Folder[]>([]);
	let loading = $state(true);
	let actionLoading = $state<string | null>(null);

	onMount(async () => {
		await loadTrash();
	});

	async function loadTrash() {
		loading = true;
		try {
			const res = await backend.listTrash(app.token);
			files = res.files ?? [];
			folders = res.folders ?? [];
		} catch {
			files = [];
			folders = [];
		}
		loading = false;
	}

	async function restoreItem(type: 'file' | 'folder', id: number) {
		actionLoading = `restore-${type}-${id}`;
		try {
			await backend.restoreItem(app.token, type, id);
			await loadTrash();
		} catch {}
		actionLoading = null;
	}

	async function permanentDelete(type: 'file' | 'folder', id: number) {
		actionLoading = `delete-${type}-${id}`;
		try {
			await backend.permanentDelete(app.token, type, id);
			await loadTrash();
		} catch {}
		actionLoading = null;
	}

	async function emptyTrash() {
		if (!confirm('Permanently delete all items in trash? This cannot be undone.')) return;
		actionLoading = 'empty-all';
		try {
			const promises: Promise<any>[] = [];
			for (const file of files) {
				promises.push(backend.permanentDelete(app.token, 'file', file.id));
			}
			for (const folder of folders) {
				promises.push(backend.permanentDelete(app.token, 'folder', folder.id));
			}
			await Promise.all(promises);
			await loadTrash();
		} catch {}
		actionLoading = null;
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(1024));
		return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
	}

	function fileIcon(mime: string): string {
		if (mime.startsWith('image/')) return 'solar:gallery-linear';
		if (mime.startsWith('video/')) return 'solar:videocamera-record-linear';
		if (mime.startsWith('audio/')) return 'solar:music-note-2-linear';
		if (mime === 'application/pdf') return 'solar:document-linear';
		if (mime.includes('zip') || mime.includes('archive') || mime.includes('compressed')) return 'solar:zip-file-linear';
		return 'solar:file-linear';
	}

	function fileIconColor(mime: string): string {
		if (mime.startsWith('image/')) return 'text-emerald-600';
		if (mime.startsWith('video/')) return 'text-purple-600';
		if (mime.startsWith('audio/')) return 'text-pink-600';
		if (mime === 'application/pdf') return 'text-red-600';
		if (mime.includes('zip') || mime.includes('archive')) return 'text-amber-600';
		return 'text-blue-600';
	}

	let isEmpty = $derived(files.length === 0 && folders.length === 0);
</script>

<svelte:head>
	<title>Trash — Nuage</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="flex items-center justify-between border-b border-border px-4 py-4 md:px-8 md:py-5">
		<div>
			<h1 class="text-lg font-semibold">Trash</h1>
			<p class="mt-1 text-sm text-muted-foreground">Items will be permanently deleted after 30 days</p>
		</div>
		{#if !loading && !isEmpty}
			<button
				class="inline-flex h-9 items-center gap-2 rounded-md border border-destructive/30 bg-destructive/10 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/20 disabled:opacity-50"
				onclick={emptyTrash}
				disabled={actionLoading !== null}
			>
				<iconify-icon icon="solar:trash-bin-2-linear" width="16"></iconify-icon>
				Empty trash
			</button>
		{/if}
	</div>

	<div class="flex-1 overflow-auto px-4 py-4 md:px-8 md:py-6">
		{#if loading}
			<div class="flex h-64 items-center justify-center">
				<div class="h-6 w-6 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
			</div>
		{:else if isEmpty}
			<div class="flex h-64 flex-col items-center justify-center text-center">
				<iconify-icon icon="solar:trash-bin-2-linear" width="48" class="text-muted-foreground/40"></iconify-icon>
				<p class="mt-4 text-sm font-medium text-muted-foreground">Trash is empty</p>
				<p class="mt-1 text-xs text-muted-foreground/70">Deleted files and folders will appear here</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-border text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
							<th class="pb-3 pr-4">Name</th>
							<th class="hidden pb-3 pr-4 sm:table-cell">Size</th>
							<th class="hidden pb-3 pr-4 md:table-cell">Deleted</th>
							<th class="pb-3 w-32 text-right">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each folders as folder}
							<tr class="border-b border-border/50 transition-colors hover:bg-muted/50">
								<td class="py-2.5 pr-4">
									<div class="flex items-center gap-3">
										<iconify-icon icon="solar:folder-linear" width="20" class="text-amber-500 shrink-0"></iconify-icon>
										<span class="truncate font-medium">{folder.name}</span>
									</div>
								</td>
								<td class="hidden py-2.5 pr-4 text-muted-foreground sm:table-cell">—</td>
								<td class="hidden py-2.5 pr-4 text-muted-foreground md:table-cell">{formatDate(folder.created_at)}</td>
								<td class="py-2.5">
									<div class="flex items-center justify-end gap-1">
										<button
											class="inline-flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium transition-colors hover:bg-muted disabled:opacity-50"
											onclick={() => restoreItem('folder', folder.id)}
											disabled={actionLoading !== null}
											aria-label="Restore folder"
										>
											<iconify-icon icon="solar:restart-linear" width="14"></iconify-icon>
											Restore
										</button>
										<button
											class="inline-flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
											onclick={() => permanentDelete('folder', folder.id)}
											disabled={actionLoading !== null}
											aria-label="Permanently delete folder"
										>
											<iconify-icon icon="solar:trash-bin-2-linear" width="14"></iconify-icon>
											Delete
										</button>
									</div>
								</td>
							</tr>
						{/each}
						{#each files as file}
							<tr class="border-b border-border/50 transition-colors hover:bg-muted/50">
								<td class="py-2.5 pr-4">
									<div class="flex items-center gap-3">
										<iconify-icon icon={fileIcon(file.mime_type)} width="20" class="{fileIconColor(file.mime_type)} shrink-0"></iconify-icon>
										<span class="truncate font-medium">{file.name}</span>
									</div>
								</td>
								<td class="hidden py-2.5 pr-4 text-muted-foreground sm:table-cell">{formatSize(file.size)}</td>
								<td class="hidden py-2.5 pr-4 text-muted-foreground md:table-cell">{formatDate(file.updated_at)}</td>
								<td class="py-2.5">
									<div class="flex items-center justify-end gap-1">
										<button
											class="inline-flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium transition-colors hover:bg-muted disabled:opacity-50"
											onclick={() => restoreItem('file', file.id)}
											disabled={actionLoading !== null}
											aria-label="Restore file"
										>
											<iconify-icon icon="solar:restart-linear" width="14"></iconify-icon>
											Restore
										</button>
										<button
											class="inline-flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
											onclick={() => permanentDelete('file', file.id)}
											disabled={actionLoading !== null}
											aria-label="Permanently delete file"
										>
											<iconify-icon icon="solar:trash-bin-2-linear" width="14"></iconify-icon>
											Delete
										</button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>
