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
	/* Below `sm:` the dialog stops being a card and becomes the screen. A centred one leaves a
	   360px phone 291px of usable width for a page of text, and the header, a 75dvh body and
	   the footer together overflow the dialog's own height cap, so the sheet scrolls as well as
	   the page inside it. `m-0` replaces Modal's load-bearing `m-auto` only where the dialog
	   already covers the viewport and centring has nothing left to do, and `[&>div]:h-full`
	   reaches Modal's one padding wrapper so the body can claim the height the header and
	   footer leave. Both are restored at `sm:`. */
	const wide =
		'm-0 h-dvh max-h-dvh w-full max-w-none rounded-none border-0 [&>div]:h-full ' +
		'sm:m-auto sm:h-auto sm:max-h-[calc(100dvh-4rem)] sm:w-[min(92vw,64rem)] sm:max-w-5xl ' +
		'sm:rounded-fc-lg sm:border sm:[&>div]:h-auto';
</script>

<Modal bind:open {onClose} class={wide}>
	{#snippet header()}
		<h2 class="truncate pr-2 text-fc-lg font-semibold text-fc-fg">{file.name}</h2>
	{/snippet}

	<div
		class="flex min-h-0 w-full flex-1 flex-col items-center justify-center gap-3 sm:max-h-[75dvh]"
	>
		{#if file.mime_type.startsWith('image/')}
			<img
				src={url}
				alt={file.name}
				class="max-h-full max-w-full rounded-fc-md object-contain sm:max-h-[70dvh]"
			/>
		{:else if file.mime_type === 'application/pdf'}
			<PdfViewer {url} />
		{:else if file.mime_type.startsWith('video/')}
			<!-- svelte-ignore a11y_media_has_caption -->
			<video controls src={url} class="max-h-full max-w-full rounded-fc-md sm:max-h-[70dvh]"
			></video>
		{:else if file.mime_type.startsWith('audio/')}
			<audio controls src={url} class="w-full max-w-80"></audio>
		{/if}
	</div>

	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" size="lg" onclick={() => (open = false)}>Close</Button>
			<Button href={url} download={file.name} size="lg" icon={icons.download}>Download</Button>
		</div>
	{/snippet}
</Modal>
