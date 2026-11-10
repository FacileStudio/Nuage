<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { EmptyState, SecretField, Spinner, toast } from '@facile/muse';
	import { backend, type PendingAction } from '$lib/backend';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import PendingCard from '$lib/components/PendingCard.svelte';
	import { icons } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	let actions = $state<PendingAction[]>([]);
	let links = $state<{ id: string; name: string; url: string }[]>([]);
	let loading = $state(true);
	let busy = $state('');

	const focused = $derived(page.url.searchParams.get('action') ?? '');
	const ordered = $derived([
		...actions.filter((action) => action.id === focused),
		...actions.filter((action) => action.id !== focused)
	]);

	onMount(load);

	/** Loads the open actions. */
	async function load() {
		loading = true;
		try {
			actions = (await backend.listPending()).actions;
		} catch {
			toast.danger(t('pending.loadFailed'));
		}
		loading = false;
	}

	/** Runs one action once, then drops it from the list; a new link stays shown. */
	async function confirm(action: PendingAction) {
		busy = action.id;
		try {
			const result = await backend.confirmPending(action.id);
			if (result.url) links = [...links, { id: action.id, name: action.item?.name ?? '', url: result.url }];
			toast.success(t(action.kind === 'share' ? 'pending.shared' : 'pending.trashed'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('pending.failed'));
			await load();
		}
		busy = '';
	}

	/** Closes one action without running it. */
	async function dismiss(action: PendingAction) {
		busy = action.id;
		try {
			await backend.dismissPending(action.id);
			toast.success(t('pending.dismissed'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('pending.failed'));
		}
		busy = '';
	}
</script>

<svelte:head>
	<title>{t('pending.title')}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-2xl flex-col gap-6 px-4 py-6 md:px-8">
	<PageHeader title={t('pending.heading')} description={t('pending.intro')} />

	{#each links as link (link.id)}
		<SecretField label={`${t('pending.newLink')} · ${link.name}`} value={link.url} sensitive={false} />
	{/each}

	{#if loading}
		<div class="flex h-48 items-center justify-center"><Spinner /></div>
	{:else if actions.length === 0}
		<EmptyState icon={icons.check} title={t('pending.none')} description={t('pending.noneBody')} />
	{:else}
		<section class="flex flex-col gap-4">
			{#each ordered as action (action.id)}
				<PendingCard {action} busy={busy === action.id} onconfirm={confirm} ondismiss={dismiss} />
			{/each}
		</section>
	{/if}
</div>
