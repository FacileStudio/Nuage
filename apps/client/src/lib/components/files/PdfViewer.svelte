<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Divider, Spinner } from '@facile/muse';
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
	 * pdf.js hands back a document that owns a worker and a chunk of transferred memory. It is
	 * released on `destroy()` and nowhere else — the previous implementation dropped the
	 * reference on close and left the worker alive, so opening a dozen PDFs in a session leaked
	 * a dozen workers for the life of the tab. `disposed` guards the async gap: the component
	 * can unmount while `getDocument` is still in flight.
	 */
	let disposed = false;

	$effect(() => {
		const target = url;
		loading = true;
		void load(target);
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
		doc = loaded;
		totalPages = loaded.numPages;
		pageNum = 1;

		const first = await loaded.getPage(1);
		const natural = first.getViewport({ scale: 1 });
		fitScale = Math.min(window.innerWidth * 0.8, 900) / natural.width;
		scale = fitScale;
		loading = false;
		await render();
	}

	async function render() {
		if (!doc || !canvas) return;
		const page = await doc.getPage(pageNum);
		const viewport = page.getViewport({ scale });
		canvas.width = viewport.width;
		canvas.height = viewport.height;
		const ctx = canvas.getContext('2d')!;
		/* The page is drawn onto white on purpose: a PDF's own background is transparent, and
		   letting a dark surface show through inverts nothing and just makes the text vanish. */
		ctx.fillStyle = '#ffffff';
		ctx.fillRect(0, 0, viewport.width, viewport.height);
		await page.render({ canvasContext: ctx, viewport }).promise;
	}

	onDestroy(() => {
		disposed = true;
		void doc?.destroy();
		doc = null;
	});

	function step(delta: number) {
		const next = pageNum + delta;
		if (next < 1 || next > totalPages) return;
		pageNum = next;
		void render();
	}

	function zoom(next: number) {
		scale = Math.min(Math.max(next, 0.25), 4);
		void render();
	}
</script>

{#if loading}
	<div class="flex h-64 items-center justify-center"><Spinner /></div>
{:else}
	<div class="flex min-h-0 flex-1 flex-col items-center gap-3">
		<div class="flex items-center gap-1 rounded-fc-md bg-fc-surface px-2 py-1.5">
			<button
				type="button"
				onclick={() => step(-1)}
				disabled={pageNum <= 1}
				aria-label="Previous page"
				class="flex size-8 items-center justify-center rounded-fc-sm transition-colors hover:bg-fc-component disabled:opacity-30 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
			>
				<iconify-icon icon={icons.chevronLeft} width="18" height="18" class="block"></iconify-icon>
			</button>
			<span class="min-w-16 text-center text-fc-xs tabular-nums text-fc-fg-muted">
				{pageNum} / {totalPages}
			</span>
			<button
				type="button"
				onclick={() => step(1)}
				disabled={pageNum >= totalPages}
				aria-label="Next page"
				class="flex size-8 items-center justify-center rounded-fc-sm transition-colors hover:bg-fc-component disabled:opacity-30 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
			>
				<iconify-icon icon={icons.arrow} width="18" height="18" class="block"></iconify-icon>
			</button>

			<Divider class="mx-1 h-4 w-px border-l border-t-0" />

			<button
				type="button"
				onclick={() => zoom(scale - 0.25)}
				aria-label="Zoom out"
				class="flex size-8 items-center justify-center rounded-fc-sm transition-colors hover:bg-fc-component focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
			>
				<iconify-icon icon={icons.minus} width="16" height="16" class="block"></iconify-icon>
			</button>
			<span class="min-w-12 text-center text-fc-xs tabular-nums text-fc-fg-muted">
				{Math.round(scale * 100)}%
			</span>
			<button
				type="button"
				onclick={() => zoom(scale + 0.25)}
				aria-label="Zoom in"
				class="flex size-8 items-center justify-center rounded-fc-sm transition-colors hover:bg-fc-component focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
			>
				<iconify-icon icon={icons.plus} width="16" height="16" class="block"></iconify-icon>
			</button>
			<button
				type="button"
				onclick={() => zoom(fitScale)}
				aria-label="Fit to width"
				title="Fit to width"
				class="flex size-8 items-center justify-center rounded-fc-sm transition-colors hover:bg-fc-component focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
			>
				<iconify-icon icon={nuage.fullscreen} width="16" height="16" class="block"></iconify-icon>
			</button>
		</div>

		<div class="min-h-0 flex-1 overflow-auto rounded-fc-md bg-fc-surface">
			<canvas bind:this={canvas} class="mx-auto block"></canvas>
		</div>
	</div>
{/if}
