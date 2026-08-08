<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { page } from '$app/state';
	import {
		Alert,
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
	import { backend, type Space, type SpaceMember, type UserProfile } from '$lib/backend';
	import { icons, nuage } from '$lib/icons';
	import { roleTone } from '$lib/roles';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import MemberAvatar from '$lib/components/MemberAvatar.svelte';

	const app = getContext<{ token: string }>('app');

	let space = $state<Space | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

	let addOpen = $state(false);
	let addRole = $state('member');
	let addError = $state('');
	let adding = $state(false);
	let removing = $state<SpaceMember | null>(null);

	let directory = $state<UserProfile[]>([]);
	let directoryLoaded = $state(false);
	let pickQuery = $state('');
	let picked = $state<UserProfile | null>(null);

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
		addRole = 'member';
		addError = '';
		pickQuery = '';
		picked = null;
		addOpen = true;
		void loadDirectory();
	}

	/*
	 * The instance directory, fetched once per visit. This replaces a text field that asked a
	 * human to type a numeric user id — the API only accepts `user_id`, but `GET /users` has
	 * always been there to turn one into the other, so the id never needed to be the user's
	 * problem. No new exposure either: that endpoint is readable by any authenticated caller.
	 */
	async function loadDirectory() {
		if (directoryLoaded) return;
		try {
			const res = await backend.listUsers(app.token);
			directory = res.users ?? [];
			directoryLoaded = true;
		} catch {
			addError = 'Could not load the user directory.';
		}
	}

	async function addMember() {
		if (!picked) return;
		const uid = Number(picked.id);
		if (!Number.isFinite(uid)) {
			addError = 'That account has an id this API will not accept.';
			return;
		}

		adding = true;
		addError = '';
		try {
			await backend.addSpaceMember(app.token, spaceId, { user_id: uid, role: addRole });
			await refreshMembers();
			addOpen = false;
			toast.success(`Added ${picked.name || picked.email}.`);
		} catch (e) {
			addError = e instanceof Error && e.message ? e.message : 'Could not add that member.';
		}
		adding = false;
	}

	/* Already-members are filtered out rather than shown and rejected by the server. */
	const candidates = $derived.by(() => {
		const taken = new Set(members.map((m) => String(m.user_id)));
		const q = pickQuery.trim().toLowerCase();
		return directory
			.filter((u) => !taken.has(String(u.id)))
			.filter((u) => !q || u.name?.toLowerCase().includes(q) || u.email?.toLowerCase().includes(q))
			.slice(0, 50);
	});

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
	<div class="flex flex-col gap-4">
		{#if addError}
			<Alert tone="danger">{addError}</Alert>
		{/if}

		<Field label="Who" helper="Anyone with an account on this Nuage.">
			<Input bind:value={pickQuery} placeholder="Search by name or email" autocomplete="off" />
		</Field>

		{#if !directoryLoaded}
			<div class="flex h-24 items-center justify-center"><Spinner /></div>
		{:else if candidates.length === 0}
			<p class="py-6 text-center text-fc-sm text-fc-fg-muted">
				{pickQuery.trim() ? 'Nobody matches that.' : 'Everyone with an account is already a member.'}
			</p>
		{:else}
			<!-- A radiogroup, not a list of independently tabbable buttons: picking one person
			     out of many is a single choice, and it should be one tab stop. -->
			<div
				class="flex max-h-64 flex-col overflow-y-auto rounded-fc-md bg-fc-surface"
				role="radiogroup"
				aria-label="Choose a person to add"
			>
				{#each candidates as candidate (candidate.id)}
					{@const selected = picked?.id === candidate.id}
					<button
						type="button"
						role="radio"
						aria-checked={selected}
						tabindex={selected || (!picked && candidate.id === candidates[0].id) ? 0 : -1}
						onclick={() => (picked = candidate)}
						class="flex items-center gap-3 border-t border-fc-border p-3 text-left transition-colors first:border-t-0 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-fc-ring {selected
							? 'bg-fc-accent text-fc-accent-fg'
							: 'hover:bg-fc-component'}"
					>
						<MemberAvatar
							name={candidate.name || candidate.email}
							src={candidate.avatar_url}
							color={selected ? undefined : candidate.color}
						/>
						<span class="flex min-w-0 flex-1 flex-col">
							<span class="truncate text-fc-sm font-medium">{candidate.name || candidate.email}</span>
							{#if candidate.name && candidate.email}
								<span class="truncate text-fc-xs {selected ? 'opacity-70' : 'text-fc-fg-muted'}">
									{candidate.email}
								</span>
							{/if}
						</span>
						{#if selected}
							<iconify-icon icon={nuage.tick} width="16" height="16" class="block shrink-0"
							></iconify-icon>
						{/if}
					</button>
				{/each}
			</div>
		{/if}

		<Field label="Role">
			<Select bind:value={addRole}>
				<option value="member">Member — read and write files</option>
				<option value="admin">Admin — also manages members</option>
			</Select>
		</Field>
	</div>

	{#snippet footer()}
		<Button
			size="lg"
			class="w-full"
			icon={nuage.memberAdd}
			disabled={adding || !picked}
			onclick={addMember}
		>
			{adding ? 'Adding…' : picked ? `Add ${picked.name || picked.email}` : 'Pick someone first'}
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
