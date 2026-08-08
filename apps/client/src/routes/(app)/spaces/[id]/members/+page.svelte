<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { page } from '$app/state';
	import {
		Badge,
		Button,
		Card,
		ConfirmModal,
		Drawer,
		EmptyState,
		Field,
		Input,
		Select,
		Spinner,
		toast
	} from '@facile/muse';
	import { backend, type Space, type SpaceMember } from '$lib/backend';
	import { icons, nuage } from '$lib/icons';
	import { roleTone } from '$lib/roles';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import MemberAvatar from '$lib/components/MemberAvatar.svelte';

	const app = getContext<{ token: string }>('app');

	let space = $state<Space | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

	let addOpen = $state(false);
	let addUserId = $state('');
	let addRole = $state('member');
	let addError = $state('');
	let adding = $state(false);
	let removing = $state<SpaceMember | null>(null);

	let spaceId = $derived(Number(page.params.id));

	onMount(loadData);

	async function loadData() {
		loading = true;
		try {
			const [spaceRes, membersRes] = await Promise.all([
				backend.getSpace(app.token, spaceId),
				backend.listSpaceMembers(app.token, spaceId)
			]);
			space = spaceRes;
			members = membersRes.members ?? [];
		} catch {
			space = null;
			members = [];
		}
		loading = false;
	}

	async function refreshMembers() {
		const res = await backend.listSpaceMembers(app.token, spaceId);
		members = res.members ?? [];
	}

	function openAdd() {
		addUserId = '';
		addRole = 'member';
		addError = '';
		addOpen = true;
	}

	async function addMember() {
		const uid = Number.parseInt(addUserId, 10);
		if (!Number.isInteger(uid) || uid <= 0) {
			addError = 'That is not a user ID.';
			return;
		}

		adding = true;
		addError = '';
		try {
			await backend.addSpaceMember(app.token, spaceId, { user_id: uid, role: addRole });
			await refreshMembers();
			addOpen = false;
			toast.success('Member added.');
		} catch (e) {
			addError = e instanceof Error && e.message ? e.message : 'Could not add that member.';
		}
		adding = false;
	}

	async function updateRole(member: SpaceMember, role: string) {
		try {
			await backend.updateSpaceMember(app.token, spaceId, member.id, { role });
			await refreshMembers();
			toast.success(`${memberName(member)} is now ${role}.`);
		} catch {
			/* Re-read rather than trust the select: the option the user picked did not stick. */
			await refreshMembers();
			toast.danger('Could not change that role.');
		}
	}

	async function removeMember() {
		const target = removing;
		if (!target) return;
		try {
			await backend.removeSpaceMember(app.token, spaceId, target.id);
			members = members.filter((m) => m.id !== target.id);
			removing = null;
			toast.success(`Removed ${memberName(target)}.`);
		} catch {
			toast.danger('Could not remove that member.');
		}
	}

	const isOwnerOrAdmin = $derived(space?.role === 'owner' || space?.role === 'admin');
	const isOwner = $derived(space?.role === 'owner');
	const memberName = (member: SpaceMember) => member.user?.name || member.user?.email || 'Unknown';
</script>

<svelte:head>
	<title>Members — {space?.name ?? 'Space'} — Nuage</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-6 md:px-8">
	<Button
		variant="ghost"
		size="sm"
		href="/spaces/{spaceId}"
		icon={icons.chevronLeft}
		class="-ml-3 self-start"
	>
		{space?.name ?? 'Space'}
	</Button>

	{#if loading}
		<div class="flex h-64 items-center justify-center"><Spinner /></div>
	{:else if !space}
		<EmptyState
			icon={icons.warning}
			title="Space not found"
			description="It may have been deleted, or you may no longer be a member."
		>
			<Button href="/spaces" icon={icons.usersGroup}>Back to spaces</Button>
		</EmptyState>
	{:else}
		<PageHeader
			title="Members"
			description="Everyone here can read and write every file in the space."
		>
			{#snippet actions()}
				{#if isOwnerOrAdmin}
					<Button icon={nuage.memberAdd} onclick={openAdd}>Add member</Button>
				{/if}
			{/snippet}
		</PageHeader>

		<Card class="flex flex-col p-0">
			{#each members as member (member.id)}
				<div
					class="flex flex-wrap items-center gap-3 border-t border-fc-border p-4 first:border-t-0"
				>
					<MemberAvatar
						name={memberName(member)}
						src={member.user?.avatar_url}
						color={member.user?.color}
					/>
					<div class="flex min-w-0 flex-1 flex-col">
						<p class="truncate text-fc-sm font-medium text-fc-fg">{memberName(member)}</p>
						{#if member.user?.name && member.user.email}
							<p class="truncate text-fc-xs text-fc-fg-muted">{member.user.email}</p>
						{/if}
					</div>

					<!-- The owner's role is not a control. Demoting the last owner orphans the space,
					     and the API refuses it anyway, so the UI should not offer the move. -->
					{#if member.role === 'owner' || !isOwnerOrAdmin}
						<Badge tone={roleTone(member.role)}>{member.role}</Badge>
					{:else}
						<div class="flex items-center gap-1">
							<Select
								value={member.role}
								aria-label="Role for {memberName(member)}"
								class="h-9 w-32 text-fc-sm"
								onchange={(e) => updateRole(member, (e.currentTarget as HTMLSelectElement).value)}
							>
								<option value="member">Member</option>
								<option value="admin">Admin</option>
								{#if isOwner}<option value="owner">Owner</option>{/if}
							</Select>
							<Button
								variant="ghost-danger"
								size="sm"
								icon={icons.remove}
								aria-label="Remove {memberName(member)}"
								onclick={() => (removing = member)}
							>
								Remove
							</Button>
						</div>
					{/if}
				</div>
			{/each}
		</Card>
	{/if}
</div>

<Drawer bind:open={addOpen} title="Add a member" showClose>
	<Field
		label="User ID"
		error={addError || undefined}
		helper="The numeric id from the user's record. Inviting by email is not wired up yet — the API only accepts an id."
	>
		<Input bind:value={addUserId} inputmode="numeric" placeholder="42" />
	</Field>
	<Field label="Role" class="mt-4">
		<Select bind:value={addRole}>
			<option value="member">Member — read and write files</option>
			<option value="admin">Admin — also manages members</option>
		</Select>
	</Field>

	{#snippet footer()}
		<Button size="lg" class="w-full" icon={nuage.memberAdd} disabled={adding} onclick={addMember}>
			{adding ? 'Adding…' : 'Add member'}
		</Button>
	{/snippet}
</Drawer>

<ConfirmModal
	open={removing !== null}
	title="Remove this member?"
	description={removing
		? `${memberName(removing)} loses access to every file in this space immediately. Files they uploaded stay in the space.`
		: ''}
	confirmLabel="Remove"
	tone="danger"
	onConfirm={removeMember}
	onCancel={() => (removing = null)}
/>
