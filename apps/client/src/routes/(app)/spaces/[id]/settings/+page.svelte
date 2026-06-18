<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { backend, type Space, type UserProfile } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	const app = getContext<{ token: string; user: UserProfile | null }>('app');
	const spaceStore = getSpaceStore();

	let space = $state<Space | null>(null);
	let loading = $state(true);
	let name = $state('');
	let description = $state('');
	let saving = $state(false);
	let message = $state('');
	let showDeleteConfirm = $state(false);
	let deleting = $state(false);

	let spaceId = $derived(Number(page.params.id));

	onMount(async () => {
		await loadSpace();
	});

	async function loadSpace() {
		loading = true;
		try {
			space = await backend.getSpace(app.token, spaceId);
			name = space.name;
			description = space.description ?? '';
		} catch {
			space = null;
		}
		loading = false;
	}

	async function saveSettings() {
		saving = true;
		message = '';
		try {
			space = await backend.updateSpace(app.token, spaceId, {
				name: name.trim(),
				description: description.trim()
			});
			message = 'Settings saved';
			if (spaceStore.current?.id === spaceId) {
				spaceStore.set(space);
			}
		} catch (e: any) {
			message = e.message || 'Failed to save';
		}
		saving = false;
		setTimeout(() => { message = ''; }, 3000);
	}

	async function confirmDelete() {
		deleting = true;
		try {
			await backend.deleteSpace(app.token, spaceId);
			if (spaceStore.current?.id === spaceId) {
				spaceStore.clear();
			}
			goto('/spaces');
		} catch {}
		deleting = false;
		showDeleteConfirm = false;
	}

	let isOwner = $derived(space?.role === 'owner');
</script>

<svelte:head>
	<title>Settings — {space?.name ?? 'Space'} — Nuage</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<div class="flex items-center gap-3">
			<a href="/spaces/{spaceId}" class="text-muted-foreground transition-colors hover:text-foreground" aria-label="Back to space">
				<iconify-icon icon="solar:arrow-left-linear" width="20"></iconify-icon>
			</a>
			<div>
				<h1 class="text-lg font-semibold">Space Settings</h1>
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
			<div class="max-w-xl space-y-8">
				<div class="space-y-4">
					<h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">General</h2>
					<div>
						<label for="space-name" class="mb-1.5 block text-sm font-medium">Name</label>
						<input
							id="space-name"
							type="text"
							bind:value={name}
							class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
					</div>
					<div>
						<label for="space-desc" class="mb-1.5 block text-sm font-medium">Description</label>
						<textarea
							id="space-desc"
							bind:value={description}
							rows="3"
							class="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
						></textarea>
					</div>
					<div class="flex items-center gap-3">
						<button
							class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
							onclick={saveSettings}
							disabled={saving}
						>
							{saving ? 'Saving...' : 'Save changes'}
						</button>
						{#if message}
							<span class="text-sm text-muted-foreground">{message}</span>
						{/if}
					</div>
				</div>

				{#if isOwner}
					<div class="space-y-4 border-t border-border pt-8">
						<h2 class="text-sm font-semibold uppercase tracking-wider text-destructive">Danger zone</h2>
						<p class="text-sm text-muted-foreground">
							Deleting a space removes all members. Files and folders that belong to this space will become unscoped.
						</p>
						<button
							class="inline-flex h-9 items-center gap-2 rounded-md border border-destructive/30 bg-destructive/10 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/20 disabled:opacity-50"
							onclick={() => showDeleteConfirm = true}
							disabled={deleting}
						>
							<iconify-icon icon="solar:trash-bin-2-linear" width="16"></iconify-icon>
							Delete space
						</button>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Delete space"
	message="Are you sure you want to delete this space? This action cannot be undone."
	confirmLabel="Delete"
	loading={deleting}
	onconfirm={confirmDelete}
/>
