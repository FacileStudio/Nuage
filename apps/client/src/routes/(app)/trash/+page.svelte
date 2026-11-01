<script lang="ts">
	import { getContext } from 'svelte';
	import { Button, ConfirmModal, EmptyState, Spinner, Table, toast } from '@facile/muse';
	import { backend, type TrashItem } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';
	import { formatDate, formatSize } from '$lib/format';
	import { fileIcon, icons, nuage } from '$lib/icons';
	import PageHeader from '$lib/components/PageHeader.svelte';

	const app = getContext<{ token: string }>('app');
	const spaceStore = getSpaceStore();

	let items = $state<TrashItem[]>([]);
	let retentionDays = $state(0);
	let loading = $state(true);
	let busy = $state(false);
	let emptying = $state(false);
	let deleteTarget = $state<TrashItem | null>(null);

	$effect(() => {
		const _spaceId = spaceStore.id;
		loadTrash();
	});

	async function loadTrash() {
		loading = true;
		try {
			const res = await backend.listTrash(app.token, { space_id: spaceStore.id });
			items = res.items ?? [];
			retentionDays = res.retention_days ?? 0;
		} catch {
			items = [];
			toast.danger('Could not load the trash.');
		}
		loading = false;
	}

	async function restoreItem(item: TrashItem) {
		busy = true;
		try {
			await backend.restoreItem(app.token, item.type, item.id);
			await loadTrash();
			toast.success(`Restored “${item.name}”.`);
		} catch {
			toast.danger(`Could not restore “${item.name}”.`);
		}
		busy = false;
	}

	async function doDelete() {
		const target = deleteTarget;
		if (!target) return;
		busy = true;
		try {
			await backend.permanentDelete(app.token, target.type, target.id);
			deleteTarget = null;
			await loadTrash();
			toast.success(`Deleted “${target.name}” for good.`);
		} catch {
			toast.danger(`Could not delete “${target.name}”.`);
		}
		busy = false;
	}

	async function doEmptyTrash() {
		busy = true;
		try {
			await backend.emptyTrash(app.token);
			emptying = false;
			await loadTrash();
			toast.success('Trash emptied.');
		} catch {
			toast.danger('Could not empty the trash.');
		}
		busy = false;
	}

	const itemIcon = (item: TrashItem) =>
		item.type === 'folder' ? icons.folder : fileIcon(item.name, item.mime_type ?? '');

	const reclaimable = $derived(
		items.reduce((sum, item) => sum + (item.type === 'file' ? (item.size ?? 0) : 0), 0)
	);

	const DAY_MS = 86_400_000;

	const daysLeft = (item: TrashItem): number | null =>
		item.expires_at
			? Math.max(0, Math.ceil((Date.parse(item.expires_at) - Date.now()) / DAY_MS))
			: null;

	const expiryLabel = (days: number) =>
		days === 0
			? 'Removed for good within the hour'
			: `Removed for good in ${days} ${days === 1 ? 'day' : 'days'}`;

	const description = $derived.by(() => {
		const keep =
			retentionDays > 0
				? `Deleted items are removed for good after ${retentionDays} ${retentionDays === 1 ? 'day' : 'days'}.`
				: 'Deleted items stay here until you remove them.';
		return reclaimable > 0
			? `${keep} Emptying the trash would give back at least ${formatSize(reclaimable)}.`
			: `${keep} Until then they still count against your storage.`;
	});
</script>

<svelte:head>
	<title>Trash — Nuage</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 md:px-8">
	<PageHeader title="Trash" {description}>
		{#snippet actions()}
			{#if !loading && items.length > 0}
				<Button variant="danger" icon={icons.remove} disabled={busy} onclick={() => (emptying = true)}>
					Empty trash
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if loading}
		<div class="flex h-64 items-center justify-center"><Spinner /></div>
	{:else if items.length === 0}
		<EmptyState
			icon={icons.remove}
			title="Trash is empty"
			description="Deleted files and folders land here, so a wrong click is never the end of the story."
		/>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">Name</th>
					<th scope="col" class="hidden sm:table-cell">Size</th>
					<th scope="col" class="hidden md:table-cell">Deleted</th>
					<th scope="col" class="text-right"><span class="sr-only">Actions</span></th>
				</tr>
			</thead>
			<tbody>
				{#each items as item (`${item.type}-${item.id}`)}
					<tr>
						<td>
							<div class="flex min-w-0 items-center gap-3">
								<iconify-icon
									icon={itemIcon(item)}
									width="18"
									height="18"
									class="block shrink-0 text-fc-fg-muted"
								></iconify-icon>
								<div class="min-w-0">
									<span class="block truncate font-medium">{item.name}</span>
									{#if daysLeft(item) !== null}
										{@const left = daysLeft(item) as number}
										<span class="block text-fc-xs {left <= 3 ? 'text-fc-warning' : 'text-fc-fg-muted'}">
											{expiryLabel(left)}
										</span>
									{/if}
								</div>
							</div>
						</td>
						<td class="hidden text-fc-fg-muted sm:table-cell">
							{item.type === 'file' && item.size != null ? formatSize(item.size) : '—'}
						</td>
						<td class="hidden text-fc-fg-muted md:table-cell">{formatDate(item.deleted_at)}</td>
						<td>
							<div class="flex items-center justify-end gap-1">
								<Button
									variant="ghost"
									size="sm"
									icon={nuage.restore}
									disabled={busy}
									onclick={() => restoreItem(item)}
								>
									Restore
								</Button>
								<!-- ghost-danger, not danger: a column of permanently red buttons turns a
								     list into a hazard sign (CHARTE §2). -->
								<Button
									variant="ghost-danger"
									size="sm"
									icon={icons.remove}
									disabled={busy}
									onclick={() => (deleteTarget = item)}
								>
									Delete
								</Button>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</div>

<ConfirmModal
	bind:open={emptying}
	title="Empty the trash?"
	description="All {items.length} {items.length === 1 ? 'item' : 'items'} go for good, and the files come off the server. There is no undo."
	confirmLabel="Empty trash"
	tone="danger"
	onConfirm={doEmptyTrash}
/>

<ConfirmModal
	open={deleteTarget !== null}
	title="Delete permanently?"
	description={deleteTarget
		? `“${deleteTarget.name}” comes off the server for good. There is no undo, and any share link pointing at it stops working.`
		: ''}
	confirmLabel="Delete forever"
	tone="danger"
	onConfirm={doDelete}
	onCancel={() => (deleteTarget = null)}
/>
