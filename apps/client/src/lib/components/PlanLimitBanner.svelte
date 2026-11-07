<script lang="ts">
	import { Alert, Button } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
	import { isSafeUpgradeUrl, planLimits, type PlanLimit } from '$lib/plan-limit';

	let refusal = $state<PlanLimit | null>(null);
	const upgrade = $derived(isSafeUpgradeUrl(refusal?.upgrade_url) ? refusal?.upgrade_url : undefined);

	$effect(() => {
		const show = (event: Event) => (refusal = (event as CustomEvent<PlanLimit>).detail);
		planLimits.addEventListener('plan-limit', show);
		return () => planLimits.removeEventListener('plan-limit', show);
	});
</script>

{#if refusal}
	<Alert tone="warning" title={t('plan.full')} class="mb-3 flex flex-col gap-2">
		<p>{t('plan.noRoom')} {t('plan.stillWorks')}</p>
		<div class="flex flex-wrap gap-2">
			{#if upgrade}
				<Button size="sm" href={upgrade} rel="noopener noreferrer">{t('quota.upgrade')}</Button>
			{/if}
			<Button size="sm" variant="ghost" onclick={() => (refusal = null)}>{t('common.dismiss')}</Button>
		</div>
	</Alert>
{/if}
