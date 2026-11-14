<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Badge, Button, Card, ConfirmModal, EmptyState, Spinner, toast } from '@facile/muse';
	import { backend, type Space, type SpaceMember } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';
	import { formatDate } from '$lib/format';
	import { icons, nuage } from '$lib/icons';
	import { roleTone } from '$lib/roles';
	import { registreSpaceUrl } from '$lib/registre';
	import { t } from '$lib/i18n/index.svelte';
	import { displayName, roleLabel } from '$lib/people';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import MemberAvatar from '$lib/components/MemberAvatar.svelte';

	const app = getContext<{ token: string }>('app');
	const spaceStore = getSpaceStore();

	let space = $state<Space | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);
	let leaveOpen = $state(false);

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

	function switchToSpace() {
		if (!space) return;
		spaceStore.set(space);
		toast.success(t('space.switched', { name: space.name }));
	}

	/*
	 * The one refusal a real user hits is the 409 a sole owner gets, and its message names the
	 * two ways out — so it is shown verbatim and the error is rethrown, which is what keeps
	 * ConfirmModal open instead of dismissing the dialog on a leave that never happened.
	 */
	async function confirmLeave() {
		const target = space;
		if (!target) return;
		try {
			await backend.leaveSpace(app.token, spaceId);
		} catch (e) {
			toast.danger(e instanceof Error && e.message ? e.message : t('space.leaveFailed'));
			throw e;
		}
		if (spaceStore.id === target.id) spaceStore.clear();
		toast.success(t('space.left', { name: target.name }));
		goto('/spaces');
	}

	const isCurrent = $derived(space !== null && spaceStore.id === space.id);
	const isOwnerOrAdmin = $derived(space?.role === 'owner' || space?.role === 'admin');
	const linked = $derived(Boolean(space?.registre_space_id));
	const memberName = (member: SpaceMember) => displayName(member.user ?? {});
</script>

<svelte:head>
	<title>{t('space.pageTitle', { name: space?.name ?? t('space.fallbackName') })}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-6 md:px-8">
	<Button variant="ghost" size="sm" href="/spaces" icon={icons.chevronLeft} class="-ml-3 self-start">
		{t('spaces.heading')}
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
		<PageHeader title={space.name} description={space.description || undefined}>
			{#snippet actions()}
				<Button icon={nuage.login} disabled={isCurrent} onclick={switchToSpace}>
					{isCurrent ? t('space.current') : t('space.switch')}
				</Button>
				{#if isOwnerOrAdmin}
					<Button variant="secondary" href="/spaces/{spaceId}/settings" icon={icons.settings}>
						{t('space.settings')}
					</Button>
				{/if}
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
				{:else}
					<Button variant="ghost-danger" icon={icons.logout} onclick={() => (leaveOpen = true)}>
						{t('space.leave')}
					</Button>
				{/if}
			{/snippet}
		</PageHeader>

		<div class="flex flex-col gap-10">
			<section class="flex flex-col gap-4">
				<div class="flex flex-wrap items-start justify-between gap-3">
					<div class="flex flex-col gap-1">
						<h2 class="text-fc-lg font-bold text-fc-fg">{t('space.membersHeading')}</h2>
						<p class="text-fc-sm text-fc-fg-muted">{t('space.membersCount', { count: members.length })}</p>
					</div>
					{#if isOwnerOrAdmin && !linked}
						<Button variant="secondary" href="/spaces/{spaceId}/members" icon={nuage.memberAdd}>
							{t('space.manage')}
						</Button>
					{/if}
				</div>

				<!-- One container, rows separated by spacing. -->
				<Card class="flex flex-col p-0">
					{#each members as member (member.id)}
						<div class="flex items-center gap-3 p-4">
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
							<Badge tone={roleTone(member.role)}>{roleLabel(member.role)}</Badge>
						</div>
					{/each}
				</Card>
			</section>

			<section class="flex flex-col gap-4">
				<h2 class="text-fc-lg font-bold text-fc-fg">{t('space.about')}</h2>
				<Card class="flex items-center gap-2 text-fc-sm text-fc-fg-muted">
					<iconify-icon icon={icons.calendar} width="16" height="16" class="block shrink-0"></iconify-icon>
					<span>{t('space.created', { date: formatDate(space.created_at) })}</span>
				</Card>
			</section>
		</div>
	{/if}
</div>

<ConfirmModal
	bind:open={leaveOpen}
	title={t('space.leaveTitle', { name: space?.name ?? t('space.leaveFallback') })}
	description={t('space.leaveText')}
	confirmLabel={t('space.leave')}
	tone="danger"
	onConfirm={confirmLeave}
/>
