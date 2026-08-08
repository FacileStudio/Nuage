<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { page } from '$app/state';
	import { Badge, Button, Card, EmptyState, Spinner, toast } from '@facile/muse';
	import { backend, type Space, type SpaceMember } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';
	import { formatDate } from '$lib/format';
	import { icons, nuage } from '$lib/icons';
	import { roleTone } from '$lib/roles';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import MemberAvatar from '$lib/components/MemberAvatar.svelte';

	const app = getContext<{ token: string }>('app');
	const spaceStore = getSpaceStore();

	let space = $state<Space | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

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
		toast.success(`Now working in “${space.name}”.`);
	}

	const isCurrent = $derived(space !== null && spaceStore.id === space.id);
	const isOwnerOrAdmin = $derived(space?.role === 'owner' || space?.role === 'admin');
	const memberName = (member: SpaceMember) => member.user?.name || member.user?.email || 'Unknown';
</script>

<svelte:head>
	<title>{space?.name ?? 'Space'} — Nuage</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-6 md:px-8">
	<Button variant="ghost" size="sm" href="/spaces" icon={icons.chevronLeft} class="-ml-3 self-start">
		Spaces
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
		<PageHeader title={space.name} description={space.description || undefined}>
			{#snippet actions()}
				<Button icon={nuage.login} disabled={isCurrent} onclick={switchToSpace}>
					{isCurrent ? 'Current space' : 'Switch to this space'}
				</Button>
				{#if isOwnerOrAdmin}
					<Button variant="outline" href="/spaces/{spaceId}/settings" icon={icons.settings}>
						Settings
					</Button>
				{/if}
			{/snippet}
		</PageHeader>

		<div class="flex flex-col gap-10">
			<section class="flex flex-col gap-4">
				<div class="flex flex-wrap items-start justify-between gap-3">
					<div class="flex flex-col gap-1">
						<h2 class="text-fc-lg font-semibold text-fc-fg">Members</h2>
						<p class="text-fc-sm text-fc-fg-muted">
							{members.length}
							{members.length === 1 ? 'person has' : 'people have'} access to everything in this space.
						</p>
					</div>
					{#if isOwnerOrAdmin}
						<Button variant="outline" href="/spaces/{spaceId}/members" icon={nuage.memberAdd}>
							Manage
						</Button>
					{/if}
				</div>

				<!-- One container, rows separated by their own rule. A border per row would draw a
				     box around every person (CHARTE §5). -->
				<Card class="flex flex-col p-0">
					{#each members as member (member.id)}
						<div class="flex items-center gap-3 border-t border-fc-border p-4 first:border-t-0">
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
							<Badge tone={roleTone(member.role)}>{member.role}</Badge>
						</div>
					{/each}
				</Card>
			</section>

			<section class="flex flex-col gap-4">
				<h2 class="text-fc-lg font-semibold text-fc-fg">About</h2>
				<Card class="flex items-center gap-2 text-fc-sm text-fc-fg-muted">
					<iconify-icon icon={icons.calendar} width="16" height="16" class="block shrink-0"></iconify-icon>
					<span>Created {formatDate(space.created_at)}</span>
				</Card>
			</section>
		</div>
	{/if}
</div>
