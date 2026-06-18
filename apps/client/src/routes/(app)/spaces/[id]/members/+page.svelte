<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import { page } from '$app/state';
	import { backend, type Space, type SpaceMember, type UserProfile } from '$lib/backend';

	const app = getContext<{ token: string; user: UserProfile | null }>('app');

	let space = $state<Space | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);
	let addUserId = $state('');
	let addRole = $state('member');
	let addError = $state('');
	let adding = $state(false);

	let spaceId = $derived(Number(page.params.id));

	onMount(async () => {
		await loadData();
	});

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

	async function addMember() {
		const uid = parseInt(addUserId, 10);
		if (isNaN(uid)) {
			addError = 'Enter a valid user ID';
			return;
		}

		adding = true;
		addError = '';
		try {
			await backend.addSpaceMember(app.token, spaceId, { user_id: uid, role: addRole });
			addUserId = '';
			addRole = 'member';
			const res = await backend.listSpaceMembers(app.token, spaceId);
			members = res.members ?? [];
		} catch (e: any) {
			addError = e.message || 'Failed to add member';
		}
		adding = false;
	}

	async function updateRole(memberId: number, role: string) {
		try {
			await backend.updateSpaceMember(app.token, spaceId, memberId, { role });
			const res = await backend.listSpaceMembers(app.token, spaceId);
			members = res.members ?? [];
		} catch {}
	}

	async function removeMember(memberId: number) {
		try {
			await backend.removeSpaceMember(app.token, spaceId, memberId);
			members = members.filter(m => m.id !== memberId);
		} catch {}
	}

	function roleBadgeClass(role: string): string {
		switch (role) {
			case 'owner': return 'bg-amber-500/10 text-amber-600';
			case 'admin': return 'bg-blue-500/10 text-blue-600';
			default: return 'bg-muted text-muted-foreground';
		}
	}

	let isOwnerOrAdmin = $derived(space?.role === 'owner' || space?.role === 'admin');
	let isOwner = $derived(space?.role === 'owner');
</script>

<svelte:head>
	<title>Members — {space?.name ?? 'Space'} — Nuage</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<div class="flex items-center gap-3">
			<a href="/spaces/{spaceId}" class="text-muted-foreground transition-colors hover:text-foreground" aria-label="Back to space">
				<iconify-icon icon="solar:arrow-left-linear" width="20"></iconify-icon>
			</a>
			<div>
				<h1 class="text-lg font-semibold">Members</h1>
				{#if space}
					<p class="mt-0.5 text-sm text-muted-foreground">{space.name}</p>
				{/if}
			</div>
		</div>
	</div>

	<div class="flex-1 overflow-auto px-4 py-6 md:px-8">
		{#if loading}
			<div class="flex items-center justify-center py-20">
				<div class="h-6 w-6 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
			</div>
		{:else if !space}
			<div class="flex flex-col items-center justify-center py-20 text-center">
				<p class="text-sm text-muted-foreground">Space not found or access denied.</p>
			</div>
		{:else}
			<div class="max-w-2xl space-y-8">
				{#if isOwnerOrAdmin}
					<div class="space-y-3">
						<h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Add member</h2>
						<div class="flex items-end gap-2">
							<div class="flex-1">
								<label for="user-id" class="mb-1.5 block text-sm font-medium">User ID</label>
								<input
									id="user-id"
									type="text"
									bind:value={addUserId}
									placeholder="Enter user ID"
									class="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
								/>
							</div>
							<div class="w-32">
								<label for="add-role" class="mb-1.5 block text-sm font-medium">Role</label>
								<select
									id="add-role"
									bind:value={addRole}
									class="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
								>
									<option value="member">Member</option>
									<option value="admin">Admin</option>
								</select>
							</div>
							<button
								class="inline-flex h-9 items-center gap-2 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
								onclick={addMember}
								disabled={adding}
							>
								{adding ? 'Adding...' : 'Add'}
							</button>
						</div>
						{#if addError}
							<p class="text-sm text-destructive">{addError}</p>
						{/if}
					</div>
				{/if}

				<div class="space-y-3">
					<h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
						{members.length} member{members.length !== 1 ? 's' : ''}
					</h2>

					{#each members as member}
						<div class="flex items-center justify-between rounded-lg border border-border p-3">
							<div class="flex items-center gap-3">
								{#if member.user?.avatar_url}
									<img src={member.user.avatar_url} alt="" class="h-9 w-9 rounded-full border border-border object-cover" />
								{:else}
									<div
										class="flex h-9 w-9 items-center justify-center rounded-full border border-border text-xs font-semibold text-white"
										style="background-color: {member.user?.color || '#6b7280'}"
									>
										{(member.user?.name || member.user?.email || '?').slice(0, 2).toUpperCase()}
									</div>
								{/if}
								<div>
									<p class="text-sm font-medium">{member.user?.name || member.user?.email}</p>
									{#if member.user?.name}
										<p class="text-xs text-muted-foreground">{member.user.email}</p>
									{/if}
								</div>
							</div>
							<div class="flex items-center gap-2">
								{#if member.role === 'owner'}
									<span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {roleBadgeClass('owner')}">
										owner
									</span>
								{:else if isOwnerOrAdmin}
									<select
										value={member.role}
										onchange={(e) => updateRole(member.id, (e.target as HTMLSelectElement).value)}
										class="h-8 rounded-md border border-border bg-background px-2 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
									>
										<option value="member">Member</option>
										<option value="admin">Admin</option>
										{#if isOwner}
											<option value="owner">Owner</option>
										{/if}
									</select>
									<button
										class="inline-flex h-8 items-center rounded-md px-2 text-xs text-destructive transition-colors hover:bg-destructive/10"
										onclick={() => removeMember(member.id)}
										aria-label="Remove member"
									>
										<iconify-icon icon="solar:trash-bin-2-linear" width="14"></iconify-icon>
									</button>
								{:else}
									<span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {roleBadgeClass(member.role)}">
										{member.role}
									</span>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>
</div>
