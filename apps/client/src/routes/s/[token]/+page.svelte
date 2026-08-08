<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { Button, Card, EmptyState, Spinner, Table } from '@facile/muse';
	import {
		backend,
		type Folder,
		type NuageFile,
		type PublicShareResponse
	} from '$lib/backend';
	import { formatSize } from '$lib/format';
	import { fileIcon, fileTypeLabel, icons, nuage } from '$lib/icons';

	let loading = $state(true);
	let error = $state('');
	let share = $state<PublicShareResponse | null>(null);

	let folderFiles = $state<NuageFile[]>([]);
	let folderFolders = $state<Folder[]>([]);
	let folderLoading = $state(false);
	let breadcrumbs = $state<{ id: number | null; name: string }[]>([]);

	let token = $derived(page.params.token ?? '');

	onMount(async () => {
		try {
			share = await backend.getPublicShare(token);
			if (share.folder) {
				breadcrumbs = [{ id: null, name: share.folder.name }];
				await loadFolderContents();
			}
		} catch {
			error = 'This link has expired or does not exist';
		}
		loading = false;
	});

	async function loadFolderContents(folderId?: number) {
		folderLoading = true;
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

	async function openSubfolder(folder: Folder) {
		breadcrumbs = [...breadcrumbs, { id: folder.id, name: folder.name }];
		await loadFolderContents(folder.id);
	}

	async function navigateBreadcrumb(index: number) {
		const crumb = breadcrumbs[index];
		breadcrumbs = breadcrumbs.slice(0, index + 1);
		await loadFolderContents(crumb.id ?? undefined);
	}
</script>

<svelte:head>
	<title>{share?.file?.name ?? share?.folder?.name ?? 'Shared'} — Nuage</title>
</svelte:head>

<div class="flex min-h-dvh flex-col">
	<header class="flex items-center gap-2 px-6 pt-6">
		<iconify-icon icon={nuage.brand} width="24" height="24" class="block text-fc-fg"></iconify-icon>
		<span class="font-fc-title text-fc-xl font-semibold tracking-tight text-fc-fg">Nuage</span>
	</header>

	<main class="flex flex-1 items-center justify-center px-4 py-8">
		{#if loading}
			<Spinner size="lg" />
		{:else if error}
			<EmptyState
				bare
				icon={nuage.shareOff}
				title={error}
				description="Check the address, or ask whoever sent it for a fresh link."
			/>
		{:else if share?.file}
			{@const file = share.file}
			<Card class="flex w-full max-w-xl flex-col gap-6">
				{#if file.mime_type.startsWith('image/')}
					<img
						src={backend.publicDownloadUrl(token, file.id)}
						alt={file.name}
						class="mx-auto max-h-[400px] rounded-fc-md bg-fc-surface object-contain"
					/>
				{:else}
					<span
						class="mx-auto flex size-24 items-center justify-center rounded-fc-lg bg-fc-surface text-fc-fg-muted"
					>
						<iconify-icon icon={fileIcon(file.name, file.mime_type)} width="40" height="40" class="block"
						></iconify-icon>
					</span>
				{/if}

				<div class="flex flex-col gap-1 text-center">
					<h1 class="text-fc-lg font-semibold break-all text-fc-fg">{file.name}</h1>
					<p class="text-fc-sm text-fc-fg-muted">
						{fileTypeLabel(file.name, file.mime_type)} · {formatSize(file.size)}
					</p>
				</div>

				<!-- A download is a navigation, so it is an anchor. `Button href` keeps it
				     middle-clickable and shows the target in the status bar (CHARTE §8). -->
				<Button
					size="lg"
					class="w-full"
					icon={icons.download}
					href={backend.publicDownloadUrl(token, file.id)}
					download={file.name}
				>
					Download
				</Button>
			</Card>
		{:else if share?.folder}
			<div class="flex w-full max-w-3xl flex-col gap-4">
				<nav class="flex flex-wrap items-center gap-1.5 text-fc-sm" aria-label="Breadcrumb">
					{#each breadcrumbs as crumb, i (i)}
						{#if i > 0}
							<iconify-icon icon={icons.arrow} width="14" height="14" class="block shrink-0 text-fc-fg-muted"
							></iconify-icon>
						{/if}
						{#if i === breadcrumbs.length - 1}
							<span class="truncate font-medium text-fc-fg" aria-current="page">{crumb.name}</span>
						{:else}
							<button
								type="button"
								class="truncate rounded-fc-sm text-fc-fg-muted transition-colors hover:text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
								onclick={() => navigateBreadcrumb(i)}
							>
								{crumb.name}
							</button>
						{/if}
					{/each}
				</nav>

				{#if folderLoading}
					<div class="flex h-48 items-center justify-center"><Spinner /></div>
				{:else if folderFolders.length === 0 && folderFiles.length === 0}
					<EmptyState icon={nuage.folderOpen} title="This folder is empty" />
				{:else}
					<Table>
						<thead>
							<tr>
								<th scope="col">Name</th>
								<th scope="col" class="hidden sm:table-cell">Size</th>
								<th scope="col" class="w-16 text-right"><span class="sr-only">Download</span></th>
							</tr>
						</thead>
						<tbody>
							{#each folderFolders as folder (folder.id)}
								<tr>
									<td colspan="3" class="p-0">
										<!-- The whole row is one button: a click handler on a `<tr>` is not
										     reachable by keyboard, which is how this shipped. -->
										<button
											type="button"
											class="flex w-full items-center gap-3 px-3 py-2 text-left transition-colors hover:bg-fc-surface focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-fc-ring"
											onclick={() => openSubfolder(folder)}
										>
											<iconify-icon icon={icons.folder} width="18" height="18" class="block shrink-0 text-fc-fg-muted"
											></iconify-icon>
											<span class="truncate font-medium">{folder.name}</span>
										</button>
									</td>
								</tr>
							{/each}
							{#each folderFiles as file (file.id)}
								<tr>
									<td>
										<div class="flex min-w-0 items-center gap-3">
											<iconify-icon
												icon={fileIcon(file.name, file.mime_type)}
												width="18"
												height="18"
												class="block shrink-0 text-fc-fg-muted"
											></iconify-icon>
											<span class="truncate font-medium">{file.name}</span>
										</div>
									</td>
									<td class="hidden text-fc-fg-muted sm:table-cell">{formatSize(file.size)}</td>
									<td class="text-right">
										<a
											href={backend.publicDownloadUrl(token, file.id)}
											download={file.name}
											aria-label="Download {file.name}"
											class="inline-flex size-11 items-center justify-center rounded-fc-pill text-fc-fg-muted transition-colors hover:bg-fc-surface hover:text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
										>
											<iconify-icon icon={icons.download} width="18" height="18" class="block"></iconify-icon>
										</a>
									</td>
								</tr>
							{/each}
						</tbody>
					</Table>
				{/if}
			</div>
		{/if}
	</main>

	<footer class="flex items-center justify-center gap-2 px-6 pb-6">
		<iconify-icon icon={nuage.brand} width="14" height="14" class="block text-fc-fg-muted"></iconify-icon>
		<span class="text-fc-xs text-fc-fg-muted">Nuage</span>
	</footer>
</div>
