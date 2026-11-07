import { expect, test } from 'bun:test';
import { formatDuration } from './format';

test('formatDuration pads seconds and switches to hours past the hour', () => {
	expect(formatDuration(0)).toBe('0:00');
	expect(formatDuration(65.9)).toBe('1:05');
	expect(formatDuration(3725)).toBe('1:02:05');
	expect(formatDuration(Number.NaN)).toBe('0:00');
});

test('formatSize follows the locale for unit and decimal separator', async () => {
	const { bindActive } = await import('./i18n/active');
	const { formatSize } = await import('./format');
	bindActive(() => 'en');
	expect(formatSize(1536)).toBe('1.5 KB');
	bindActive(() => 'fr');
	expect(formatSize(1536)).toBe('1,5 Ko');
	expect(formatSize(0)).toBe('0 o');
	bindActive(() => 'en');
});
