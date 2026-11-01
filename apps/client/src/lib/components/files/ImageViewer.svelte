<script lang="ts">
	import { Spinner } from '@facile/muse';
	import { nuage } from '$lib/icons';

	let { url, name }: { url: string; name: string } = $props();

	let scale = $state(1);
	let x = $state(0);
	let y = $state(0);
	let rotation = $state(0);
	let loaded = $state(false);
	let failed = $state(false);
	let dragging = $state(false);
	let stage: HTMLDivElement | null = null;
	const pointers = new Map<number, { x: number; y: number }>();
	let pinchSpan = 0;

	const MIN = 1;
	const MAX = 8;

	$effect(() => {
		void url;
		zoomTo(1);
		rotation = 0;
		loaded = false;
		failed = false;
	});

	function zoomTo(next: number, originX = 0, originY = 0) {
		const clamped = Math.min(MAX, Math.max(MIN, next));
		const ratio = clamped / scale;
		x = clamped === 1 ? 0 : originX - (originX - x) * ratio;
		y = clamped === 1 ? 0 : originY - (originY - y) * ratio;
		scale = clamped;
	}

	function fromCenter(clientX: number, clientY: number) {
		const rect = stage!.getBoundingClientRect();
		return [clientX - rect.left - rect.width / 2, clientY - rect.top - rect.height / 2] as const;
	}

	function onDoubleClick(e: MouseEvent) {
		const [ox, oy] = fromCenter(e.clientX, e.clientY);
		zoomTo(scale > 1 ? 1 : 2.5, ox, oy);
	}

	function onPointerDown(e: PointerEvent) {
		pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		if (pointers.size === 2) {
			const [a, b] = [...pointers.values()];
			pinchSpan = Math.hypot(a.x - b.x, a.y - b.y);
		}
		dragging = scale > 1 || pointers.size === 2;
	}

	function onPointerMove(e: PointerEvent) {
		const last = pointers.get(e.pointerId);
		if (!last) return;
		pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
		if (pointers.size === 2) {
			const [a, b] = [...pointers.values()];
			const next = Math.hypot(a.x - b.x, a.y - b.y);
			const [ox, oy] = fromCenter((a.x + b.x) / 2, (a.y + b.y) / 2);
			zoomTo((scale * next) / pinchSpan, ox, oy);
			pinchSpan = next;
		} else if (scale > 1) {
			x += e.clientX - last.x;
			y += e.clientY - last.y;
		}
	}

	function onPointerUp(e: PointerEvent) {
		pointers.delete(e.pointerId);
		dragging = pointers.size > 0 && scale > 1;
	}

	function stageAction(node: HTMLDivElement) {
		stage = node;
		const onWheel = (e: WheelEvent) => {
			e.preventDefault();
			const [ox, oy] = fromCenter(e.clientX, e.clientY);
			zoomTo(scale * Math.exp(-e.deltaY * (e.ctrlKey ? 0.01 : 0.002)), ox, oy);
		};
		node.addEventListener('wheel', onWheel, { passive: false });
		return { destroy: () => node.removeEventListener('wheel', onWheel) };
	}
</script>

<div class="relative flex h-full w-full flex-col">
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		use:stageAction
		class="relative flex min-h-0 flex-1 touch-none items-center justify-center overflow-hidden select-none {scale > 1
			? dragging
				? 'cursor-grabbing'
				: 'cursor-grab'
			: 'cursor-zoom-in'}"
		ondblclick={onDoubleClick}
		onpointerdown={onPointerDown}
		onpointermove={onPointerMove}
		onpointerup={onPointerUp}
		onpointercancel={onPointerUp}
	>
		{#if !loaded && !failed}
			<div class="absolute inset-0 flex items-center justify-center"><Spinner /></div>
		{/if}
		{#if failed}
			<p class="text-fc-sm text-fc-fg-muted">This image could not be displayed.</p>
		{:else}
			<img
				src={url}
				alt={name}
				draggable="false"
				onload={() => (loaded = true)}
				onerror={() => (failed = true)}
				class="max-h-full max-w-full object-contain shadow-lg {dragging ? '' : 'transition-transform duration-200 ease-out'} {loaded
					? 'opacity-100'
					: 'opacity-0'}"
				style="transform: translate({x}px, {y}px) scale({scale}) rotate({rotation}deg)"
			/>
		{/if}
	</div>

	<div
		class="absolute bottom-4 left-1/2 flex -translate-x-1/2 items-center gap-1 rounded-fc-pill border border-fc-border bg-fc-component/90 p-1 text-fc-fg shadow-lg backdrop-blur"
	>
		<button type="button" class="viewer-tool" aria-label="Zoom out" disabled={scale <= MIN} onclick={() => zoomTo(scale / 1.5)}>
			<iconify-icon icon={nuage.zoomOut} width="18" height="18" class="block"></iconify-icon>
		</button>
		<button type="button" class="viewer-tool w-14 text-fc-xs font-medium tabular-nums" aria-label="Reset zoom" onclick={() => {
			zoomTo(1);
			rotation = 0;
		}}>
			{Math.round(scale * 100)}%
		</button>
		<button type="button" class="viewer-tool" aria-label="Zoom in" disabled={scale >= MAX} onclick={() => zoomTo(scale * 1.5)}>
			<iconify-icon icon={nuage.zoomIn} width="18" height="18" class="block"></iconify-icon>
		</button>
		<span class="mx-1 h-5 w-px bg-fc-border"></span>
		<button type="button" class="viewer-tool" aria-label="Rotate" onclick={() => (rotation = (rotation + 90) % 360)}>
			<iconify-icon icon={nuage.rotate} width="18" height="18" class="block"></iconify-icon>
		</button>
	</div>
</div>
