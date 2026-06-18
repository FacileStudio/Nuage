<script lang="ts">
	import { onMount } from 'svelte';
	import { backend, type Space } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';

	let { token }: { token: string } = $props();

	const spaceStore = getSpaceStore();

	let spaces = $state<Space[]>([]);
	let open = $state(false);
	let loading = $state(true);

	onMount(async () => {
		await loadSpaces();
	});

	async function loadSpaces() {
		loading = true;
		try {
			const res = await backend.listSpaces(token);
			spaces = res.spaces ?? [];

			const savedId = spaceStore.getSavedId();
			if (savedId) {
				const match = spaces.find(s => s.id === savedId);
				if (match) {
					spaceStore.set(match);
				} else {
					spaceStore.clear();
				}
			}
		} catch {
			spaces = [];
		}
		loading = false;
	}

	function selectSpace(space: Space | null) {
		spaceStore.set(space);
		open = false;
	}

	function handleClickOutside(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.space-switcher')) {
			open = false;
		}
	}

	$effect(() => {
		if (open) {
			document.addEventListener('click', handleClickOutside);
			return () => document.removeEventListener('click', handleClickOutside);
		}
	});
</script>

{#if !loading && spaces.length > 0}
	<div class="space-switcher relative px-3 pb-2">
		<button
			class="flex w-full items-center gap-2.5 rounded-lg border border-border/60 bg-muted/30 px-3 py-2 text-left text-sm transition-colors hover:bg-muted/60"
			onclick={() => open = !open}
		>
			<iconify-icon
				icon={spaceStore.current ? 'solar:users-group-rounded-bold-duotone' : 'solar:user-circle-bold-duotone'}
				width="18"
				class="shrink-0 text-muted-foreground"
			></iconify-icon>
			<span class="min-w-0 flex-1 truncate font-medium">
				{spaceStore.current?.name ?? 'Personal'}
			</span>
			<iconify-icon
				icon="solar:alt-arrow-down-linear"
				width="14"
				class="shrink-0 text-muted-foreground transition-transform {open ? 'rotate-180' : ''}"
			></iconify-icon>
		</button>

		{#if open}
			<div class="absolute left-3 right-3 z-50 mt-1 overflow-hidden rounded-lg border border-border bg-background shadow-lg">
				<div class="max-h-64 overflow-auto p-1">
					<button
						class="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm transition-colors {!spaceStore.current ? 'bg-foreground text-background' : 'text-foreground hover:bg-muted'}"
						onclick={() => selectSpace(null)}
					>
						<iconify-icon icon="solar:user-circle-bold-duotone" width="16" class="shrink-0"></iconify-icon>
						Personal
					</button>

					{#each spaces as space}
						<button
							class="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm transition-colors {spaceStore.current?.id === space.id ? 'bg-foreground text-background' : 'text-foreground hover:bg-muted'}"
							onclick={() => selectSpace(space)}
						>
							<iconify-icon icon="solar:users-group-rounded-bold-duotone" width="16" class="shrink-0"></iconify-icon>
							<span class="min-w-0 flex-1 truncate">{space.name}</span>
							<span class="shrink-0 text-xs opacity-60">{space.role}</span>
						</button>
					{/each}
				</div>

				<div class="border-t border-border p-1">
					<a
						href="/spaces"
						class="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
						onclick={() => open = false}
					>
						<iconify-icon icon="solar:settings-linear" width="16" class="shrink-0"></iconify-icon>
						Manage spaces
					</a>
				</div>
			</div>
		{/if}
	</div>
{/if}
