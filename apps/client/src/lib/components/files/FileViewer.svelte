<script lang="ts">
	import { onMount } from 'svelte';
	import { Button, Tooltip } from '@facile/muse';
	import { formatSize, relativeTime } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';
	import { fileIcon, fileTypeLabel, icons, nuage } from '$lib/icons';
	import { previewKind, type ViewerFile } from '$lib/files/entries';
	import { swipe } from '$lib/files/swipe';
	import ViewerStage from './ViewerStage.svelte';

	let {
		files,
		startId,
		urlFor,
		onClose,
		onShare
	}: {
		files: ViewerFile[];
		startId: number;
		urlFor: (file: ViewerFile) => string;
		onClose: () => void;
		onShare?: (file: ViewerFile) => void;
	} = $props();

	let dialog = $state<HTMLDialogElement | null>(null);
	let index = $state(0);
	let zoomed = $state(false);

	const file = $derived(files[index]);
	const url = $derived(file ? urlFor(file) : '');
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

	const canSwipe = () => kind !== 'pdf' && !(kind === 'image' && zoomed);

	function onKeydown(e: KeyboardEvent) {
		if (e.target instanceof HTMLInputElement) return;
		if (e.key === 'ArrowLeft') step(-1);
		if (e.key === 'ArrowRight') step(1);
	}
</script>

<dialog
	bind:this={dialog}
	aria-label={file?.name ?? t('viewer.preview')}
	onclose={onClose}
	onkeydown={onKeydown}
	class="m-0 h-dvh max-h-none w-full max-w-none overflow-hidden overscroll-none bg-fc-page p-0 text-fc-fg backdrop:bg-fc-scrim"
>
	{#if file}
		<div
			class="flex h-full flex-col pt-[env(safe-area-inset-top)] pr-[env(safe-area-inset-right)] pb-[env(safe-area-inset-bottom)] pl-[env(safe-area-inset-left)]"
		>
			<header class="flex items-center gap-3 bg-fc-component px-2 py-2 sm:px-4">
				<Tooltip text={t('viewer.close')} describe={false} class="contents">
					<button type="button" class="viewer-tool size-11" aria-label={t('viewer.close')} onclick={() => dialog?.close()}>
						<iconify-icon icon={icons.close} width="20" height="20" class="block"></iconify-icon>
					</button>
				</Tooltip>
				<span class="hidden size-9 shrink-0 items-center justify-center rounded-fc-sm bg-fc-surface text-fc-fg-muted sm:flex">
					<iconify-icon icon={fileIcon(file.name, file.mime_type)} width="18" height="18" class="block"></iconify-icon>
				</span>
				<div class="min-w-0 flex-1">
					<h2 class="truncate text-fc-sm font-bold sm:text-fc-md" title={file.name}>{file.name}</h2>
					<p class="truncate text-fc-xs text-fc-fg-muted">
						{fileTypeLabel(file.name, file.mime_type)} · {formatSize(file.size)}{file.updated_at
							? ` · ${relativeTime(file.updated_at)}`
							: ''}
					</p>
				</div>
				{#if files.length > 1}
					<span class="hidden text-fc-xs text-fc-fg-muted tabular-nums md:block">{index + 1} / {files.length}</span>
				{/if}
				{#if onShare}
					<Tooltip text={t('viewer.share')} describe={false} class="contents">
						<button type="button" class="viewer-tool size-11" aria-label={t('viewer.share')} onclick={() => onShare(file)}>
							<iconify-icon icon={nuage.share} width="20" height="20" class="block"></iconify-icon>
						</button>
					</Tooltip>
				{/if}
				<Button href={url} download={file.name} icon={icons.download} class="hidden sm:inline-flex">{t('viewer.download')}</Button>
				<Tooltip text={t('viewer.download')} describe={false} class="contents">
					<a href={url} download={file.name} class="viewer-tool size-11 sm:hidden" aria-label={t('viewer.download')}>
						<iconify-icon icon={icons.download} width="20" height="20" class="block"></iconify-icon>
					</a>
				</Tooltip>
			</header>

			<div
				use:swipe={{ onSwipe: (direction) => step(direction), enabled: canSwipe }}
				class="relative flex min-h-0 flex-1 touch-pan-y items-center justify-center bg-fc-surface p-2 sm:p-6"
			>
				{#key file.id}
					<ViewerStage {file} {url} bind:zoomed />
				{/key}

				{#if files.length > 1}
					<Tooltip text={t('viewer.previous')} describe={false} class="contents">
						<button
							type="button"
							aria-label={t('viewer.previous')}
							onclick={() => step(-1)}
							class="absolute top-1/2 left-2 hidden size-11 -translate-y-1/2 items-center justify-center rounded-fc-pill bg-fc-component/90 text-fc-fg shadow-lg backdrop-blur transition-colors hover:bg-fc-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring sm:flex"
						>
							<iconify-icon icon={nuage.previous} width="22" height="22" class="block"></iconify-icon>
						</button>
					</Tooltip>
					<Tooltip text={t('viewer.next')} describe={false} class="contents">
						<button
							type="button"
							aria-label={t('viewer.next')}
							onclick={() => step(1)}
							class="absolute top-1/2 right-2 hidden size-11 -translate-y-1/2 items-center justify-center rounded-fc-pill bg-fc-component/90 text-fc-fg shadow-lg backdrop-blur transition-colors hover:bg-fc-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring sm:flex"
						>
							<iconify-icon icon={nuage.next} width="22" height="22" class="block"></iconify-icon>
						</button>
					</Tooltip>
				{/if}
			</div>

			{#if files.length > 1}
				<nav
					aria-label={t('viewer.siblings')}
					class="flex items-center justify-between bg-fc-component px-2 sm:hidden"
				>
					<Tooltip text={t('viewer.previous')} describe={false} class="contents">
						<button type="button" class="viewer-tool" aria-label={t('viewer.previous')} onclick={() => step(-1)}>
							<iconify-icon icon={nuage.previous} width="22" height="22" class="block"></iconify-icon>
						</button>
					</Tooltip>
					<span class="text-fc-xs text-fc-fg-muted tabular-nums">{index + 1} / {files.length}</span>
					<Tooltip text={t('viewer.next')} describe={false} class="contents">
						<button type="button" class="viewer-tool" aria-label={t('viewer.next')} onclick={() => step(1)}>
							<iconify-icon icon={nuage.next} width="22" height="22" class="block"></iconify-icon>
						</button>
					</Tooltip>
				</nav>
			{/if}
		</div>
	{/if}
</dialog>
