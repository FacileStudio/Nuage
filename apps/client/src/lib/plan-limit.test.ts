import { expect, test } from 'bun:test';
import { isSafeUpgradeUrl, notePlanLimit, planLimitFrom, planLimits } from './plan-limit';

test('planLimitFrom reads the 402 envelope and drops unsafe links', () => {
	const wire = {
		error: { code: 'payment_required', message: 'Full', key: 'nuage.storage_gb', limit: 5, upgrade_url: 'https://sso.facile.studio/spaces' }
	};
	expect(planLimitFrom(wire)).toEqual({
		message: 'Full',
		key: 'nuage.storage_gb',
		limit: 5,
		upgrade_url: 'https://sso.facile.studio/spaces'
	});
	expect(planLimitFrom({ error: { message: 'm', upgrade_url: 'javascript:alert(1)' } })?.upgrade_url).toBeUndefined();
	expect(planLimitFrom(null)).toBeNull();
	expect(planLimitFrom({})).toBeNull();
});

test('isSafeUpgradeUrl accepts https and same-origin paths only', () => {
	expect(isSafeUpgradeUrl('/spaces')).toBe(true);
	expect(isSafeUpgradeUrl('https://sso.facile.studio/spaces/spc_1')).toBe(true);
	expect(isSafeUpgradeUrl('//evil.example')).toBe(false);
	expect(isSafeUpgradeUrl('http://sso.facile.studio/spaces')).toBe(false);
	expect(isSafeUpgradeUrl('https://user:pw@sso.facile.studio')).toBe(false);
	expect(isSafeUpgradeUrl('data:text/html,x')).toBe(false);
});

test('notePlanLimit announces only a 402', () => {
	const seen: unknown[] = [];
	planLimits.addEventListener('plan-limit', (e: Event) => seen.push((e as CustomEvent).detail));
	notePlanLimit(413, { error: { message: 'too big' } });
	notePlanLimit(402, { error: { message: 'Full', key: 'nuage.storage_gb' } });
	expect(seen).toEqual([{ message: 'Full', key: 'nuage.storage_gb', upgrade_url: undefined, limit: undefined }]);
});
