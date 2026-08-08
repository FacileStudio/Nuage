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
			toast.danger('Could not check for an existing link.');
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
			toast.success('Public link created.');
		} catch {
			toast.danger('Could not create that link.');
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
			toast.success('Link revoked.');
		} catch {
			toast.danger('Could not revoke that link.');
		}
		working = false;
	}

	const url = $derived(share ? `${window.location.origin}/s/${share.token}` : '');
</script>

<Modal bind:open {onClose} title="Share “{target.name}”" showClose>
	{#if loading}
		<div class="flex h-32 items-center justify-center"><Spinner /></div>
	{:else if share}
		<div class="flex flex-col gap-4">
			<Alert tone="warning">
				Anyone with this link can open the {target.type} without signing in.
			</Alert>

			<!-- `sensitive={false}`: the URL is meant to be handed out, so masking it would be
			     theatre. SecretField is still the right control — it owns the copy button, the
			     confirmation swap and the live region (CHARTE §14). -->
			<SecretField label="Public link" value={url} sensitive={false} />

			<p class="text-fc-sm text-fc-fg-muted">
				{share.expires_at ? `Expires ${formatDateTime(share.expires_at)}.` : 'This link never expires.'}
			</p>
		</div>
	{:else}
		<div class="flex flex-col gap-4">
			<p class="text-fc-sm text-fc-fg-muted">
				There is no public link for this {target.type} yet.
			</p>
			<Field label="Expires" helper="A link that never lapses is the one nobody remembers to revoke.">
				<Select bind:value={expiry}>
					<option value="none">Never</option>
					<option value="1d">In 1 day</option>
					<option value="7d">In 7 days</option>
					<option value="30d">In 30 days</option>
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
					Revoke link
				</Button>
				<Button onclick={() => (open = false)}>Done</Button>
			{:else if !loading}
				<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
				<Button icon={nuage.link} disabled={working} onclick={create}>
					{working ? 'Creating…' : 'Create public link'}
				</Button>
			{/if}
		</div>
	{/snippet}
</Modal>

<ConfirmModal
	bind:open={revoking}
	title="Revoke this link?"
	description="Anyone still holding it gets a dead page. The {target.type} itself is untouched, and you would have to share it again to undo this."
	confirmLabel="Revoke"
	tone="danger"
	onConfirm={revoke}
/>
