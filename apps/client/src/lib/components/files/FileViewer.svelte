<script lang="ts">
	import { onMount } from 'svelte';
	import { Button } from '@facile/muse';
	import { backend, type NuageFile } from '$lib/backend';
	import { formatSize, relativeTime } from '$lib/format';
	import { fileIcon, fileTypeLabel, icons, nuage } from '$lib/icons';
	import ImageViewer from './ImageViewer.svelte';
	import MediaPlayer from './MediaPlayer.svelte';
	import PdfViewer from './PdfViewer.svelte';
	import { previewKind } from '$lib/files/entries';

	let {
		files,
		startId,
		onClose,
		onShare
	}: {
		files: NuageFile[];
		startId: number;
		onClose: () => void;
		onShare?: (file: NuageFile) => void;
	} = $props();

	let dialog = $state<HTMLDialogElement | null>(null);
	let index = $state(0);

	const file = $derived(files[index]);
	const url = $derived(file ? backend.downloadUrl(file.id) : '');
	const kind = $derived(file ? previewKind(file.name, file.mime_type) : null);

	$effect.pre(() => {
		const found = files.findIndex((f) => f.id === startId);
		index = found < 0 ? 0 : found;
	});

	onMount(() => {
		dialog?.showModal();
		return () => dialog?.close();
	});

	const step = (delta: number) => {
		if (files.length > 1) index = (index + delta + files.length) % files.length;
	};

	function onKeydown(e: KeyboardEvent) {
		if (e.target instanceof HTMLInputElement) return;
		if (e.key === 'ArrowLeft') step(-1);
		if (e.key === 'ArrowRight') step(1);
	}
</script>

<dialog
	bind:this={dialog}
	aria-label={file?.name ?? 'File preview'}
	onclose={onClose}
	onkeydown={onKeydown}
	class="m-0 h-dvh max-h-none w-full max-w-none bg-fc-page p-0 text-fc-fg backdrop:bg-fc-scrim"
>
	{#if file}
		<div class="flex h-full flex-col">
			<header class="flex items-center gap-3 border-b border-fc-border bg-fc-component px-2 py-2 sm:px-4">
				<button type="button" class="viewer-tool size-11" aria-label="Close preview" onclick={() => dialog?.close()}>
					<iconify-icon icon={icons.close} width="20" height="20" class="block"></iconify-icon>
				</button>
				<span class="hidden size-9 shrink-0 items-center justify-center rounded-fc-sm bg-fc-surface text-fc-fg-muted sm:flex">
					<iconify-icon icon={fileIcon(file.name, file.mime_type)} width="18" height="18" class="block"></iconify-icon>
				</span>
				<div class="min-w-0 flex-1">
					<h2 class="truncate text-fc-sm font-semibold sm:text-fc-md" title={file.name}>{file.name}</h2>
					<p class="truncate text-fc-xs text-fc-fg-muted">
						{fileTypeLabel(file.name, file.mime_type)} · {formatSize(file.size)} · {relativeTime(file.updated_at)}
					</p>
				</div>
				{#if files.length > 1}
					<span class="hidden text-fc-xs text-fc-fg-muted tabular-nums md:block">{index + 1} / {files.length}</span>
				{/if}
				{#if onShare}
					<button type="button" class="viewer-tool size-11" aria-label="Share" onclick={() => onShare(file)}>
						<iconify-icon icon={nuage.share} width="20" height="20" class="block"></iconify-icon>
					</button>
				{/if}
				<Button href={url} download={file.name} icon={icons.download} class="hidden sm:inline-flex">Download</Button>
				<a href={url} download={file.name} class="viewer-tool size-11 sm:hidden" aria-label="Download">
					<iconify-icon icon={icons.download} width="20" height="20" class="block"></iconify-icon>
				</a>
			</header>

			<div class="relative flex min-h-0 flex-1 items-center justify-center bg-fc-surface p-2 sm:p-6">
				{#key file.id}
					{#if kind === 'image'}
						<ImageViewer {url} name={file.name} />
					{:else if kind === 'pdf'}
						<div class="h-full w-full max-w-5xl"><PdfViewer {url} /></div>
					{:else if kind === 'video' || kind === 'audio'}
						<MediaPlayer {url} {kind} name={file.name} />
					{/if}
				{/key}

				{#if files.length > 1}
					<button
						type="button"
						aria-label="Previous file"
						onclick={() => step(-1)}
						class="absolute top-1/2 left-2 hidden size-11 -translate-y-1/2 items-center justify-center rounded-fc-pill border border-fc-border bg-fc-component/90 text-fc-fg shadow-lg backdrop-blur transition-colors hover:bg-fc-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring sm:flex"
					>
						<iconify-icon icon={nuage.previous} width="22" height="22" class="block"></iconify-icon>
					</button>
					<button
						type="button"
						aria-label="Next file"
						onclick={() => step(1)}
						class="absolute top-1/2 right-2 hidden size-11 -translate-y-1/2 items-center justify-center rounded-fc-pill border border-fc-border bg-fc-component/90 text-fc-fg shadow-lg backdrop-blur transition-colors hover:bg-fc-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring sm:flex"
					>
						<iconify-icon icon={nuage.next} width="22" height="22" class="block"></iconify-icon>
					</button>
				{/if}
			</div>
		</div>
	{/if}
</dialog>
