<script lang="ts">
	import { Button } from '@facile/muse';
	import { previewKind, type ViewerFile } from '$lib/files/entries';
	import { t } from '$lib/i18n/index.svelte';
	import { fileIcon, fileTypeLabel, icons } from '$lib/icons';
	import ImageViewer from './ImageViewer.svelte';
	import MediaPlayer from './MediaPlayer.svelte';
	import PdfViewer from './PdfViewer.svelte';

	let {
		file,
		url,
		zoomed = $bindable(false)
	}: { file: ViewerFile; url: string; zoomed?: boolean } = $props();

	const kind = $derived(previewKind(file.name, file.mime_type));
</script>

{#if kind === 'image'}
	<ImageViewer {url} name={file.name} bind:zoomed />
{:else if kind === 'pdf'}
	<div class="h-full w-full max-w-5xl"><PdfViewer {url} /></div>
{:else if kind === 'video' || kind === 'audio'}
	<MediaPlayer {url} {kind} name={file.name} />
{:else}
	<div class="flex max-w-sm flex-col items-center gap-4 text-center">
		<span class="flex size-24 items-center justify-center rounded-fc-lg bg-fc-component text-fc-fg-muted">
			<iconify-icon icon={fileIcon(file.name, file.mime_type)} width="40" height="40" class="block"
			></iconify-icon>
		</span>
		<div class="flex min-w-0 flex-col gap-1">
			<p class="text-fc-lg font-semibold break-all text-fc-fg">{file.name}</p>
			<p class="text-fc-sm text-fc-fg-muted">
				{fileTypeLabel(file.name, file.mime_type)} · {t('viewer.noPreview')}
			</p>
		</div>
		<Button href={url} download={file.name} size="lg" icon={icons.download}>{t('viewer.download')}</Button>
	</div>
{/if}
