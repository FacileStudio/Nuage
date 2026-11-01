import { expect, test } from 'bun:test';
import { formatDuration } from './format';

test('formatDuration pads seconds and switches to hours past the hour', () => {
	expect(formatDuration(0)).toBe('0:00');
	expect(formatDuration(65.9)).toBe('1:05');
	expect(formatDuration(3725)).toBe('1:02:05');
	expect(formatDuration(Number.NaN)).toBe('0:00');
});
