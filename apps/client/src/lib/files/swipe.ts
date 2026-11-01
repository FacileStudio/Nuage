export const SWIPE_DISTANCE = 56;
export const SWIPE_MAX_MS = 700;
const HORIZONTAL_BIAS = 1.5;
const IGNORED = 'input, button, a, textarea, select, [data-no-swipe]';

/**
 * Reads one finger's travel as a horizontal swipe. Returns `1` for a swipe to the left (go to
 * the next item), `-1` for a swipe to the right, and `null` for anything too short, too slow
 * or too diagonal to be a deliberate flick.
 */
export function swipeDirection(dx: number, dy: number, ms: number): 1 | -1 | null {
	if (ms > SWIPE_MAX_MS || Math.abs(dx) < SWIPE_DISTANCE) return null;
	if (Math.abs(dx) < Math.abs(dy) * HORIZONTAL_BIAS) return null;
	return dx < 0 ? 1 : -1;
}

export type SwipeOptions = {
	onSwipe: (direction: 1 | -1) => void;
	enabled?: () => boolean;
};

/**
 * Svelte action: calls `onSwipe` when a single touch or pen pointer flicks across the node.
 * A second finger cancels the gesture so a pinch never navigates, and a gesture that starts
 * on a control (slider, button, link) is left to that control.
 */
export function swipe(node: HTMLElement, initial: SwipeOptions) {
	let options = initial;
	let start: { id: number; x: number; y: number; at: number } | null = null;
	const active = new Set<number>();

	const down = (e: PointerEvent) => {
		active.add(e.pointerId);
		const blocked = (e.target as Element | null)?.closest(IGNORED);
		const refused = e.pointerType === 'mouse' || blocked || options.enabled?.() === false;
		start =
			active.size > 1 || refused
				? null
				: { id: e.pointerId, x: e.clientX, y: e.clientY, at: e.timeStamp };
	};

	const up = (e: PointerEvent) => {
		active.delete(e.pointerId);
		const begun = start;
		start = null;
		if (!begun || begun.id !== e.pointerId) return;
		const direction = swipeDirection(e.clientX - begun.x, e.clientY - begun.y, e.timeStamp - begun.at);
		if (direction) options.onSwipe(direction);
	};

	const cancel = (e: PointerEvent) => {
		active.delete(e.pointerId);
		start = null;
	};

	node.addEventListener('pointerdown', down);
	node.addEventListener('pointerup', up);
	node.addEventListener('pointercancel', cancel);
	return {
		update(next: SwipeOptions) {
			options = next;
		},
		destroy() {
			node.removeEventListener('pointerdown', down);
			node.removeEventListener('pointerup', up);
			node.removeEventListener('pointercancel', cancel);
		}
	};
}
