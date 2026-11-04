<script lang="ts">
	import { Alert, Button } from '@facile/muse';
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
	<Alert tone="warning" title="Your plan is full" class="mb-3 flex flex-col gap-2">
		<p>{refusal.message ?? 'Your plan has no room for this.'} Reading, downloading and deleting still work.</p>
		<div class="flex flex-wrap gap-2">
			{#if upgrade}
				<Button size="sm" href={upgrade} rel="noopener noreferrer">Upgrade plan</Button>
			{/if}
			<Button size="sm" variant="ghost" onclick={() => (refusal = null)}>Dismiss</Button>
		</div>
	</Alert>
{/if}
