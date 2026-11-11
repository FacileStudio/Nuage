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
	import { ApiError, backend, type Space, type SpaceMember, type UserProfile } from '$lib/backend';
	import { icons, nuage } from '$lib/icons';
	import { roleTone } from '$lib/roles';
	import { registreSpaceUrl } from '$lib/registre';
	import { t } from '$lib/i18n/index.svelte';
	import { displayName, roleLabel } from '$lib/people';
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
	 * The people the caller already shares a space with: GET /users shows nobody else. Anyone
	 * outside that circle is added by typing their exact email address.
	 */
	async function loadDirectory() {
		if (directoryLoaded) return;
		try {
			const res = await backend.listUsers(app.token);
			directory = res.users ?? [];
			directoryLoaded = true;
		} catch {
			addError = t('members.directoryFailed');
		}
	}

	async function addMember() {
		if (!picked && !byEmail) return;
		const uid = picked ? Number(picked.id) : NaN;
		if (picked && !Number.isFinite(uid)) {
			addError = t('members.badId');
			return;
		}
		const name = picked ? displayName(picked) : typedEmail;

		adding = true;
		addError = '';
		try {
			await backend.addSpaceMember(
				app.token,
				spaceId,
				picked ? { user_id: uid, role: addRole } : { email: typedEmail, role: addRole }
			);
			await refreshMembers();
			addOpen = false;
			toast.success(t('members.added', { name }));
		} catch (e) {
			addError =
				e instanceof ApiError && e.status === 404
					? t('members.noAccount')
					: e instanceof Error && e.message
						? e.message
						: t('members.addFailed');
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

	const typedEmail = $derived(pickQuery.trim().toLowerCase());
	const byEmail = $derived(
		!picked &&
			/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(typedEmail) &&
			!directory.some((u) => u.email?.toLowerCase() === typedEmail)
	);

	async function updateRole(member: SpaceMember, role: string) {
		try {
			await backend.updateSpaceMember(app.token, spaceId, member.id, { role });
			await refreshMembers();
			toast.success(t('members.roleChanged', { name: memberName(member), role: roleLabel(role) }));
		} catch {
			/* Re-read rather than trust the select: the option the user picked did not stick. */
			await refreshMembers();
			toast.danger(t('members.roleFailed'));
		}
	}

	async function removeMember() {
		const target = removing;
		if (!target) return;
		try {
			await backend.removeSpaceMember(app.token, spaceId, target.id);
			members = members.filter((m) => m.id !== target.id);
			removing = null;
			toast.success(t('members.removed', { name: memberName(target) }));
		} catch (e) {
			toast.danger(e instanceof Error && e.message ? e.message : t('members.removeFailed'));
			throw e;
		}
	}

	const linked = $derived(Boolean(space?.registre_space_id));
	const isOwnerOrAdmin = $derived(!linked && (space?.role === 'owner' || space?.role === 'admin'));
	const isOwner = $derived(space?.role === 'owner');
	const ownerCount = $derived(members.filter((m) => m.role === 'owner').length);
	const memberName = (member: SpaceMember) => displayName(member.user ?? {});

	function canRemove(member: SpaceMember) {
		if (!isOwnerOrAdmin) return false;
		if (space?.role === 'admin' && (member.role === 'owner' || member.role === 'admin')) return false;
		if (member.role === 'owner' && ownerCount <= 1) return false;
		return true;
	}
</script>

<svelte:head>
	<title>{t('members.title', { name: space?.name ?? t('space.fallbackName') })}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-6 md:px-8">
	<Button
		variant="ghost"
		size="sm"
		href="/spaces/{spaceId}"
		icon={icons.chevronLeft}
		class="-ml-3 self-start"
	>
		{space?.name ?? t('space.fallbackName')}
	</Button>

	{#if loading}
		<div class="flex h-64 items-center justify-center"><Spinner /></div>
	{:else if !space}
		<EmptyState
			icon={icons.space}
			title={t('space.notFound')}
			description={t('space.notFoundText')}
		>
			<Button href="/spaces" icon={icons.chevronLeft}>{t('space.backToSpaces')}</Button>
		</EmptyState>
	{:else}
		<PageHeader
			title={t('space.membersHeading')}
			description={t('members.description')}
		>
			{#snippet actions()}
				{#if linked}
					<Button
						variant="secondary"
						href={registreSpaceUrl(space?.registre_space_id)}
						target="_blank"
						rel="noopener noreferrer"
						icon={icons.space}
					>
						{t('spaces.manageInRegistre')}
					</Button>
				{:else if isOwnerOrAdmin}
					<Button icon={nuage.memberAdd} onclick={openAdd}>{t('members.add')}</Button>
				{/if}
			{/snippet}
		</PageHeader>

		<Card class="flex flex-col p-0">
			{#each members as member (member.id)}
				<div
					class="flex flex-wrap items-center gap-3 p-4"
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

					<div class="flex items-center gap-1">
						<!-- The owner's role is not a control: `updateMember` refuses to change any
						     owner's role at all, so the select would only ever produce an error.
						     Removal is a separate question — `removeMember` allows it as long as
						     another owner remains. -->
						{#if member.role === 'owner' || !isOwnerOrAdmin}
							<Badge tone={roleTone(member.role)}>{roleLabel(member.role)}</Badge>
						{:else}
							<Select
								value={member.role}
								aria-label={t('members.roleFor', { name: memberName(member) })}
								class="h-9 w-32 text-fc-sm"
								onchange={(e) => updateRole(member, (e.currentTarget as HTMLSelectElement).value)}
							>
								<option value="member">{t('role.member')}</option>
								<option value="admin">{t('role.admin')}</option>
								{#if isOwner}<option value="owner">{t('role.owner')}</option>{/if}
							</Select>
						{/if}
						{#if canRemove(member)}
							<Button
								variant="ghost-danger"
								size="sm"
								icon={icons.remove}
								aria-label={t('members.removeLabel', { name: memberName(member) })}
								onclick={() => (removing = member)}
							>
								{t('common.remove')}
							</Button>
						{/if}
					</div>
				</div>
			{/each}
		</Card>
	{/if}
</div>

<Drawer bind:open={addOpen} title={t('members.drawerTitle')} showClose>
	<div class="flex flex-col gap-4">
		{#if addError}
			<Alert tone="danger">{addError}</Alert>
		{/if}

		<Field label={t('members.who')} helper={t('members.whoHelper')}>
			<Input bind:value={pickQuery} placeholder={t('members.search')} autocomplete="off" />
		</Field>

		{#if !directoryLoaded}
			<div class="flex h-24 items-center justify-center"><Spinner /></div>
		{:else if candidates.length === 0}
			<p class="py-6 text-center text-fc-sm text-fc-fg-muted">
				{byEmail
					? t('members.byEmail', { email: typedEmail })
					: pickQuery.trim()
						? t('members.nobody')
						: t('members.everyone')}
			</p>
		{:else}
			<!-- A radiogroup, not a list of independently tabbable buttons: picking one person
			     out of many is a single choice, and it should be one tab stop. -->
			<div
				class="flex max-h-64 flex-col overflow-y-auto rounded-fc-md bg-fc-surface"
				role="radiogroup"
				aria-label={t('members.choose')}
			>
				{#each candidates as candidate (candidate.id)}
					{@const selected = picked?.id === candidate.id}
					<button
						type="button"
						role="radio"
						aria-checked={selected}
						tabindex={selected || (!picked && candidate.id === candidates[0].id) ? 0 : -1}
						onclick={() => (picked = candidate)}
						class="flex items-center gap-3 p-3 text-left transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-fc-ring {selected
							? 'bg-fc-accent text-fc-accent-fg'
							: 'hover:bg-fc-component'}"
					>
						<MemberAvatar
							name={displayName(candidate)}
							src={candidate.avatar_url}
							color={selected ? undefined : candidate.color}
						/>
						<span class="flex min-w-0 flex-1 flex-col">
							<span class="truncate text-fc-sm font-medium">{displayName(candidate)}</span>
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

		<Field label={t('members.role')}>
			<Select bind:value={addRole}>
				<option value="member">{t('members.role.member')}</option>
				<option value="admin">{t('members.role.admin')}</option>
			</Select>
		</Field>
	</div>

	{#snippet footer()}
		<Button
			size="lg"
			class="w-full"
			icon={nuage.memberAdd}
			disabled={adding || (!picked && !byEmail)}
			onclick={addMember}
		>
			{adding
				? t('members.adding')
				: picked
					? t('members.addPerson', { name: displayName(picked) })
					: byEmail
						? t('members.addPerson', { name: typedEmail })
						: t('members.pickFirst')}
		</Button>
	{/snippet}
</Drawer>

<ConfirmModal
	open={removing !== null}
	title={t('members.removeTitle')}
	description={removing ? t('members.removeText', { name: memberName(removing) }) : ''}
	confirmLabel={t('common.remove')}
	tone="danger"
	onConfirm={removeMember}
	onCancel={() => (removing = null)}
/>
