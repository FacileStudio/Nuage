<script lang="ts">
	import { getContext } from 'svelte';
	import { backend, type ActivityEntry } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';

	const app = getContext<{ token: string; user: { id: string; email: string; name: string } | null }>('app');
	const spaceStore = getSpaceStore();

	let activities = $state<ActivityEntry[]>([]);
	let loading = $state(true);
	let loadingMore = $state(false);
	let currentPage = $state(1);
	let total = $state(0);
	let perPage = 30;

	let hasMore = $derived(activities.length < total);

	$effect(() => {
		const _spaceId = spaceStore.id;
		loadActivity();
	});

	async function loadActivity() {
		loading = true;
		try {
			const sid = spaceStore.id;
			const res = await backend.listActivity(app.token, { page: 1, per_page: perPage, space_id: sid });
			activities = res.activities ?? [];
			total = res.total;
			currentPage = 1;
		} catch {
			activities = [];
			total = 0;
		}
		loading = false;
	}

	async function loadMore() {
		if (loadingMore || !hasMore) return;
		loadingMore = true;
		try {
			const nextPage = currentPage + 1;
			const sid = spaceStore.id;
			const res = await backend.listActivity(app.token, { page: nextPage, per_page: perPage, space_id: sid });
			activities = [...activities, ...(res.activities ?? [])];
			total = res.total;
			currentPage = nextPage;
		} catch {}
		loadingMore = false;
	}

	function eventIcon(eventType: string): string {
		const map: Record<string, string> = {
			'file.uploaded': 'solar:upload-linear',
			'file.deleted': 'solar:trash-bin-2-linear',
			'file.updated': 'solar:pen-linear',
			'file.versioned': 'solar:layers-linear',
			'file.restored': 'solar:restart-linear',
			'file.permanently_deleted': 'solar:trash-bin-2-linear',
			'folder.created': 'solar:add-folder-linear',
			'folder.updated': 'solar:pen-linear',
			'folder.deleted': 'solar:trash-bin-2-linear',
			'folder.restored': 'solar:restart-linear',
			'folder.permanently_deleted': 'solar:trash-bin-2-linear',
			'share.created': 'solar:share-linear',
			'share.revoked': 'solar:link-broken-linear'
		};
		return map[eventType] ?? 'solar:bolt-linear';
	}

	function eventColor(eventType: string): string {
		if (eventType.includes('uploaded') || eventType.includes('created')) return 'text-emerald-600 bg-emerald-100';
		if (eventType.includes('permanently_deleted')) return 'text-red-700 bg-red-100';
		if (eventType.includes('deleted')) return 'text-red-500 bg-red-50';
		if (eventType.includes('updated') || eventType.includes('versioned')) return 'text-amber-600 bg-amber-100';
		if (eventType.includes('restored')) return 'text-sky-600 bg-sky-100';
		if (eventType === 'share.created') return 'text-blue-600 bg-blue-100';
		if (eventType === 'share.revoked') return 'text-orange-600 bg-orange-100';
		return 'text-muted-foreground bg-muted';
	}

	function eventLabel(eventType: string): string {
		const map: Record<string, string> = {
			'file.uploaded': 'Uploaded',
			'file.deleted': 'Deleted',
			'file.updated': 'Updated',
			'file.versioned': 'New version',
			'file.restored': 'Restored',
			'file.permanently_deleted': 'Permanently deleted',
			'folder.created': 'Created folder',
			'folder.updated': 'Updated folder',
			'folder.deleted': 'Deleted folder',
			'folder.restored': 'Restored folder',
			'folder.permanently_deleted': 'Permanently deleted folder',
			'share.created': 'Shared',
			'share.revoked': 'Revoked share'
		};
		return map[eventType] ?? eventType;
	}

	function relativeTime(iso: string): string {
		const now = Date.now();
		const then = new Date(iso).getTime();
		const diff = now - then;

		const seconds = Math.floor(diff / 1000);
		if (seconds < 60) return 'just now';
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		if (days === 1) return 'yesterday';
		if (days < 7) return `${days}d ago`;
		const weeks = Math.floor(days / 7);
		if (weeks < 5) return `${weeks}w ago`;
		const months = Math.floor(days / 30);
		if (months < 12) return `${months}mo ago`;
		const years = Math.floor(days / 365);
		return `${years}y ago`;
	}

	function dateHeading(iso: string): string {
		const date = new Date(iso);
		const now = new Date();
		const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
		const target = new Date(date.getFullYear(), date.getMonth(), date.getDate());
		const diff = today.getTime() - target.getTime();
		const days = Math.floor(diff / (1000 * 60 * 60 * 24));

		if (days === 0) return 'Today';
		if (days === 1) return 'Yesterday';
		if (days < 7) return date.toLocaleDateString('en-US', { weekday: 'long' });
		return date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
	}

	function groupByDate(entries: ActivityEntry[]): { heading: string; items: ActivityEntry[] }[] {
		const groups: { heading: string; items: ActivityEntry[] }[] = [];
		let currentHeading = '';

		for (const entry of entries) {
			const heading = dateHeading(entry.created_at);
			if (heading !== currentHeading) {
				currentHeading = heading;
				groups.push({ heading, items: [] });
			}
			groups[groups.length - 1].items.push(entry);
		}

		return groups;
	}

	let grouped = $derived(groupByDate(activities));
</script>

<svelte:head>
	<title>Activity — Nuage</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<h1 class="text-lg font-semibold">Activity</h1>
		<p class="mt-1 text-sm text-muted-foreground">Recent actions across your files and folders</p>
	</div>

	<div class="flex-1 overflow-auto px-4 py-4 md:px-8 md:py-6">
		{#if loading}
			<div class="flex h-64 items-center justify-center">
				<div class="h-6 w-6 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
			</div>
		{:else if activities.length === 0}
			<div class="flex h-64 flex-col items-center justify-center text-center">
				<iconify-icon icon="solar:history-linear" width="48" class="text-muted-foreground/40"></iconify-icon>
				<p class="mt-4 text-sm font-medium text-muted-foreground">No activity yet</p>
				<p class="mt-1 text-xs text-muted-foreground/70">Actions like uploads, edits, and shares will show up here</p>
			</div>
		{:else}
			<div class="mx-auto max-w-2xl">
				{#each grouped as group}
					<div class="mb-6">
						<h2 class="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">{group.heading}</h2>
						<div class="space-y-3">
							{#each group.items as entry}
								<div class="flex items-start gap-3">
									<div class="mt-[13px] flex h-5 w-5 shrink-0 items-center justify-center rounded-full {eventColor(entry.event_type)}">
										<iconify-icon icon={eventIcon(entry.event_type)} width="12"></iconify-icon>
									</div>
									<div class="min-w-0 flex-1 rounded-lg border border-border/60 bg-background p-3 transition-colors hover:bg-muted/30">
										<div class="flex items-start justify-between gap-3">
											<div class="min-w-0 flex-1">
												<p class="text-sm">
													<span class="font-medium">{eventLabel(entry.event_type)}</span>
													{#if entry.resource_name}
														<span class="text-muted-foreground"> — </span>
														<span class="font-medium text-foreground">{entry.resource_name}</span>
													{/if}
												</p>
											</div>
											<span class="shrink-0 text-xs text-muted-foreground">{relativeTime(entry.created_at)}</span>
										</div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/each}

				{#if hasMore}
					<div class="flex justify-center py-4">
						<button
							class="inline-flex h-9 items-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-muted disabled:opacity-50"
							onclick={loadMore}
							disabled={loadingMore}
						>
							{#if loadingMore}
								<div class="h-4 w-4 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
								Loading...
							{:else}
								<iconify-icon icon="solar:alt-arrow-down-linear" width="16"></iconify-icon>
								Load more
							{/if}
						</button>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>
