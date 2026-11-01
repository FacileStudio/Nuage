<script lang="ts">
	import { Button, Modal, Spinner, toast } from '@facile/muse';
	import { backend, type Folder } from '$lib/backend';
	import type { Entry } from '$lib/files/selection.svelte';
	import { icons, nuage } from '$lib/icons';

	let {
		token,
		spaceId,
		items,
		originId,
		trail,
		onMove,
		onClose
	}: {
		token: string;
		spaceId: number | null;
		items: (Entry & { name: string })[];
		originId: number | null;
		trail: { id: number | null; name: string }[];
		onMove: (target: number | null, label: string) => void;
		onClose: () => void;
	} = $props();

	let open = $state(true);
	let path = $state<{ id: number | null; name: string }[]>([]);
	let folders = $state<Folder[]>([]);
	let loading = $state(true);

	const here = $derived(path[path.length - 1]);
	const movingFolders = $derived(
		new Set(items.filter((i) => i.type === 'folder').map((i) => i.id))
	);
	const title = $derived(items.length === 1 ? `Move “${items[0].name}”` : `Move ${items.length} items`);
	const unchanged = $derived(here?.id === originId);

	$effect(() => {
		path = trail.length > 0 ? [...trail] : [{ id: null, name: 'Files' }];
	});

	$effect(() => {
		const target = here;
		if (target) void load(target.id);
	});

	async function load(parentId: number | null) {
		loading = true;
		try {
			const res = await backend.listFolders(token, { parent_id: parentId, space_id: spaceId });
			folders = res.folders ?? [];
		} catch {
			folders = [];
			toast.danger('Could not load folders.');
		}
		loading = false;
	}

	function enter(folder: Folder) {
		path = [...path, { id: folder.id, name: folder.name }];
	}

	function confirm() {
		if (!here || unchanged) return;
		onMove(here.id, here.name);
		open = false;
	}
</script>

<Modal bind:open {onClose} {title} showClose class="sm:max-w-lg">
	<nav class="flex min-w-0 flex-wrap items-center gap-1 text-fc-sm" aria-label="Destination">
		{#each path as crumb, i (crumb.id ?? 'root')}
			{#if i > 0}
				<iconify-icon icon={icons.arrow} width="14" height="14" class="block text-fc-fg-muted"
				></iconify-icon>
			{/if}
			<button
				type="button"
				disabled={i === path.length - 1}
				onclick={() => (path = path.slice(0, i + 1))}
				class="truncate rounded-fc-sm px-1.5 py-0.5 text-fc-fg-muted transition-colors hover:text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring disabled:font-medium disabled:text-fc-fg"
			>
				{crumb.name}
			</button>
		{/each}
	</nav>

	<div class="h-72 overflow-y-auto rounded-fc-md border border-fc-border">
		{#if loading}
			<div class="flex h-full items-center justify-center"><Spinner /></div>
		{:else if folders.length === 0}
			<div class="flex h-full flex-col items-center justify-center gap-2 text-fc-fg-muted">
				<iconify-icon icon={nuage.folderOpen} width="28" height="28" class="block"></iconify-icon>
				<p class="text-fc-sm">No subfolders here</p>
			</div>
		{:else}
			<ul class="flex flex-col py-1">
				{#each folders as folder (folder.id)}
					{@const blocked = movingFolders.has(folder.id)}
					<li>
						<button
							type="button"
							disabled={blocked}
							onclick={() => enter(folder)}
							class="flex min-h-11 w-full items-center gap-3 px-3 text-left text-fc-sm text-fc-fg transition-colors hover:bg-fc-surface focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-fc-ring disabled:cursor-not-allowed disabled:opacity-40"
						>
							<iconify-icon icon={icons.folder} width="18" height="18" class="block shrink-0 text-fc-fg-muted"
							></iconify-icon>
							<span class="min-w-0 flex-1 truncate">{folder.name}</span>
							{#if !blocked}
								<iconify-icon icon={icons.arrow} width="14" height="14" class="block text-fc-fg-muted"
								></iconify-icon>
							{/if}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</div>

	{#snippet footer()}
		<div class="flex items-center justify-end gap-2">
			<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button icon={nuage.move} disabled={!here || unchanged} onclick={confirm}>
				{unchanged ? 'Already here' : `Move to ${here?.name ?? ''}`}
			</Button>
		</div>
	{/snippet}
</Modal>
