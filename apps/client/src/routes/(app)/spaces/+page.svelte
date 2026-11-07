<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { Badge, Button, Card, EmptyState, Spinner, toast } from '@facile/muse';
	import { backend, type Space } from '$lib/backend';
	import { icons } from '$lib/icons';
	import { roleTone } from '$lib/roles';
	import { registreSpaceUrl } from '$lib/registre';
	import { t } from '$lib/i18n/index.svelte';
	import { roleLabel } from '$lib/people';
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
			toast.danger(t('spaces.loadFailed'));
		}
		loading = false;
	}
</script>

<svelte:head>
	<title>{t('spaces.title')}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 md:px-8">
	<PageHeader title={t('spaces.heading')} description={t('spaces.description')}>
		{#snippet actions()}
			<Button href={registreSpaceUrl()} target="_blank" rel="noopener noreferrer" icon={icons.space}>
				{t('spaces.manageInRegistre')}
			</Button>
		{/snippet}
	</PageHeader>

	{#if loading}
		<div class="flex h-64 items-center justify-center"><Spinner /></div>
	{:else if spaces.length === 0}
		<EmptyState
			icon={icons.space}
			title={t('spaces.emptyTitle')}
			description={t('spaces.emptyText')}
		>
			<Button href={registreSpaceUrl()} target="_blank" rel="noopener noreferrer" icon={icons.plus}>
				{t('spaces.create')}
			</Button>
		</EmptyState>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2">
			{#each spaces as space (space.id)}
				<!-- `Card href` renders a real anchor, so these are middle-clickable and show their
				     target in the status bar — and it brings the focus ring and hover step with it. -->
				<Card href="/spaces/{space.id}" class="flex items-center gap-4">
					<span class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface">
						<iconify-icon icon={icons.space} width="20" height="20" class="block text-fc-fg-muted"
						></iconify-icon>
					</span>
					<div class="flex min-w-0 flex-1 flex-col gap-0.5">
						<p class="truncate text-fc-sm font-medium text-fc-fg">{space.name}</p>
						{#if space.description}
							<p class="truncate text-fc-xs text-fc-fg-muted">{space.description}</p>
						{/if}
					</div>
					<Badge tone={roleTone(space.role)}>{roleLabel(space.role)}</Badge>
				</Card>
			{/each}
		</div>
	{/if}
</div>
