<script lang="ts">
	import { Button, Modal } from '@facile/muse';
	import type { NuageFile } from '$lib/backend';
	import { icons } from '$lib/icons';
	import PdfViewer from './PdfViewer.svelte';

	let {
		file,
		url,
		onClose
	}: {
		file: NuageFile;
		url: string;
		onClose: () => void;
	} = $props();

	let open = $state(true);

	/* `max-w-lg` is Modal's largest size and a media lightbox needs more than that; twMerge
	   resolves the conflict in favour of the class passed in. Everything else — the focus
	   trap, Escape, the scroll lock, the backdrop hit-test — comes from Modal unchanged. */
	const wide = 'max-w-5xl w-[min(92vw,64rem)]';
</script>

<Modal bind:open {onClose} class={wide}>
	{#snippet header()}
		<h2 class="truncate pr-2 text-fc-lg font-semibold text-fc-fg">{file.name}</h2>
	{/snippet}

	<div class="flex max-h-[75dvh] min-h-0 flex-col items-center gap-3">
		{#if file.mime_type.startsWith('image/')}
			<img src={url} alt={file.name} class="max-h-[70dvh] max-w-full rounded-fc-md object-contain" />
		{:else if file.mime_type === 'application/pdf'}
			<PdfViewer {url} />
		{:else if file.mime_type.startsWith('video/')}
			<!-- svelte-ignore a11y_media_has_caption -->
			<video controls src={url} class="max-h-[70dvh] max-w-full rounded-fc-md"></video>
		{:else if file.mime_type.startsWith('audio/')}
			<audio controls src={url} class="w-full max-w-80"></audio>
		{/if}
	</div>

	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (open = false)}>Close</Button>
			<Button href={url} download={file.name} icon={icons.download}>Download</Button>
		</div>
	{/snippet}
</Modal>
