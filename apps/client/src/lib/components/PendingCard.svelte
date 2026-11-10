<script lang="ts">
	import { Badge, Button, Card } from '@facile/muse';
	import type { PendingAction } from '$lib/backend';
	import { formatDateTime, formatSize } from '$lib/format';
	import { fileIcon, icons, nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	let {
		action,
		busy,
		onconfirm,
		ondismiss
	}: {
		action: PendingAction;
		busy: boolean;
		onconfirm: (action: PendingAction) => void;
		ondismiss: (action: PendingAction) => void;
	} = $props();

	const item = $derived(action.item);
	const sharing = $derived(action.kind === 'share');
	const itemIcon = $derived(
		item?.type === 'folder' ? icons.folder : fileIcon(item?.name ?? '', item?.mime_type ?? '')
	);

	/** Names the client that asked: the host of a client URL, else its id. */
	function clientName(clientId: string): string {
		if (clientId === 'bulle') return 'Bulle';
		try {
			return new URL(clientId).host;
		} catch {
			return clientId || '?';
		}
	}
</script>

<Card class="flex flex-col gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<Badge tone={sharing ? 'warning' : 'danger'} class="shrink-0 whitespace-nowrap">
			{t(sharing ? 'pending.kind.share' : 'pending.kind.trash')}
		</Badge>
		<span class="text-fc-xs text-fc-fg-muted">
			{t('pending.askedBy', { client: clientName(action.client_id) })} · {formatDateTime(action.created_at)}
		</span>
	</div>

	{#if item}
		<div class="flex items-center gap-3 rounded-fc-md bg-fc-surface p-4">
			<iconify-icon icon={itemIcon} width="24" height="24" class="shrink-0 text-fc-fg-muted"></iconify-icon>
			<div class="flex min-w-0 flex-col gap-1">
				<p class="break-words text-fc-md font-medium text-fc-fg">{item.name}</p>
				<p class="text-fc-sm text-fc-fg-muted">
					{item.type === 'folder' ? t('pending.folderAll') : formatSize(item.size ?? 0)}
				</p>
			</div>
		</div>
		{#if sharing}
			<p class="text-fc-sm text-fc-warning">
				{t('pending.shareWarn')}
				{action.expires_in_days
					? t('pending.shareDays', { count: action.expires_in_days })
					: t('pending.shareForever')}
			</p>
		{:else}
			<p class="text-fc-sm text-fc-fg-muted">{t('pending.trashNote')}</p>
		{/if}
	{:else}
		<p class="break-words text-fc-sm text-fc-fg-muted">{action.summary}</p>
	{/if}

	{#if action.state === 'changed'}
		<p class="text-fc-sm text-fc-warning">{t('pending.changed')}</p>
	{:else if action.state === 'missing'}
		<p class="text-fc-sm text-fc-warning">{t('pending.missing')}</p>
	{/if}

	<div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
		<Button variant="ghost" class="w-full sm:w-auto [@media(pointer:coarse)]:h-11" disabled={busy} onclick={() => ondismiss(action)}>
			{t('pending.dismiss')}
		</Button>
		{#if action.state === 'ready'}
			<Button
				variant={sharing ? 'primary' : 'danger'}
				class="w-full sm:w-auto [@media(pointer:coarse)]:h-11"
				disabled={busy}
				icon={sharing ? nuage.share : icons.remove}
				onclick={() => onconfirm(action)}
			>
				{t(sharing ? 'pending.confirmShare' : 'pending.confirmTrash')}
			</Button>
		{/if}
	</div>
</Card>
