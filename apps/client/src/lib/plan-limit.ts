import { REGISTRE_ORIGIN } from '$lib/registre';

/** A plan refusal read from a 402 body: what was refused and where to upgrade. */
export type PlanLimit = {
	message?: string;
	upgrade_url?: string;
	key?: string;
	limit?: number | string;
};

type Envelope = Record<string, unknown>;

const isRecord = (value: unknown): value is Envelope => typeof value === 'object' && value !== null;

const text = (value: unknown): string | undefined =>
	typeof value === 'string' && value.trim() !== '' ? value : undefined;

const amount = (value: unknown): number | string | undefined =>
	typeof value === 'number' && Number.isFinite(value) ? value : text(value);

/**
 * True for a same-origin path, or an https URL on Registre's exact origin (the OIDC issuer)
 * with no credentials. Any other host, and `javascript:` or `data:`, never reaches an `href`.
 */
export function isSafeUpgradeUrl(raw: string | undefined): raw is string {
	if (!raw || /[\s\\\p{Cc}]/u.test(raw)) return false;
	if (raw.startsWith('/')) return !raw.startsWith('//');
	try {
		const url = new URL(raw);
		return url.origin === REGISTRE_ORIGIN && !url.username && !url.password;
	} catch {
		return false;
	}
}

/**
 * Reads a 402 body, in tronc's `{"error":{...}}` envelope or bare, into a `PlanLimit`, or
 * `null` when it is not one. An unsafe `upgrade_url` is dropped.
 */
export function planLimitFrom(body: unknown): PlanLimit | null {
	if (!isRecord(body)) return null;
	const inner = isRecord(body.error) ? body.error : body;
	const upgrade = text(inner.upgrade_url);
	const parsed: PlanLimit = {
		message: text(inner.message),
		upgrade_url: isSafeUpgradeUrl(upgrade) ? upgrade : undefined,
		key: text(inner.key),
		limit: amount(inner.limit)
	};
	return Object.values(parsed).every((v) => v === undefined) ? null : parsed;
}

/** Where plan refusals are announced: a `plan-limit` CustomEvent whose detail is the `PlanLimit`. */
export const planLimits = new EventTarget();

/** Announces a 402 body on `planLimits` so the banner shows it. Any other status is ignored. */
export function notePlanLimit(status: number, body: unknown): void {
	if (status !== 402) return;
	const detail = planLimitFrom(body) ?? { message: 'Your plan has no room for this.' };
	planLimits.dispatchEvent(new CustomEvent<PlanLimit>('plan-limit', { detail }));
}
