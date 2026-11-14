<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { Button, Card, EmptyState, Spinner, Tooltip } from '@facile/muse';
	import {
		backend,
		type PublicFile,
		type PublicFolder,
		type PublicShareResponse
	} from '$lib/backend';
	import { isPreviewable, previewKind } from '$lib/files/entries';
	import { formatSize } from '$lib/format';
	import { fileIcon, fileTypeLabel, icons, nuage } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';
	import FileViewer from '$lib/components/files/FileViewer.svelte';
	import ViewerStage from '$lib/components/files/ViewerStage.svelte';

	let loading = $state(true);
	let error = $state('');
	let share = $state<PublicShareResponse | null>(null);

	let folderFiles = $state<PublicFile[]>([]);
	let folderFolders = $state<PublicFolder[]>([]);
	let folderLoading = $state(false);
	let breadcrumbs = $state<{ id: number | null; name: string }[]>([]);
	let viewing = $state<PublicFile | null>(null);
	let brokenThumbs = $state<Record<number, boolean>>({});

	let token = $derived(page.params.token ?? '');
	const previewable = $derived(folderFiles.filter(isPreviewable));
	const title = $derived(share?.file?.name ?? share?.folder?.name ?? t('public.shared'));

	onMount(async () => {
		try {
			share = await backend.getPublicShare(token);
			if (share.folder) {
				breadcrumbs = [{ id: null, name: share.folder.name }];
				await loadFolderContents();
			}
		} catch {
			error = t('public.expired');
		}
		loading = false;
	});

	async function loadFolderContents(folderId?: number) {
		folderLoading = true;
		brokenThumbs = {};
		try {
			const res = await backend.getPublicShareFiles(token, folderId);
			folderFiles = res.files ?? [];
			folderFolders = res.folders ?? [];
		} catch {
			folderFiles = [];
			folderFolders = [];
		}
		folderLoading = false;
	}

	async function openSubfolder(folder: PublicFolder) {
		breadcrumbs = [...breadcrumbs, { id: folder.id, name: folder.name }];
		await loadFolderContents(folder.id);
	}

	async function navigateBreadcrumb(index: number) {
		const crumb = breadcrumbs[index];
		breadcrumbs = breadcrumbs.slice(0, index + 1);
		await loadFolderContents(crumb.id ?? undefined);
	}

	const urlFor = (file: { id: number }) => backend.publicDownloadUrl(token, file.id);

	function shareThumb(file: PublicFile): { src: string; srcset?: string } | null {
		if (previewKind(file.name, file.mime_type) !== 'image' || brokenThumbs[file.id]) return null;
		return {
			src: backend.publicThumbnailUrl(token, file),
			srcset: backend.publicThumbnailSrcset(token, file)
		};
	}
</script>

<svelte:head>
	<meta name="robots" content="noindex, nofollow" />
	<title>{t('public.title', { name: title })}</title>
</svelte:head>

{#snippet brand()}
	<span class="flex shrink-0 items-center gap-2">
		<iconify-icon icon={nuage.brand} width="24" height="24" class="block text-fc-fg"></iconify-icon>
		<span class="font-fc-title text-fc-xl font-bold tracking-tight text-fc-fg max-sm:sr-only">
			Nuage
		</span>
	</span>
{/snippet}

{#snippet thumb(file: PublicFile)}
	{@const preview = shareThumb(file)}
	{#if preview}
		<img
			src={preview.src}
			srcset={preview.srcset}
			sizes="(min-width: 1024px) 25vw, (min-width: 640px) 33vw, 50vw"
			alt=""
			loading="lazy"
			decoding="async"
			onerror={() => (brokenThumbs[file.id] = true)}
			class="aspect-[4/3] w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
		/>
	{:else}
		<span class="flex aspect-[4/3] w-full items-center justify-center text-fc-fg-muted">
			<iconify-icon icon={fileIcon(file.name, file.mime_type)} width="40" height="40" class="block"
			></iconify-icon>
		</span>
	{/if}
{/snippet}

<div class="flex h-dvh flex-col bg-fc-page text-fc-fg">
	<header class="flex items-center gap-3 bg-fc-component px-4 py-2 sm:px-6">
		{@render brand()}

		{#if share?.file}
			{@const file = share.file}
			<span class="hidden h-6 w-px shrink-0 bg-fc-surface sm:block"></span>
			<div class="min-w-0 flex-1">
				<h1 class="truncate text-fc-sm font-bold sm:text-fc-md" title={file.name}>{file.name}</h1>
				<p class="truncate text-fc-xs text-fc-fg-muted">
					{fileTypeLabel(file.name, file.mime_type)} · {formatSize(file.size)}
				</p>
			</div>
			<Button
				href={urlFor(file)}
				download={file.name}
				icon={icons.download}
				class="hidden sm:inline-flex"
			>
				{t('viewer.download')}
			</Button>
			<Tooltip text={t('public.downloadFile', { name: file.name })} describe={false} class="contents">
				<a
					href={urlFor(file)}
					download={file.name}
					aria-label={t('public.downloadFile', { name: file.name })}
					class="viewer-tool size-11 sm:hidden"
				>
					<iconify-icon icon={icons.download} width="20" height="20" class="block"></iconify-icon>
				</a>
			</Tooltip>
		{/if}
	</header>

	{#if loading}
		<main class="flex flex-1 items-center justify-center"><Spinner size="lg" /></main>
	{:else if error}
		<main class="flex flex-1 items-center justify-center px-4">
			<EmptyState
				bare
				icon={nuage.shareOff}
				title={error}
				description={t('public.expiredHint')}
			/>
		</main>
	{:else if share?.file}
		<main
			class="relative flex min-h-0 flex-1 items-center justify-center bg-fc-surface p-2 sm:p-6"
		>
			{#key share.file.id}
				<ViewerStage file={share.file} url={urlFor(share.file)} />
			{/key}
		</main>
	{:else if share?.folder}
		<main class="min-h-0 flex-1 overflow-y-auto">
			<div class="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 sm:px-6">
				<nav class="flex flex-wrap items-center gap-1 text-fc-sm" aria-label={t('files.breadcrumb')}>
					{#each breadcrumbs as crumb, i (crumb.id ?? 'root')}
						{#if i > 0}
							<iconify-icon
								icon={icons.arrow}
								width="14"
								height="14"
								class="block shrink-0 text-fc-fg-muted"
							></iconify-icon>
						{/if}
						{#if i === breadcrumbs.length - 1}
							<span class="truncate px-2 font-medium text-fc-fg" aria-current="page">{crumb.name}</span>
						{:else}
							<button
								type="button"
								onclick={() => navigateBreadcrumb(i)}
								class="inline-flex min-h-11 items-center truncate rounded-fc-sm px-2 text-fc-fg-muted transition-colors hover:text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
							>
								{crumb.name}
							</button>
						{/if}
					{/each}
				</nav>

				{#if folderLoading}
					<div class="flex h-48 items-center justify-center"><Spinner /></div>
				{:else if folderFolders.length === 0 && folderFiles.length === 0}
					<EmptyState
						bare
						icon={nuage.folderOpen}
						title={t('public.emptyTitle')}
						description={t('public.emptyText')}
					/>
				{:else}
					{#if folderFolders.length > 0}
						<section class="flex flex-col gap-3" aria-label={t('files.folders')}>
							<h2 class="text-fc-xs font-medium tracking-wide text-fc-fg-muted uppercase">{t('files.folders')}</h2>
							<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
								{#each folderFolders as folder (folder.id)}
									<Card class="group relative flex items-center gap-3 p-0 transition-colors hover:bg-fc-surface">
										<button
											type="button"
											aria-label={t('files.open', { name: folder.name })}
											onclick={() => openSubfolder(folder)}
											class="absolute inset-0 rounded-fc-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
										></button>
										<div class="pointer-events-none flex min-h-14 min-w-0 flex-1 items-center gap-3 px-3">
											<iconify-icon
												icon={icons.folder}
												width="20"
												height="20"
												class="block shrink-0 text-fc-fg-muted"
											></iconify-icon>
											<span class="truncate text-fc-sm font-medium text-fc-fg">{folder.name}</span>
										</div>
									</Card>
								{/each}
							</div>
						</section>
					{/if}
					{#if folderFiles.length > 0}
						<section class="flex flex-col gap-3" aria-label={t('files.files')}>
							<h2 class="text-fc-xs font-medium tracking-wide text-fc-fg-muted uppercase">{t('files.files')}</h2>
							<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
								{#each folderFiles as file (file.id)}
									<Card class="group relative flex flex-col gap-0 p-0 transition-colors hover:bg-fc-surface">
										{#if isPreviewable(file)}
											<button
												type="button"
												aria-label={t('public.preview', { name: file.name })}
												onclick={() => (viewing = file)}
												class="m-2 mb-0 block overflow-hidden rounded-fc-sm bg-fc-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
											>
												{@render thumb(file)}
											</button>
										{:else}
											<a
												href={urlFor(file)}
												download={file.name}
												aria-label={t('public.downloadFile', { name: file.name })}
												class="m-2 mb-0 block overflow-hidden rounded-fc-sm bg-fc-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
											>
												{@render thumb(file)}
											</a>
										{/if}
										<div class="flex min-w-0 items-center gap-1 py-1.5 pr-1 pl-3">
											<div class="min-w-0 flex-1">
												<span class="block truncate text-fc-sm font-medium text-fc-fg" title={file.name}>
													{file.name}
												</span>
												<span class="block truncate text-fc-xs text-fc-fg-muted">{formatSize(file.size)}</span>
											</div>
											<Tooltip text={t('public.downloadFile', { name: file.name })} describe={false} class="contents">
												<a
													href={urlFor(file)}
													download={file.name}
													aria-label={t('public.downloadFile', { name: file.name })}
													class="viewer-tool size-11 text-fc-fg-muted hover:text-fc-fg"
												>
													<iconify-icon icon={icons.download} width="18" height="18" class="block"
													></iconify-icon>
												</a>
											</Tooltip>
										</div>
									</Card>
								{/each}
							</div>
						</section>
					{/if}
				{/if}
			</div>
		</main>
	{/if}
</div>

{#if viewing}
	<FileViewer files={previewable} startId={viewing.id} {urlFor} onClose={() => (viewing = null)} />
{/if}
