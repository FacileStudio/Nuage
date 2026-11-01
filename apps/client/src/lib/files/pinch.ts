export type PinchOptions = {
	onPinch: (factor: number, centerX: number, centerY: number) => void;
};

const span = (touches: TouchList) =>
	Math.hypot(touches[0].clientX - touches[1].clientX, touches[0].clientY - touches[1].clientY);

/**
 * Svelte action: turns a two-finger pinch into `onPinch(factor, centerX, centerY)`, with the
 * factor accumulated per animation frame so a heavy renderer is asked once per frame, not once
 * per touch event. The scroller needs `touch-action: pan-x pan-y`, otherwise the browser zooms
 * the whole page instead of handing the gesture over.
 */
export function pinch(node: HTMLElement, initial: PinchOptions) {
	let options = initial;
	let last = 0;
	let pending = 1;
	let center = { x: 0, y: 0 };
	let frame = 0;

	const flush = () => {
		frame = 0;
		const factor = pending;
		pending = 1;
		options.onPinch(factor, center.x, center.y);
	};

	const start = (e: TouchEvent) => {
		last = e.touches.length === 2 ? span(e.touches) : 0;
	};

	const move = (e: TouchEvent) => {
		if (e.touches.length !== 2 || !last) return;
		if (e.cancelable) e.preventDefault();
		const next = span(e.touches);
		pending *= next / last;
		last = next;
		center = {
			x: (e.touches[0].clientX + e.touches[1].clientX) / 2,
			y: (e.touches[0].clientY + e.touches[1].clientY) / 2
		};
		if (!frame) frame = requestAnimationFrame(flush);
	};

	const end = () => {
		last = 0;
	};

	node.addEventListener('touchstart', start, { passive: true });
	node.addEventListener('touchmove', move, { passive: false });
	node.addEventListener('touchend', end);
	node.addEventListener('touchcancel', end);
	return {
		update(next: PinchOptions) {
			options = next;
		},
		destroy() {
			if (frame) cancelAnimationFrame(frame);
			node.removeEventListener('touchstart', start);
			node.removeEventListener('touchmove', move);
			node.removeEventListener('touchend', end);
			node.removeEventListener('touchcancel', end);
		}
	};
}
