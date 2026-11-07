import type { QuotaResponse } from '$lib/backend';
import { t } from '$lib/i18n/index.svelte';
import { isSafeUpgradeUrl } from '$lib/plan-limit';

/** The name shown for where the storage limit comes from. The API reports a source, never a plan name. */
export function planName(quota: QuotaResponse | null): string {
	if (!quota) return '—';
	if (quota.source === 'plan') return t('settings.subscription.plan.plan');
	if (quota.source === 'override') return t('settings.subscription.plan.override');
	return t('settings.subscription.plan.instance');
}

/** Where the person changes their plan: the safe upgrade link, only when a plan sets the limit. */
export function upgradeHref(quota: QuotaResponse | null): string | undefined {
	return quota?.source === 'plan' && isSafeUpgradeUrl(quota.upgrade_url) ? quota.upgrade_url : undefined;
}
