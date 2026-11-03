<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { Badge, Button, Card, EmptyState, Spinner, toast } from '@facile/muse';
	import { backend, type Space } from '$lib/backend';
	import { icons } from '$lib/icons';
	import { roleTone } from '$lib/roles';
	import { registreSpaceUrl } from '$lib/registre';
	import PageHeader from '$lib/components/PageHeader.svelte';

	const app = getContext<{ token: string }>('app');

	let spaces = $state<Space[]>([]);
	let loading = $state(true);

	onMount(loadSpaces);

	async function loadSpaces() {
		loading = true;
		try {
			const res = await backend.listSpaces(app.token);
			spaces = res.spaces ?? [];
		} catch {
			spaces = [];
			toast.danger('Could not load your spaces.');
		}
		loading = false;
	}
</script>

<svelte:head>
	<title>Spaces — Nuage</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 md:px-8">
	<PageHeader title="Spaces" description="Shared storage for a team. Files in a space belong to the space, not to you.">
		{#snippet actions()}
			<Button href={registreSpaceUrl()} target="_blank" rel="noopener noreferrer" icon={icons.settings}>
				Manage in Registre
			</Button>
		{/snippet}
	</PageHeader>

	{#if loading}
		<div class="flex h-64 items-center justify-center"><Spinner /></div>
	{:else if spaces.length === 0}
		<EmptyState
			icon={icons.usersGroup}
			title="No spaces yet"
			description="Spaces are created in Registre and show up here with their members, in every Facile app."
		>
			<Button href={registreSpaceUrl()} target="_blank" rel="noopener noreferrer" icon={icons.plus}>
				Create a space in Registre
			</Button>
		</EmptyState>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2">
			{#each spaces as space (space.id)}
				<!-- `Card href` renders a real anchor, so these are middle-clickable and show their
				     target in the status bar — and it brings the focus ring and hover step with it. -->
				<Card href="/spaces/{space.id}" class="flex items-center gap-4">
					<span class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface">
						<iconify-icon icon={icons.usersGroup} width="20" height="20" class="block text-fc-fg-muted"
						></iconify-icon>
					</span>
					<div class="flex min-w-0 flex-1 flex-col gap-0.5">
						<p class="truncate text-fc-sm font-medium text-fc-fg">{space.name}</p>
						{#if space.description}
							<p class="truncate text-fc-xs text-fc-fg-muted">{space.description}</p>
						{/if}
					</div>
					<Badge tone={roleTone(space.role)}>{space.role}</Badge>
				</Card>
			{/each}
		</div>
	{/if}
</div>
