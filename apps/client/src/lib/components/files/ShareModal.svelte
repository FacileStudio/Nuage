<script lang="ts">
	import {
		Alert,
		Button,
		ConfirmModal,
		Field,
		Modal,
		SecretField,
		Select,
		Spinner,
		toast
	} from '@facile/muse';
	import { backend, type Share } from '$lib/backend';
	import { formatDateTime } from '$lib/format';
	import { nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	let {
		token,
		spaceId,
		target,
		onClose
	}: {
		token: string;
		spaceId: number | null;
		target: { type: 'file' | 'folder'; id: number; name: string };
		onClose: () => void;
	} = $props();

	let open = $state(true);
	let loading = $state(true);
	let working = $state(false);
	let share = $state<Share | null>(null);
	let expiry = $state('none');
	let revoking = $state(false);

	const EXPIRY_DAYS: Record<string, number> = { '1d': 1, '7d': 7, '30d': 30 };

	$effect(() => {
		void loadExisting();
	});

	async function loadExisting() {
		try {
			const res = await backend.listMyShares(token);
			share =
				res.shares.find((s) =>
					target.type === 'file' ? s.file_id === target.id : s.folder_id === target.id
				) ?? null;
		} catch {
			toast.danger(t('share.checkFailed'));
		}
		loading = false;
	}

	async function create() {
		working = true;
		try {
			const data: {
				file_id?: number;
				folder_id?: number;
				expires_at?: string;
				space_id?: number | null;
			} = target.type === 'file' ? { file_id: target.id } : { folder_id: target.id };
			if (spaceId != null) data.space_id = spaceId;

			const days = EXPIRY_DAYS[expiry];
			if (days) {
				const at = new Date();
				at.setDate(at.getDate() + days);
				data.expires_at = at.toISOString();
			}

			share = await backend.createShare(token, data);
			toast.success(t('share.created'));
		} catch {
			toast.danger(t('share.createFailed'));
		}
		working = false;
	}

	async function revoke() {
		if (!share) return;
		working = true;
		try {
			await backend.deleteShare(token, share.id);
			share = null;
			expiry = 'none';
			revoking = false;
			toast.success(t('share.revoked'));
		} catch {
			toast.danger(t('share.revokeFailed'));
		}
		working = false;
	}

	const url = $derived(share ? `${window.location.origin}/s/${share.token}` : '');
</script>

<Modal bind:open {onClose} title={t('share.title', { name: target.name })} showClose>
	{#if loading}
		<div class="flex h-32 items-center justify-center"><Spinner /></div>
	{:else if share}
		<div class="flex flex-col gap-4">
			<Alert tone="warning">{t(target.type === 'file' ? 'share.warnFile' : 'share.warnFolder')}</Alert>

			<!-- `sensitive={false}`: the URL is meant to be handed out, so masking it would be
			     theatre. SecretField is still the right control — it owns the copy button, the
			     confirmation swap and the live region (CHARTE §14). -->
			<SecretField label={t('share.publicLink')} value={url} sensitive={false} />

			<p class="text-fc-sm text-fc-fg-muted">
				{share.expires_at
					? t('share.expiresAt', { date: formatDateTime(share.expires_at) })
					: t('share.neverExpires')}
			</p>
		</div>
	{:else}
		<div class="flex flex-col gap-4">
			<p class="text-fc-sm text-fc-fg-muted">
				{t(target.type === 'file' ? 'share.noneFile' : 'share.noneFolder')}
			</p>
			<Field label={t('share.expires')} helper={t('share.expiresHelper')}>
				<Select bind:value={expiry}>
					<option value="none">{t('share.never')}</option>
					<option value="1d">{t('share.in1d')}</option>
					<option value="7d">{t('share.in7d')}</option>
					<option value="30d">{t('share.in30d')}</option>
				</Select>
			</Field>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex flex-wrap justify-end gap-2">
			{#if share}
				<Button
					variant="ghost-danger"
					icon={nuage.shareOff}
					disabled={working}
					onclick={() => (revoking = true)}
				>
					{t('share.revokeLink')}
				</Button>
				<Button onclick={() => (open = false)}>{t('common.done')}</Button>
			{:else if !loading}
				<Button variant="ghost" onclick={() => (open = false)}>{t('common.cancel')}</Button>
				<Button icon={nuage.link} disabled={working} onclick={create}>
					{working ? t('common.creating') : t('share.create')}
				</Button>
			{/if}
		</div>
	{/snippet}
</Modal>

<ConfirmModal
	bind:open={revoking}
	title={t('share.revokeTitle')}
	description={t(target.type === 'file' ? 'share.revokeFile' : 'share.revokeFolder')}
	confirmLabel={t('common.revoke')}
	tone="danger"
	onConfirm={revoke}
/>
