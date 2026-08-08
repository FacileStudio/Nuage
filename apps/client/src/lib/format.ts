/** Human-readable byte size, e.g. `1.4 GB`. Bytes stay whole; every larger unit gets one decimal. */
export function formatSize(bytes: number): string {
	if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
	return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

/** Locale date for a timestamp the API returns as ISO-8601. */
export function formatDate(iso: string): string {
	const date = new Date(iso);
	return Number.isNaN(date.getTime()) ? '—' : date.toLocaleDateString();
}

/** Locale date and time, for anything where the hour matters. */
export function formatDateTime(iso: string): string {
	const date = new Date(iso);
	return Number.isNaN(date.getTime())
		? '—'
		: date.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
}

const RELATIVE_STEPS: [limit: number, unit: Intl.RelativeTimeFormatUnit, per: number][] = [
	[60, 'second', 1],
	[3600, 'minute', 60],
	[86400, 'hour', 3600],
	[604800, 'day', 86400],
	[2629800, 'week', 604800],
	[31557600, 'month', 2629800]
];

/**
 * "3 minutes ago" through `Intl.RelativeTimeFormat`, so the wording follows the reader's
 * locale instead of a hand-rolled English ladder.
 */
export function relativeTime(iso: string): string {
	const then = new Date(iso).getTime();
	if (Number.isNaN(then)) return '—';

	const seconds = Math.round((then - Date.now()) / 1000);
	const magnitude = Math.abs(seconds);
	const format = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });

	if (magnitude < 45) return format.format(0, 'second');
	for (const [limit, unit, per] of RELATIVE_STEPS) {
		if (magnitude < limit) return format.format(Math.round(seconds / per), unit);
	}
	return format.format(Math.round(seconds / 31557600), 'year');
}
