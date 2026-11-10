<script lang="ts">
	import { onMount } from 'svelte';
	import { IdeasPage, IdeasUpsell, toast, type ApprovalRequest, type Idea } from '@facile/muse';
	import { ideasStore } from '$lib/ideas.svelte';
	import { toIdea } from '$lib/ideas';
	import { t } from '$lib/i18n/index.svelte';
	import { REGISTRE_SPACES_URL } from '$lib/registre';
	import { isSafeUpgradeUrl } from '$lib/plan-limit';
	import { getSpaceStore } from '$lib/space.svelte';

	const spaceStore = getSpaceStore();

	const cards = $derived((ideasStore.page?.ideas ?? []).map((idea) => toIdea(idea, ideasStore.runs[idea.id], t)));
	const upgradeUrl = $derived(ideasStore.upgradeUrl || ideasStore.page?.upgrade_url);
	const upgradeHref = $derived(isSafeUpgradeUrl(upgradeUrl) ? upgradeUrl : REGISTRE_SPACES_URL);

	const examples = $derived([
		{ kind: 'task', title: t('ideas.example.invoices.title'), why: t('ideas.example.invoices.why') },
		{ kind: 'research', title: t('ideas.example.duplicates.title'), why: t('ideas.example.duplicates.why') },
		{ kind: 'follow_up', title: t('ideas.example.share.title'), why: t('ideas.example.share.why') }
	]);

	onMount(() => void ideasStore.load(spaceStore.id));

	async function dismiss(idea: Idea) {
		if (await ideasStore.dismiss(idea.id)) toast.info(t('ideas.dismissed'));
	}

	function answer(idea: Idea, approval: ApprovalRequest, approve: boolean) {
		void ideasStore.decide(idea.id, approval.id, approve);
	}
</script>

<svelte:head>
	<title>{t('ideas.title')}</title>
</svelte:head>

<div class="mx-auto w-full max-w-5xl px-4 py-6 md:px-8">
	{#if ideasStore.upsell}
		<IdeasUpsell {upgradeHref} {examples} />
	{:else}
		<IdeasPage
			ideas={cards}
			loading={ideasStore.loading}
			generating={ideasStore.generating}
			error={ideasStore.error ? t(ideasStore.error) : null}
			lastGeneratedAt={ideasStore.page?.last_batch_at ?? null}
			busyIds={ideasStore.busyIds}
			description={t('ideas.description')}
			onGenerate={() => void ideasStore.generate()}
			onRetry={() => void ideasStore.load(spaceStore.id)}
			onRun={(idea) => void ideasStore.run(idea.id)}
			onDismiss={dismiss}
			onApprove={(idea, approval) => answer(idea, approval, true)}
			onDecline={(idea, approval) => answer(idea, approval, false)}
		/>
	{/if}
</div>
