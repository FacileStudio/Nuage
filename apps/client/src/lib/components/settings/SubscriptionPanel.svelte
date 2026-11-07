<script lang="ts">
	import { SettingsRow, SubscriptionSection } from '@facile/muse';
	import type { QuotaResponse } from '$lib/backend';
	import { formatSize } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';
	import { planName, upgradeHref } from '$lib/plan';

	let { quota }: { quota: QuotaResponse | null } = $props();

	const GIGABYTE = 1024 ** 3;
	const gigabytes = (bytes: number) => Math.round((bytes / GIGABYTE) * 100) / 100;

	const limited = $derived(quota !== null && quota.storage_limit > 0);
	const usage = $derived(
		quota && limited
			? [{ label: t('settings.subscription.storage'), used: gigabytes(quota.storage_used), limit: gigabytes(quota.storage_limit) }]
			: []
	);
</script>

<SubscriptionSection plan={planName(quota)} manageHref={upgradeHref(quota)} {usage}>
	{#if quota && !limited}
		<SettingsRow label={t('settings.subscription.storageLabel')} description={t('quota.used', { used: formatSize(quota.storage_used) })} />
	{/if}
</SubscriptionSection>
