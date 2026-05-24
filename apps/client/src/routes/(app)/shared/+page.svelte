<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import { backend, type Share, type NuageFile } from '$lib/backend';

	const app = getContext<{ token: string; user: { id: string; email: string; name: string } | null }>('app');

	let shares = $state<Share[]>([]);
	let loading = $state(true);

	onMount(async () => {
		await loadShares();
	});

	async function loadShares() {
		loading = true;
		try {
			const res = await backend.listSharedWithMe(app.token);
			shares = res.shares ?? [];
		} catch {
			shares = [];
		}
		loading = false;
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

	function permissionBadgeClass(permission: string): string {
		if (permission === 'write' || permission === 'edit') return 'bg-amber-100 text-amber-800';
		return 'bg-blue-100 text-blue-800';
	}

	function openItem(share: Share) {
		if (share.file) {
			const url = backend.downloadUrl(app.token, share.file.id);
			window.open(url, '_blank');
		} else if (share.folder) {
			window.location.href = `/drive?folder=${share.folder.id}`;
		}
	}
</script>

<svelte:head>
	<title>Shared — Nuage</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<h1 class="text-lg font-semibold">Shared with me</h1>
		<p class="mt-1 text-sm text-muted-foreground">Files and folders others have shared with you</p>
	</div>

	<div class="flex-1 overflow-auto px-4 py-4 md:px-8 md:py-6">
		{#if loading}
			<div class="flex h-64 items-center justify-center">
				<div class="h-6 w-6 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
			</div>
		{:else if shares.length === 0}
			<div class="flex h-64 flex-col items-center justify-center text-center">
				<iconify-icon icon="solar:share-linear" width="48" class="text-muted-foreground/40"></iconify-icon>
				<p class="mt-4 text-sm font-medium text-muted-foreground">Nothing shared with you yet</p>
				<p class="mt-1 text-xs text-muted-foreground/70">When someone shares files or folders with you, they'll appear here</p>
			</div>
		{:else}
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
				{#each shares as share}
					<button
						class="flex items-start gap-4 rounded-lg border border-border p-4 text-left transition-colors hover:bg-muted"
						onclick={() => openItem(share)}
					>
						<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
							{#if share.file}
								<iconify-icon icon={fileIcon(share.file.mime_type)} width="22" class={fileIconColor(share.file.mime_type)}></iconify-icon>
							{:else}
								<iconify-icon icon="solar:folder-linear" width="22" class="text-amber-500"></iconify-icon>
							{/if}
						</div>
						<div class="min-w-0 flex-1">
							<p class="truncate text-sm font-medium">
								{share.file?.name ?? share.folder?.name ?? 'Untitled'}
							</p>
							<div class="mt-1.5 flex flex-wrap items-center gap-2">
								<span class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium {permissionBadgeClass(share.permission)}">
									{share.permission}
								</span>
								<span class="text-[11px] text-muted-foreground">
									{formatDate(share.created_at)}
								</span>
							</div>
							{#if share.file}
								<p class="mt-1 text-[11px] text-muted-foreground">{formatSize(share.file.size)}</p>
							{/if}
						</div>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
