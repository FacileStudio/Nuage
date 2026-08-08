<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		Button,
		ConfirmModal,
		EmptyState,
		Field,
		Input,
		SettingsRow,
		SettingsSection,
		Spinner,
		Textarea,
		toast
	} from '@facile/muse';
	import { backend, type Space } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';
	import { icons } from '$lib/icons';
	import PageHeader from '$lib/components/PageHeader.svelte';

	const app = getContext<{ token: string }>('app');
	const spaceStore = getSpaceStore();

	let space = $state<Space | null>(null);
	let loading = $state(true);
	let name = $state('');
	let description = $state('');
	let saving = $state(false);
	let deleteOpen = $state(false);

	let spaceId = $derived(Number(page.params.id));

	onMount(loadSpace);

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
		try {
			space = await backend.updateSpace(app.token, spaceId, {
				name: name.trim(),
				description: description.trim()
			});
			if (spaceStore.current?.id === spaceId) spaceStore.set(space);
			toast.success('Space updated.');
		} catch (e) {
			toast.danger(e instanceof Error && e.message ? e.message : 'Could not save that.');
		}
		saving = false;
	}

	async function confirmDelete() {
		await backend.deleteSpace(app.token, spaceId);
		if (spaceStore.current?.id === spaceId) spaceStore.clear();
		toast.success('Space deleted.');
		goto('/spaces');
	}

	const isOwner = $derived(space?.role === 'owner');
</script>

<svelte:head>
	<title>Settings — {space?.name ?? 'Space'} — Nuage</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-10 px-4 py-6 md:px-8">
	<div class="flex flex-col gap-4">
		<Button
			variant="ghost"
			size="sm"
			href="/spaces/{spaceId}"
			icon={icons.chevronLeft}
			class="-ml-3 self-start"
		>
			{space?.name ?? 'Space'}
		</Button>
		<PageHeader title="Space settings" />
	</div>

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
		<SettingsSection title="General" description="How this space is labelled everywhere it appears.">
			<SettingsRow label="Name" stacked>
				<Field>
					<Input bind:value={name} placeholder="Design team" />
				</Field>
			</SettingsRow>
			<SettingsRow label="Description" stacked>
				<Field>
					<Textarea bind:value={description} rows={3} placeholder="What belongs in here." />
				</Field>
			</SettingsRow>
			<SettingsRow>
				<Button icon={icons.check} disabled={saving} onclick={saveSettings}>
					{saving ? 'Saving…' : 'Save changes'}
				</Button>
			</SettingsRow>
		</SettingsSection>

		{#if isOwner}
			<!-- Danger zone sits last and never gets a heading of its own above the fold: a
			     destructive action should cost a scroll (CHARTE §14). -->
			<SettingsSection
				title="Danger zone"
				description="Deleting the space removes every member. Its files and folders survive, but they stop being scoped to a space and only the uploader keeps them."
			>
				<SettingsRow
					label="Delete this space"
					description="There is no undo, and the name is not reserved afterwards."
				>
					<Button variant="danger" icon={icons.remove} onclick={() => (deleteOpen = true)}>
						Delete space
					</Button>
				</SettingsRow>
			</SettingsSection>
		{/if}
	{/if}
</div>

<ConfirmModal
	bind:open={deleteOpen}
	title="Delete “{space?.name ?? 'this space'}”?"
	description="Every member loses access at once. The files inside stop being shared and fall back to their uploader. This cannot be undone."
	confirmLabel="Delete space"
	tone="danger"
	onConfirm={confirmDelete}
/>
