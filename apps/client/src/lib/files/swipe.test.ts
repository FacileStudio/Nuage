import { expect, test } from 'bun:test';
import { swipeDirection } from './swipe';

test('swipeDirection reads a quick horizontal flick and refuses the rest', () => {
	expect(swipeDirection(-120, 10, 200)).toBe(1);
	expect(swipeDirection(120, -10, 200)).toBe(-1);
	expect(swipeDirection(-30, 0, 100)).toBeNull();
	expect(swipeDirection(-120, 0, 900)).toBeNull();
	expect(swipeDirection(-100, 90, 200)).toBeNull();
});
