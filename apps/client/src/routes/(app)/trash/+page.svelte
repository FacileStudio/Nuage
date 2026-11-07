<script lang="ts">
	import { getContext } from 'svelte';
	import { Button, ConfirmModal, EmptyState, Spinner, Table, toast } from '@facile/muse';
	import { backend, type TrashItem } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';
	import { formatDate, formatSize } from '$lib/format';
	import { fileIcon, icons, nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';
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
			toast.danger(t('trash.loadFailed'));
		}
		loading = false;
	}

	async function restoreItem(item: TrashItem) {
		busy = true;
		try {
			await backend.restoreItem(app.token, item.type, item.id);
			await loadTrash();
			toast.success(t('trash.restored', { name: item.name }));
		} catch {
			toast.danger(t('trash.restoreFailed', { name: item.name }));
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
			toast.success(t('trash.deletedForGood', { name: target.name }));
		} catch {
			toast.danger(t('trash.deleteFailed', { name: target.name }));
		}
		busy = false;
	}

	async function doEmptyTrash() {
		busy = true;
		try {
			await backend.emptyTrash(app.token);
			emptying = false;
			await loadTrash();
			toast.success(t('trash.emptied'));
		} catch {
			toast.danger(t('trash.emptyFailed'));
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
		days === 0 ? t('trash.expiresNow') : t('trash.expiresIn', { count: days });

	const description = $derived.by(() => {
		const keep =
			retentionDays > 0 ? t('trash.keepFor', { count: retentionDays }) : t('trash.keepForever');
		return reclaimable > 0
			? t('trash.reclaim', { keep, size: formatSize(reclaimable) })
			: t('trash.stillCounts', { keep });
	});
</script>

<svelte:head>
	<title>{t('trash.title')}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 md:px-8">
	<PageHeader title={t('trash.heading')} {description}>
		{#snippet actions()}
			{#if !loading && items.length > 0}
				<Button
					variant="danger"
					icon={icons.remove}
					class="[@media(pointer:coarse)]:h-11"
					disabled={busy}
					onclick={() => (emptying = true)}
				>
					{t('trash.empty')}
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if loading}
		<div class="flex h-64 items-center justify-center"><Spinner /></div>
	{:else if items.length === 0}
		<EmptyState
			icon={icons.remove}
			title={t('trash.emptyTitle')}
			description={t('trash.emptyText')}
		/>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">{t('common.name')}</th>
					<th scope="col" class="hidden sm:table-cell">{t('common.size')}</th>
					<th scope="col" class="hidden md:table-cell">{t('trash.deleted')}</th>
					<th scope="col" class="text-right"><span class="sr-only">{t('settings.tokens.actions')}</span></th>
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
									class="max-sm:size-11 max-sm:px-0 [@media(pointer:coarse)]:h-11"
									disabled={busy}
									onclick={() => restoreItem(item)}
								>
									<span class="max-sm:sr-only">{t('trash.restore')}</span>
								</Button>
								<!-- ghost-danger, not danger: a column of permanently red buttons turns a
								     list into a hazard sign (CHARTE §2). -->
								<Button
									variant="ghost-danger"
									size="sm"
									icon={icons.remove}
									class="max-sm:size-11 max-sm:px-0 [@media(pointer:coarse)]:h-11"
									disabled={busy}
									onclick={() => (deleteTarget = item)}
								>
									<span class="max-sm:sr-only">{t('common.delete')}</span>
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
	title={t('trash.emptyConfirmTitle')}
	description={t('trash.emptyConfirmText', { count: items.length })}
	confirmLabel={t('trash.empty')}
	tone="danger"
	onConfirm={doEmptyTrash}
/>

<ConfirmModal
	open={deleteTarget !== null}
	title={t('trash.deleteConfirmTitle')}
	description={deleteTarget ? t('trash.deleteConfirmText', { name: deleteTarget.name }) : ''}
	confirmLabel={t('trash.deleteForever')}
	tone="danger"
	onConfirm={doDelete}
	onCancel={() => (deleteTarget = null)}
/>
