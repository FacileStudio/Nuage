<script lang="ts">
	import { getContext } from 'svelte';
	import { Badge, Button, ConfirmModal, EmptyState, Spinner, Table, toast } from '@facile/muse';
	import { backend, type Share } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';
	import { formatDate } from '$lib/format';
	import { fileIcon, icons, nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';

	const app = getContext<{ token: string }>('app');
	const spaceStore = getSpaceStore();

	let shares = $state<Share[]>([]);
	let loading = $state(true);
	let revoking = $state<Share | null>(null);

	$effect(() => {
		const _spaceId = spaceStore.id;
		loadShares();
	});

	async function loadShares() {
		loading = true;
		try {
			const res = await backend.listMyShares(app.token, { space_id: spaceStore.id });
			shares = res.shares ?? [];
		} catch {
			shares = [];
			toast.danger(t('shared.loadFailed'));
		}
		loading = false;
	}

	async function revokeShare() {
		const target = revoking;
		if (!target) return;
		try {
			await backend.deleteShare(app.token, target.id);
			shares = shares.filter((s) => s.id !== target.id);
			revoking = null;
			toast.success(t('shared.revoked'));
		} catch {
			toast.danger(t('shared.revokeFailed'));
		}
	}

	async function copyLink(share: Share) {
		try {
			await navigator.clipboard.writeText(`${window.location.origin}/s/${share.token}`);
			toast.success(t('shared.copied'));
		} catch {
			toast.danger(t('shared.clipboardFailed'));
		}
	}

	const itemName = (share: Share) => share.file?.name ?? share.folder?.name ?? t('shared.untitled');

	const shareIcon = (share: Share) =>
		share.file ? fileIcon(share.file.name, share.file.mime_type) : icons.folder;

	const isExpired = (iso: string | null) => iso !== null && new Date(iso).getTime() < Date.now();
</script>

<svelte:head>
	<title>{t('shared.title')}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 md:px-8">
	<PageHeader
		title={t('shared.heading')}
		description={t('shared.description')}
	/>

	{#if loading}
		<div class="flex h-64 items-center justify-center"><Spinner /></div>
	{:else if shares.length === 0}
		<EmptyState
			icon={nuage.share}
			title={t('shared.emptyTitle')}
			description={t('shared.emptyText')}
		>
			<Button href="/files" icon={icons.folder}>{t('shared.goToFiles')}</Button>
		</EmptyState>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">{t('common.name')}</th>
					<th scope="col" class="hidden md:table-cell">{t('shared.link')}</th>
					<th scope="col" class="hidden sm:table-cell">{t('shared.expiry')}</th>
					<th scope="col" class="hidden lg:table-cell">{t('shared.created')}</th>
					<th scope="col" class="text-right"><span class="sr-only">{t('settings.tokens.actions')}</span></th>
				</tr>
			</thead>
			<tbody>
				{#each shares as share (share.id)}
					<tr>
						<td>
							<div class="flex min-w-0 items-center gap-3">
								<iconify-icon
									icon={shareIcon(share)}
									width="18"
									height="18"
									class="block shrink-0 text-fc-fg-muted"
								></iconify-icon>
								<span class="truncate font-medium">{itemName(share)}</span>
							</div>
						</td>
						<!-- A share token is a machine string, so it wears the mono face (CHARTE §3). -->
						<td class="hidden md:table-cell">
							<span class="block max-w-[200px] truncate font-fc-mono text-fc-xs text-fc-fg-muted">
								/s/{share.token}
							</span>
						</td>
						<td class="hidden sm:table-cell">
							{#if isExpired(share.expires_at)}
								<Badge tone="danger">{t('shared.expired')}</Badge>
							{:else if share.expires_at}
								<span class="text-fc-xs text-fc-fg-muted">{formatDate(share.expires_at)}</span>
							{:else}
								<span class="text-fc-xs text-fc-fg-muted">{t('shared.never')}</span>
							{/if}
						</td>
						<td class="hidden text-fc-fg-muted lg:table-cell">{formatDate(share.created_at)}</td>
						<td>
							<div class="flex items-center justify-end gap-1">
								<Button variant="ghost" size="sm" icon={icons.copy} onclick={() => copyLink(share)}>
									{t('shared.copy')}
								</Button>
								<Button
									variant="ghost-danger"
									size="sm"
									icon={nuage.shareOff}
									onclick={() => (revoking = share)}
								>
									{t('common.revoke')}
								</Button>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</div>

<!-- Revoking used to fire straight off the click. It is destructive and it is not undoable,
     so it goes through a confirmation that says what actually breaks (CHARTE §14). -->
<ConfirmModal
	open={revoking !== null}
	title={t('shared.revokeTitle')}
	description={revoking ? t('shared.revokeText', { name: itemName(revoking) }) : ''}
	confirmLabel={t('common.revoke')}
	tone="danger"
	onConfirm={revokeShare}
	onCancel={() => (revoking = null)}
/>
