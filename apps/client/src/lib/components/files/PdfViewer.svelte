<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Spinner } from '@facile/muse';
	import { icons, nuage } from '$lib/icons';
	import { openPdf, type PdfSession } from '$lib/files/pdf';
	import './pdf-viewer.css';

	let { url }: { url: string } = $props();

	let container = $state<HTMLDivElement | null>(null);
	let viewerEl = $state<HTMLDivElement | null>(null);
	let loading = $state(true);
	let failed = $state(false);
	let pageNum = $state(1);
	let pageInput = $state('1');
	let totalPages = $state(0);
	let scale = $state(1);
	let query = $state('');
	let matches = $state({ current: 0, total: 0 });
	let searchOpen = $state(false);
	let searchInput = $state<HTMLInputElement | null>(null);

	let session: PdfSession | null = null;
	let disposed = false;

	$effect(() => {
		const target = url;
		if (container && viewerEl) void load(target, container, viewerEl);
	});

	async function load(target: string, box: HTMLDivElement, pagesEl: HTMLDivElement) {
		loading = true;
		failed = false;
		try {
			const opened = await openPdf(target, box, pagesEl, {
				onPage: (page) => {
					pageNum = page;
					pageInput = String(page);
				},
				onScale: (value) => (scale = value),
				onMatches: (value) => (matches = value)
			});
			if (disposed) return opened.destroy();
			session?.destroy();
			session = opened;
			totalPages = opened.pages;
		} catch {
			failed = true;
		}
		loading = false;
	}

	function goTo(page: number) {
		if (session) pageInput = String(session.goTo(page));
	}

	function onSearchKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			session?.find(query, true, e.shiftKey);
		}
		if (e.key === 'Escape' && query) {
			e.stopPropagation();
			query = '';
			session?.find('');
		}
	}

	function openSearch() {
		searchOpen = true;
		queueMicrotask(() => searchInput?.focus());
	}

	function onKeydown(e: KeyboardEvent) {
		const mod = e.ctrlKey || e.metaKey;
		if (mod && e.key === 'f') {
			e.preventDefault();
			openSearch();
		} else if (mod && (e.key === '=' || e.key === '+' || e.key === '-')) {
			e.preventDefault();
			session?.zoom(e.key === '-' ? -1 : 1);
		} else if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
			e.stopPropagation();
		}
	}

	function wheel(node: HTMLElement) {
		const handler = (e: WheelEvent) => {
			if (!e.ctrlKey && !e.metaKey) return;
			e.preventDefault();
			session?.zoomBy(Math.exp(-e.deltaY * 0.01));
		};
		node.addEventListener('wheel', handler, { passive: false });
		return { destroy: () => node.removeEventListener('wheel', handler) };
	}

	onDestroy(() => {
		disposed = true;
		session?.destroy();
	});
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="flex h-full w-full flex-col overflow-hidden rounded-fc-lg bg-fc-surface" onkeydown={onKeydown}>
	<div class="flex flex-wrap items-center gap-1 border-b border-fc-border bg-fc-component px-2 py-1.5 text-fc-fg">
		<div class="flex items-center gap-1 text-fc-sm">
			<button type="button" class="viewer-tool" aria-label="Previous page" disabled={pageNum <= 1} onclick={() => goTo(pageNum - 1)}>
				<iconify-icon icon={nuage.previous} width="18" height="18" class="block"></iconify-icon>
			</button>
			<input
				bind:value={pageInput}
				onchange={() => goTo(Number(pageInput))}
				inputmode="numeric"
				aria-label="Page number"
				class="h-8 w-12 rounded-fc-sm border border-fc-border bg-fc-bg text-center text-fc-sm tabular-nums focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
			/>
			<span class="text-fc-fg-muted tabular-nums">/ {totalPages || '–'}</span>
			<button type="button" class="viewer-tool" aria-label="Next page" disabled={pageNum >= totalPages} onclick={() => goTo(pageNum + 1)}>
				<iconify-icon icon={nuage.next} width="18" height="18" class="block"></iconify-icon>
			</button>
		</div>

		<div class="flex-1"></div>

		{#if searchOpen}
			<div class="flex items-center gap-1">
				<input
					bind:this={searchInput}
					bind:value={query}
					oninput={() => session?.find(query)}
					onkeydown={onSearchKeydown}
					type="search"
					placeholder="Find in document"
					aria-label="Find in document"
					class="h-8 w-36 rounded-fc-sm border border-fc-border bg-fc-bg px-2 text-fc-sm sm:w-48 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
				/>
				{#if query}
					<span class="text-fc-xs text-fc-fg-muted tabular-nums">{matches.current}/{matches.total}</span>
				{/if}
				<button type="button" class="viewer-tool" aria-label="Previous match" onclick={() => session?.find(query, true, true)}>
					<iconify-icon icon={icons.chevronUp} width="18" height="18" class="block"></iconify-icon>
				</button>
				<button type="button" class="viewer-tool" aria-label="Next match" onclick={() => session?.find(query, true)}>
					<iconify-icon icon={icons.chevronDown} width="18" height="18" class="block"></iconify-icon>
				</button>
			</div>
		{:else}
			<button type="button" class="viewer-tool" aria-label="Find in document" onclick={openSearch}>
				<iconify-icon icon={icons.search} width="18" height="18" class="block"></iconify-icon>
			</button>
		{/if}

		<span class="mx-1 hidden h-5 w-px bg-fc-border sm:block"></span>

		<div class="flex items-center gap-1">
			<button type="button" class="viewer-tool" aria-label="Zoom out" onclick={() => session?.zoom(-1)}>
				<iconify-icon icon={nuage.zoomOut} width="18" height="18" class="block"></iconify-icon>
			</button>
			<span class="w-12 text-center text-fc-xs font-medium tabular-nums">{Math.round(scale * 100)}%</span>
			<button type="button" class="viewer-tool" aria-label="Zoom in" onclick={() => session?.zoom(1)}>
				<iconify-icon icon={nuage.zoomIn} width="18" height="18" class="block"></iconify-icon>
			</button>
			<button type="button" class="viewer-tool" aria-label="Fit to width" onclick={() => session?.fitWidth()}>
				<iconify-icon icon={nuage.fitWidth} width="18" height="18" class="block"></iconify-icon>
			</button>
		</div>
	</div>

	<div class="relative min-h-0 flex-1">
		<div bind:this={container} use:wheel class="absolute inset-0 overflow-auto px-3">
			<div bind:this={viewerEl} class="pdfViewer"></div>
		</div>
		{#if loading}
			<div class="absolute inset-0 flex items-center justify-center"><Spinner /></div>
		{:else if failed}
			<div class="absolute inset-0 flex items-center justify-center p-6 text-center text-fc-sm text-fc-fg-muted">
				This PDF could not be opened. Download it to view it in another app.
			</div>
		{/if}
	</div>
</div>
