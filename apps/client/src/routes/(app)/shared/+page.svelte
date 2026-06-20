<script lang="ts">
	import { getContext } from 'svelte';
	import { backend, type Share } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';

	const app = getContext<{ token: string; user: { id: string; email: string; name: string } | null }>('app');
	const spaceStore = getSpaceStore();

	let shares = $state<Share[]>([]);
	let loading = $state(true);
	let revoking = $state<number | null>(null);
	let copiedId = $state<number | null>(null);

	$effect(() => {
		const _spaceId = spaceStore.id;
		loadShares();
	});

	async function loadShares() {
		loading = true;
		try {
			const sid = spaceStore.id;
			const res = await backend.listMyShares(app.token, { space_id: sid });
			shares = res.shares ?? [];
		} catch {
			shares = [];
		}
		loading = false;
	}

	async function revokeShare(id: number) {
		revoking = id;
		try {
			await backend.deleteShare(app.token, id);
			shares = shares.filter((s) => s.id !== id);
		} catch {}
		revoking = null;
	}

	async function copyLink(share: Share) {
		const url = `${window.location.origin}/s/${share.token}`;
		await navigator.clipboard.writeText(url);
		copiedId = share.id;
		setTimeout(() => {
			if (copiedId === share.id) copiedId = null;
		}, 2000);
	}

	function shareUrl(share: Share): string {
		return `${window.location.origin}/s/${share.token}`;
	}

	function itemName(share: Share): string {
		return share.file?.name ?? share.folder?.name ?? 'Untitled';
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function formatExpiration(iso: string | null): string {
		if (!iso) return 'No expiration';
		return `Expires ${formatDate(iso)}`;
	}

	function isExpired(iso: string | null): boolean {
		if (!iso) return false;
		return new Date(iso).getTime() < Date.now();
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
</script>

<svelte:head>
	<title>Shared links — Nuage</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<h1 class="text-lg font-semibold">Shared links</h1>
		<p class="mt-1 text-sm text-muted-foreground">Manage your public share links</p>
	</div>

	<div class="flex-1 overflow-auto px-4 py-4 md:px-8 md:py-6">
		{#if loading}
			<div class="flex h-64 items-center justify-center">
				<div class="h-6 w-6 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
			</div>
		{:else if shares.length === 0}
			<div class="flex h-64 flex-col items-center justify-center text-center">
				<iconify-icon icon="solar:share-linear" width="48" class="text-muted-foreground/40"></iconify-icon>
				<p class="mt-4 text-sm font-medium text-muted-foreground">No shared links yet</p>
				<p class="mt-1 text-xs text-muted-foreground/70">Right-click a file and select Share to create a public link</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-border text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
							<th class="pb-3 pr-4">Name</th>
							<th class="hidden pb-3 pr-4 md:table-cell">Link</th>
							<th class="hidden pb-3 pr-4 sm:table-cell">Expiration</th>
							<th class="hidden pb-3 pr-4 lg:table-cell">Created</th>
							<th class="pb-3 w-44 text-right">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each shares as share}
							<tr class="border-b border-border/50 transition-colors hover:bg-muted/50">
								<td class="py-2.5 pr-4">
									<div class="flex items-center gap-3">
										{#if share.file}
											<iconify-icon icon={fileIcon(share.file.mime_type)} width="20" class="{fileIconColor(share.file.mime_type)} shrink-0"></iconify-icon>
										{:else}
											<iconify-icon icon="solar:folder-linear" width="20" class="text-amber-500 shrink-0"></iconify-icon>
										{/if}
										<span class="truncate font-medium">{itemName(share)}</span>
									</div>
								</td>
								<td class="hidden py-2.5 pr-4 md:table-cell">
									<span class="inline-block max-w-[200px] truncate rounded bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">
										/s/{share.token}
									</span>
								</td>
								<td class="hidden py-2.5 pr-4 sm:table-cell">
									{#if isExpired(share.expires_at)}
										<span class="text-xs font-medium text-destructive">Expired</span>
									{:else}
										<span class="text-xs text-muted-foreground">{formatExpiration(share.expires_at)}</span>
									{/if}
								</td>
								<td class="hidden py-2.5 pr-4 text-xs text-muted-foreground lg:table-cell">{formatDate(share.created_at)}</td>
								<td class="py-2.5">
									<div class="flex items-center justify-end gap-1">
										<button
											class="inline-flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium transition-colors hover:bg-muted disabled:opacity-50"
											onclick={() => copyLink(share)}
										>
											{#if copiedId === share.id}
												<iconify-icon icon="solar:check-read-linear" width="14" class="text-emerald-600"></iconify-icon>
												<span class="text-emerald-600">Copied!</span>
											{:else}
												<iconify-icon icon="solar:copy-linear" width="14"></iconify-icon>
												Copy
											{/if}
										</button>
										<button
											class="inline-flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
											onclick={() => revokeShare(share.id)}
											disabled={revoking !== null}
											aria-label="Revoke share link"
										>
											{#if revoking === share.id}
												<div class="h-3 w-3 animate-spin rounded-full border-2 border-destructive border-t-transparent"></div>
											{:else}
												<iconify-icon icon="solar:link-broken-linear" width="14"></iconify-icon>
											{/if}
											Revoke
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
