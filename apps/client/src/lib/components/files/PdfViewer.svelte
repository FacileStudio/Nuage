<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Divider, IconButton, Spinner } from '@facile/muse';
	import { icons, nuage } from '$lib/icons';

	let { url }: { url: string } = $props();

	let canvas = $state<HTMLCanvasElement | null>(null);
	let doc = $state<any>(null);
	let pageNum = $state(1);
	let totalPages = $state(0);
	let scale = $state(1);
	let fitScale = $state(1);
	let loading = $state(true);

	/*
	 * What the gesture shows before the render catches up. A pdf.js page costs tens of
	 * milliseconds to rasterise, so a pinch cannot drive `scale` frame by frame: it stretches
	 * the bitmap already on screen and commits one real render when the fingers lift. `scale`
	 * and `translate` are separate CSS properties rather than one `transform` on purpose —
	 * that lets the swipe spring back on a transition while the pinch snaps, with no
	 * transition, onto the freshly drawn page.
	 */
	let pinchScale = $state(1);
	let pinchOrigin = $state('50% 50%');
	let swipeX = $state(0);
	let swiping = $state(false);

	/*
	 * pdf.js hands back a document that owns a worker and a chunk of transferred memory. It is
	 * released on `destroy()` and nowhere else — the previous implementation dropped the
	 * reference on close and left the worker alive, so opening a dozen PDFs in a session leaked
	 * a dozen workers for the life of the tab. `disposed` guards the async gap: the component
	 * can unmount while `getDocument` is still in flight.
	 */
	let disposed = false;

	/*
	 * The task pdf.js is currently drawing with, kept so the next draw can cancel it. Two
	 * `render()` calls on one canvas make pdf.js throw, which a fast click on the page arrows
	 * was already enough to trigger.
	 *
	 * Deliberately not `$state`. `draw` reads it synchronously, and `draw` is called from the
	 * render effect, so a reactive version becomes one of that effect's dependencies: assigning
	 * the new task re-runs the effect, which cancels the task it just started and assigns
	 * another, forever. Plain, like `disposed`.
	 */
	let renderTask: any = null;

	type Anchor = { nx: number; ny: number; vx: number; vy: number };

	/*
	 * Everything the gestures carry between events is plain for the same reason `renderTask`
	 * is: the render effect calls `draw` synchronously, so any rune these touched would join
	 * its dependency set and a pointer move would start re-rasterising pages.
	 */
	let box: HTMLElement | null = null;
	let naturalWidth = 0;
	let fitted = true;
	let anchor: Anchor | null = null;
	let pinch: (Anchor & { ids: [number, number]; span: number }) | null = null;
	let drag: {
		id: number;
		x: number;
		y: number;
		lx: number;
		ly: number;
		mode: 'idle' | 'swipe' | 'pan';
		mouse: boolean;
	} | null = null;
	let tap = { time: 0, x: 0, y: 0 };
	const pointers = new Map<number, { x: number; y: number }>();

	const MIN_SCALE = 0.25;
	const MAX_SCALE = 4;
	const ZOOM_STEP = 1.25;
	const DRAG_SLOP = 8;
	const SWIPE_COMMIT = 64;
	const SWIPE_RESIST = 0.35;
	const TAP_WINDOW = 320;
	const TAP_SLOP = 24;
	const SCROLL_STEP = 64;

	$effect(() => {
		const target = url;
		loading = true;
		void load(target);
	});

	/*
	 * Drawing is reactive rather than called by hand. The canvas does not exist until `loading`
	 * flips and Svelte flushes the DOM, and `load` used to call the draw on that same tick: with
	 * `bind:this` not yet run, `canvas` was still null, the draw returned early and the first
	 * page stayed blank until a page change re-entered it with the element bound.
	 */
	$effect(() => {
		const target = canvas;
		const current = doc;
		const page = pageNum;
		const zoomLevel = scale;
		if (!target || !current) return;
		void draw(current, target, page, zoomLevel);
	});

	async function load(target: string) {
		const pdfjs = await import('pdfjs-dist');
		pdfjs.GlobalWorkerOptions.workerSrc = '/pdf.worker.min.mjs';
		const loaded = await pdfjs.getDocument(target).promise;

		if (disposed) {
			void loaded.destroy();
			return;
		}

		void doc?.destroy();

		const first = await loaded.getPage(1);
		naturalWidth = first.getViewport({ scale: 1 }).width;
		fitted = true;
		fit();

		/*
		 * The scale is settled before `doc` lands, so the render effect fires once, at the
		 * size the page will keep. Assigning them the other way round draws the page at the
		 * previous document's zoom and immediately cancels it.
		 */
		pageNum = 1;
		totalPages = loaded.numPages;
		doc = loaded;
		loading = false;
	}

	/*
	 * Fit is measured against the element the canvas actually sits in, not the window. The old
	 * `window.innerWidth * 0.8` capped at 900 was neither: inside a modal it was wrong by
	 * whatever the modal's own width happened to be, and on a phone it overflowed the box every
	 * time. A ResizeObserver keeps it honest through rotation and window resizes.
	 */
	function fit() {
		const width = box?.clientWidth ?? 0;
		if (!width || !naturalWidth) return;
		fitScale = width / naturalWidth;
		if (fitted) scale = fitScale;
	}

	function surface(node: HTMLElement) {
		box = node;
		const observer = new ResizeObserver(() => fit());
		observer.observe(node);
		/* Non-passive because a trackpad pinch arrives as ctrl+wheel and has to be claimed
		   before the browser zooms the whole page. */
		node.addEventListener('wheel', onWheel, { passive: false });
		return {
			destroy() {
				observer.disconnect();
				node.removeEventListener('wheel', onWheel);
				box = null;
			}
		};
	}

	/*
	 * A canvas has two sizes: the CSS box it occupies and the bitmap behind it. Sizing that bitmap
	 * in CSS pixels hands a HiDPI screen a half or a third of the pixels it can show and lets the
	 * browser upscale the difference, which is the blur. Past a certain area the allocation fails
	 * and the canvas paints nothing at all, so the ratio is reduced to fit rather than refusing to
	 * draw: a soft page beats a blank one. 2^24 is the total-pixel ceiling iOS Safari enforces and
	 * the long-standing pdf.js `maxCanvasPixels` default.
	 */
	const MAX_CANVAS_PIXELS = 16777216;

	async function draw(current: any, target: HTMLCanvasElement, page: number, zoomLevel: number) {
		renderTask?.cancel();

		const rendered = await current.getPage(page);
		const logical = rendered.getViewport({ scale: zoomLevel });
		const budget = Math.sqrt(MAX_CANVAS_PIXELS / (logical.width * logical.height));
		const ratio = Math.min(window.devicePixelRatio || 1, budget);
		const device = rendered.getViewport({ scale: zoomLevel * ratio });

		target.width = Math.floor(device.width);
		target.height = Math.floor(device.height);
		target.style.width = `${logical.width}px`;
		target.style.height = `${logical.height}px`;
		settle(target, logical.width, logical.height);

		const ctx = target.getContext('2d')!;
		/* The page is drawn onto white on purpose: a PDF's own background is transparent, and
		   letting a dark surface show through inverts nothing and just makes the text vanish. */
		ctx.fillStyle = '#ffffff';
		ctx.fillRect(0, 0, target.width, target.height);

		const task = rendered.render({ canvasContext: ctx, viewport: device });
		renderTask = task;
		try {
			await task.promise;
		} catch (err: any) {
			/* A cancellation is this component asking for it; anything else is worth seeing. */
			if (err?.name !== 'RenderingCancelledException') throw err;
		} finally {
			if (renderTask === task) renderTask = null;
		}
	}

	/*
	 * Hands the page back from the gesture to the bitmap that just replaced it. Clearing the
	 * CSS scale any earlier snaps the page to its old size for a frame; clearing it here, in the
	 * same block that sets the new canvas dimensions, means the two swap in one paint. The
	 * anchor then puts the point the user zoomed on, or the top of a page they turned to, back
	 * where they expect it. Both run after the first `await`, so neither is a tracked read.
	 */
	function settle(target: HTMLCanvasElement, width: number, height: number) {
		pinchScale = 1;
		pinchOrigin = '50% 50%';
		const held = anchor;
		anchor = null;
		if (!held || !box) return;
		box.scrollLeft = target.offsetLeft + held.nx * width - held.vx;
		box.scrollTop = target.offsetTop + held.ny * height - held.vy;
	}

	onDestroy(() => {
		disposed = true;
		renderTask?.cancel();
		void doc?.destroy();
		doc = null;
	});

	function step(delta: number) {
		const next = pageNum + delta;
		if (next < 1 || next > totalPages) return;
		pageNum = next;
		anchor = { nx: 0.5, ny: 0, vx: (box?.clientWidth ?? 0) / 2, vy: 0 };
	}

	function clamp(next: number) {
		return Math.min(Math.max(next, MIN_SCALE), MAX_SCALE);
	}

	function zoomAt(next: number, clientX: number, clientY: number) {
		const target = canvas;
		const value = clamp(next);
		if (!target || !box || value === scale) return;
		const page = target.getBoundingClientRect();
		const frame = box.getBoundingClientRect();
		anchor = {
			nx: page.width ? (clientX - page.left) / page.width : 0.5,
			ny: page.height ? (clientY - page.top) / page.height : 0.5,
			vx: clientX - frame.left,
			vy: clientY - frame.top
		};
		scale = value;
	}

	function zoomCentre(next: number) {
		if (!box) {
			scale = clamp(next);
			return;
		}
		const frame = box.getBoundingClientRect();
		zoomAt(next, frame.left + frame.width / 2, frame.top + frame.height / 2);
	}

	function zoomBy(factor: number) {
		fitted = false;
		zoomCentre(scale * factor);
	}

	function resetFit() {
		fitted = true;
		zoomCentre(fitScale);
	}

	function onWheel(event: WheelEvent) {
		if (loading || !event.ctrlKey) return;
		event.preventDefault();
		fitted = false;
		zoomAt(scale * Math.exp(-event.deltaY / 300), event.clientX, event.clientY);
	}

	function typing(target: EventTarget | null): boolean {
		if (!(target instanceof HTMLElement)) return false;
		const tag = target.tagName;
		return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || target.isContentEditable;
	}

	/*
	 * Escape is missing on purpose: the muse Modal around this component owns it, and binding
	 * it here would close the dialog twice. Up and down are here because the scroll box cannot
	 * take a `tabindex` — a11y rejects one on a non-interactive role — so this is what a
	 * keyboard has to reach the rest of a tall page with.
	 */
	function onKey(event: KeyboardEvent) {
		if (loading || event.ctrlKey || event.metaKey || event.altKey) return;
		if (typing(event.target)) return;
		if (event.key === 'ArrowLeft' || event.key === 'PageUp') step(-1);
		else if (event.key === 'ArrowRight' || event.key === 'PageDown') step(1);
		else if (event.key === 'ArrowUp') scrollPage(-SCROLL_STEP);
		else if (event.key === 'ArrowDown') scrollPage(SCROLL_STEP);
		else if (event.key === '+' || event.key === '=') zoomBy(ZOOM_STEP);
		else if (event.key === '-') zoomBy(1 / ZOOM_STEP);
		else if (event.key === '0') resetFit();
		else return;
		event.preventDefault();
	}

	function scrollPage(delta: number) {
		if (box) box.scrollTop += delta;
	}

	function startPinch() {
		const target = canvas;
		if (!target || !box) return;
		const ids = [...pointers.keys()].slice(0, 2) as [number, number];
		const a = pointers.get(ids[0])!;
		const b = pointers.get(ids[1])!;
		const mid = { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
		const page = target.getBoundingClientRect();
		const frame = box.getBoundingClientRect();
		pinch = {
			ids,
			span: Math.hypot(a.x - b.x, a.y - b.y) || 1,
			nx: page.width ? (mid.x - page.left) / page.width : 0.5,
			ny: page.height ? (mid.y - page.top) / page.height : 0.5,
			vx: mid.x - frame.left,
			vy: mid.y - frame.top
		};
		pinchOrigin = `${mid.x - page.left}px ${mid.y - page.top}px`;
		drag = null;
		swiping = false;
		swipeX = 0;
	}

	function trackPinch() {
		if (!pinch) return;
		const a = pointers.get(pinch.ids[0]);
		const b = pointers.get(pinch.ids[1]);
		if (!a || !b) return;
		/* Clamped against the committed range so the preview never promises a zoom the
		   render will refuse. */
		pinchScale = clamp((scale * Math.hypot(a.x - b.x, a.y - b.y)) / pinch.span) / scale;
	}

	function endPinch() {
		const held = pinch;
		pinch = null;
		if (!held) return;
		const next = clamp(scale * pinchScale);
		if (next === scale) {
			pinchScale = 1;
			pinchOrigin = '50% 50%';
			return;
		}
		fitted = false;
		anchor = { nx: held.nx, ny: held.ny, vx: held.vx, vy: held.vy };
		scale = next;
	}

	function resist(dx: number): number {
		const blocked = dx > 0 ? pageNum <= 1 : pageNum >= totalPages;
		return blocked ? dx * SWIPE_RESIST : dx;
	}

	function trackDrag(event: PointerEvent) {
		if (!drag || !box) return;
		const dx = event.clientX - drag.x;
		const dy = event.clientY - drag.y;
		if (drag.mode === 'idle') {
			if (Math.abs(dx) < DRAG_SLOP && Math.abs(dy) < DRAG_SLOP) return;
			const spills = box.scrollWidth > box.clientWidth + 1;
			drag.mode = !spills && Math.abs(dx) > Math.abs(dy) ? 'swipe' : 'pan';
			swiping = drag.mode === 'swipe';
		}
		if (drag.mode === 'swipe') {
			swipeX = resist(dx);
		} else {
			box.scrollLeft -= event.clientX - drag.lx;
			/* Touch keeps the browser's own vertical scrolling, momentum and all, through
			   `touch-pan-y`. A mouse has no scrollbar to reach for — they are hidden suite-wide
			   — so it is the one pointer that has to drag the other axis by hand. */
			if (drag.mouse) box.scrollTop -= event.clientY - drag.ly;
		}
		drag.lx = event.clientX;
		drag.ly = event.clientY;
	}

	function onPointerDown(event: PointerEvent) {
		if (loading) return;
		/* A release the browser never reports leaves a phantom finger in the map, and the next
		   real touch then reads as a pinch. The primary pointer starts every gesture, so it is
		   the one place a reset is always safe. */
		if (event.isPrimary) pointers.clear();
		pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });
		if (pointers.size === 2) {
			startPinch();
			return;
		}
		if (pointers.size > 2 || pinch) return;
		/* Touch pointers are captured to their target by the spec; a mouse is not, and without
		   this a drag that leaves the box never reports its release. */
		const mouse = event.pointerType !== 'touch';
		if (mouse) (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
		drag = {
			id: event.pointerId,
			x: event.clientX,
			y: event.clientY,
			lx: event.clientX,
			ly: event.clientY,
			mode: 'idle',
			mouse
		};
	}

	function onPointerMove(event: PointerEvent) {
		const point = pointers.get(event.pointerId);
		if (!point) return;
		point.x = event.clientX;
		point.y = event.clientY;
		if (pinch) {
			trackPinch();
			return;
		}
		if (drag?.id === event.pointerId) trackDrag(event);
	}

	function toggleZoom(event: PointerEvent) {
		if (!fitted) {
			resetFit();
			return;
		}
		fitted = false;
		zoomAt(1, event.clientX, event.clientY);
	}

	/* One path for double-tap and double-click: pointer events report a mouse, a finger and a
	   pen the same way, so the touch case never needs a second detector beside `dblclick`. */
	function onTap(event: PointerEvent) {
		const near = Math.hypot(event.clientX - tap.x, event.clientY - tap.y) < TAP_SLOP;
		if (near && event.timeStamp - tap.time < TAP_WINDOW) {
			tap = { time: 0, x: 0, y: 0 };
			toggleZoom(event);
			return;
		}
		tap = { time: event.timeStamp, x: event.clientX, y: event.clientY };
	}

	function onPointerUp(event: PointerEvent) {
		pointers.delete(event.pointerId);
		if (pinch && pointers.size < 2) endPinch();
		if (drag?.id !== event.pointerId) return;
		const { mode, x } = drag;
		const dx = event.clientX - x;
		drag = null;
		swiping = false;
		swipeX = 0;
		if (mode === 'idle') onTap(event);
		else if (mode === 'swipe' && Math.abs(dx) >= SWIPE_COMMIT) step(dx > 0 ? -1 : 1);
	}

	function onPointerCancel(event: PointerEvent) {
		pointers.delete(event.pointerId);
		if (pinch && pointers.size < 2) endPinch();
		if (drag?.id !== event.pointerId) return;
		drag = null;
		swiping = false;
		swipeX = 0;
	}
</script>

<svelte:window onkeydown={onKey} />

<div class="relative flex min-h-0 w-full flex-1 flex-col">
	<!--
	  `touch-pan-y` is the whole mobile compromise in one class. Vertical scrolling stays the
	  browser's, so reading a page keeps its momentum, while every horizontal move and every
	  second finger reaches the handlers below: a swipe when the page fits, a pan when it does
	  not, a pinch either way.
	-->
	<div
		use:surface
		onpointerdown={onPointerDown}
		onpointermove={onPointerMove}
		onpointerup={onPointerUp}
		onpointercancel={onPointerCancel}
		role="region"
		aria-label="Document page"
		class="relative min-h-0 w-full flex-1 touch-pan-y overflow-auto overscroll-contain rounded-fc-md bg-fc-surface"
	>
		<canvas
			bind:this={canvas}
			style:translate="{swipeX}px"
			style:scale={pinchScale}
			style:transform-origin={pinchOrigin}
			class="mx-auto block {swiping ? '' : 'transition-[translate] duration-200 ease-fc'}"
		></canvas>
	</div>

	{#if loading}
		<div class="absolute inset-0 flex items-center justify-center rounded-fc-md bg-fc-surface">
			<Spinner />
		</div>
	{:else}
		<!--
		  Floating rather than stacked. A row of chrome above the page costs a phone 56px out of
		  a 75dvh box for something it needs twice a document; over the page it costs nothing,
		  and the wrapper stays `pointer-events-none` so the band either side of the pill is
		  still swipeable. Glass and a pill, following MobileNav — the suite's one floating bar.

		  Two departures from MobileNav, both because what sits behind this bar is a white PDF
		  page rather than the app's own themed chrome: the fill is 85% rather than 70%, and the
		  contents wear `text-fc-fg` rather than the muted tone. At MobileNav's numbers a dark
		  theme leaves muted grey on a backdrop the blur has pulled halfway to white, which
		  measures 3.8:1.
		-->
		<div class="pointer-events-none absolute inset-x-0 bottom-3 z-10 flex justify-center px-2">
			<div
				class="pointer-events-auto flex max-w-full items-center gap-0.5 overflow-x-auto rounded-fc-pill bg-fc-bg/85 p-1 text-fc-fg shadow-lg backdrop-blur-2xl backdrop-saturate-150 sm:gap-1 sm:p-1.5"
			>
				<IconButton
					variant="ghost"
					onclick={() => step(-1)}
					disabled={pageNum <= 1}
					aria-label="Previous page"
					class="text-fc-fg"
				>
					<iconify-icon icon={icons.chevronLeft} width="20" height="20"></iconify-icon>
				</IconButton>
				<span aria-live="polite" class="min-w-14 shrink-0 text-center text-fc-xs tabular-nums">
					{pageNum} / {totalPages}
				</span>
				<IconButton
					variant="ghost"
					onclick={() => step(1)}
					disabled={pageNum >= totalPages}
					aria-label="Next page"
					class="text-fc-fg"
				>
					<iconify-icon icon={icons.arrow} width="20" height="20"></iconify-icon>
				</IconButton>

				<!--
				  The zoom trio is desktop-only. On a phone pinch and double-tap already do it,
				  and three more targets would push the pill past the 336px a 360px screen has.
				-->
				<Divider class="mx-1 my-0 hidden h-5 w-px shrink-0 border-t-0 border-l sm:block" />
				<IconButton
					variant="ghost"
					onclick={() => zoomBy(1 / ZOOM_STEP)}
					aria-label="Zoom out"
					class="hidden text-fc-fg sm:inline-flex"
				>
					<iconify-icon icon={icons.minus} width="18" height="18"></iconify-icon>
				</IconButton>
				<span class="hidden min-w-12 shrink-0 text-center text-fc-xs tabular-nums sm:block">
					{Math.round(scale * 100)}%
				</span>
				<IconButton
					variant="ghost"
					onclick={() => zoomBy(ZOOM_STEP)}
					aria-label="Zoom in"
					class="hidden text-fc-fg sm:inline-flex"
				>
					<iconify-icon icon={icons.plus} width="18" height="18"></iconify-icon>
				</IconButton>

				<Divider class="mx-1 my-0 h-5 w-px shrink-0 border-t-0 border-l" />
				<IconButton
					variant="ghost"
					onclick={resetFit}
					aria-label="Fit to width"
					class="text-fc-fg"
				>
					<iconify-icon icon={nuage.fullscreen} width="18" height="18"></iconify-icon>
				</IconButton>
			</div>
		</div>
	{/if}
</div>
