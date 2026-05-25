<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import {
		backend,
		type NuageFile,
		type Folder,
		type PublicShareResponse,
		type PublicShareFilesResponse
	} from '$lib/backend';

	let loading = $state(true);
	let error = $state('');
	let share = $state<PublicShareResponse | null>(null);

	let folderFiles = $state<NuageFile[]>([]);
	let folderFolders = $state<Folder[]>([]);
	let folderLoading = $state(false);
	let breadcrumbs = $state<{ id: number | null; name: string }[]>([]);
	let currentFolderId = $state<number | null>(null);

	let token = $derived(page.params.token ?? '');

	onMount(async () => {
		try {
			share = await backend.getPublicShare(token);
			if (share.folder) {
				breadcrumbs = [{ id: null, name: share.folder.name }];
				await loadFolderContents();
			}
		} catch {
			error = 'This link has expired or doesn\u2019t exist';
		}
		loading = false;
	});

	async function loadFolderContents(folderId?: number) {
		folderLoading = true;
		try {
			const res = await backend.getPublicShareFiles(token, folderId);
			folderFiles = res.files ?? [];
			folderFolders = res.folders ?? [];
		} catch {
			folderFiles = [];
			folderFolders = [];
		}
		folderLoading = false;
	}

	async function openSubfolder(folder: Folder) {
		currentFolderId = folder.id;
		breadcrumbs = [...breadcrumbs, { id: folder.id, name: folder.name }];
		await loadFolderContents(folder.id);
	}

	async function navigateBreadcrumb(index: number) {
		const crumb = breadcrumbs[index];
		breadcrumbs = breadcrumbs.slice(0, index + 1);
		currentFolderId = crumb.id;
		await loadFolderContents(crumb.id ?? undefined);
	}

	function downloadFile(file: NuageFile) {
		const url = backend.publicDownloadUrl(token, file.id);
		const a = document.createElement('a');
		a.href = url;
		a.download = file.name;
		a.click();
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

	function mimeLabel(mime: string): string {
		if (mime.startsWith('image/')) return 'Image';
		if (mime.startsWith('video/')) return 'Video';
		if (mime.startsWith('audio/')) return 'Audio';
		if (mime === 'application/pdf') return 'PDF';
		if (mime.includes('zip') || mime.includes('archive') || mime.includes('compressed')) return 'Archive';
		if (mime.includes('spreadsheet') || mime.includes('excel') || mime.includes('csv')) return 'Spreadsheet';
		if (mime.includes('presentation') || mime.includes('powerpoint')) return 'Presentation';
		if (mime.includes('word') || mime.includes('document')) return 'Document';
		if (mime.startsWith('text/')) return 'Text';
		return 'File';
	}
</script>

<svelte:head>
	{#if share?.file}
		<title>{share.file.name} — Nuage</title>
	{:else if share?.folder}
		<title>{share.folder.name} — Nuage</title>
	{:else}
		<title>Shared — Nuage</title>
	{/if}
</svelte:head>

<div class="flex min-h-[100dvh] flex-col bg-background">
	<div class="flex items-center gap-2 px-6 pt-6">
		<iconify-icon icon="solar:cloud-bold-duotone" width="24" class="text-foreground"></iconify-icon>
		<span class="text-lg font-bold font-heading tracking-tight">Nuage</span>
	</div>

	<div class="flex flex-1 items-center justify-center px-4 py-8">
		{#if loading}
			<div class="flex flex-col items-center gap-4">
				<div class="h-8 w-8 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
			</div>
		{:else if error}
			<div class="flex flex-col items-center gap-4 text-center">
				<div class="flex h-16 w-16 items-center justify-center rounded-full bg-muted">
					<iconify-icon icon="solar:link-broken-linear" width="32" class="text-muted-foreground"></iconify-icon>
				</div>
				<div>
					<p class="text-lg font-semibold">{error}</p>
					<p class="mt-1 text-sm text-muted-foreground">Check the URL or ask the person who shared it with you.</p>
				</div>
			</div>
		{:else if share?.file}
			{@const file = share.file}
			<div class="w-full max-w-xl">
				<div class="rounded-xl border border-border bg-card p-6 shadow-sm">
					{#if file.mime_type.startsWith('image/')}
						<div class="mb-6 overflow-hidden rounded-lg border border-border bg-muted/30">
							<img
								src={backend.publicDownloadUrl(token, file.id)}
								alt={file.name}
								class="mx-auto max-h-[400px] object-contain"
							/>
						</div>
					{:else}
						<div class="mb-6 flex justify-center">
							<div class="flex h-24 w-24 items-center justify-center rounded-2xl bg-muted">
								<iconify-icon icon={fileIcon(file.mime_type)} width="48" class={fileIconColor(file.mime_type)}></iconify-icon>
							</div>
						</div>
					{/if}

					<div class="text-center">
						<h1 class="text-xl font-semibold break-all">{file.name}</h1>
						<div class="mt-2 flex items-center justify-center gap-3 text-sm text-muted-foreground">
							<span>{mimeLabel(file.mime_type)}</span>
							<span class="h-1 w-1 rounded-full bg-muted-foreground/40"></span>
							<span>{formatSize(file.size)}</span>
						</div>
					</div>

					<div class="mt-6">
						<a
							href={backend.publicDownloadUrl(token, file.id)}
							download={file.name}
							class="flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-6 py-3 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
						>
							<iconify-icon icon="solar:download-linear" width="18"></iconify-icon>
							Download
						</a>
					</div>
				</div>
			</div>
		{:else if share?.folder}
			<div class="w-full max-w-3xl">
				<div class="rounded-xl border border-border bg-card shadow-sm">
					<div class="border-b border-border px-6 py-4">
						<nav class="flex items-center gap-1.5 text-sm">
							{#each breadcrumbs as crumb, i}
								{#if i > 0}
									<iconify-icon icon="solar:alt-arrow-right-linear" width="14" class="text-muted-foreground shrink-0"></iconify-icon>
								{/if}
								{#if i === breadcrumbs.length - 1}
									<span class="font-medium truncate">{crumb.name}</span>
								{:else}
									<button
										class="truncate text-muted-foreground hover:text-foreground transition-colors"
										onclick={() => navigateBreadcrumb(i)}
									>
										{crumb.name}
									</button>
								{/if}
							{/each}
						</nav>
					</div>

					<div class="p-4">
						{#if folderLoading}
							<div class="flex h-48 items-center justify-center">
								<div class="h-6 w-6 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
							</div>
						{:else if folderFolders.length === 0 && folderFiles.length === 0}
							<div class="flex h-48 flex-col items-center justify-center text-center">
								<iconify-icon icon="solar:folder-open-linear" width="40" class="text-muted-foreground/40"></iconify-icon>
								<p class="mt-3 text-sm text-muted-foreground">This folder is empty</p>
							</div>
						{:else}
							<div class="overflow-x-auto">
								<table class="w-full text-sm">
									<thead>
										<tr class="border-b border-border text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
											<th class="pb-3 pr-4">Name</th>
											<th class="hidden pb-3 pr-4 sm:table-cell">Size</th>
											<th class="pb-3 w-10"></th>
										</tr>
									</thead>
									<tbody>
										{#each folderFolders as folder}
											<tr
												class="group cursor-pointer border-b border-border/50 transition-colors hover:bg-muted/50"
												onclick={() => openSubfolder(folder)}
											>
												<td class="py-2.5 pr-4">
													<div class="flex items-center gap-3">
														<iconify-icon icon="solar:folder-linear" width="20" class="text-amber-500 shrink-0"></iconify-icon>
														<span class="truncate font-medium">{folder.name}</span>
													</div>
												</td>
												<td class="hidden py-2.5 pr-4 text-muted-foreground sm:table-cell">{formatSize(folder.size)}</td>
												<td class="py-2.5"></td>
											</tr>
										{/each}
										{#each folderFiles as file}
											<tr class="group border-b border-border/50 transition-colors hover:bg-muted/50">
												<td class="py-2.5 pr-4">
													<div class="flex items-center gap-3">
														<iconify-icon icon={fileIcon(file.mime_type)} width="20" class="{fileIconColor(file.mime_type)} shrink-0"></iconify-icon>
														<span class="truncate font-medium">{file.name}</span>
													</div>
												</td>
												<td class="hidden py-2.5 pr-4 text-muted-foreground sm:table-cell">{formatSize(file.size)}</td>
												<td class="py-2.5">
													<button
														class="flex h-8 w-8 items-center justify-center rounded-md transition-colors hover:bg-muted"
														onclick={() => downloadFile(file)}
														aria-label="Download {file.name}"
													>
														<iconify-icon icon="solar:download-linear" width="16" class="text-muted-foreground"></iconify-icon>
													</button>
												</td>
											</tr>
										{/each}
									</tbody>
								</table>
							</div>
						{/if}
					</div>
				</div>
			</div>
		{/if}
	</div>

	<div class="flex items-center justify-center gap-2 px-6 pb-6 pt-2">
		<iconify-icon icon="solar:cloud-bold-duotone" width="14" class="text-muted-foreground/60"></iconify-icon>
		<span class="text-xs text-muted-foreground/60">Nuage</span>
	</div>
</div>
