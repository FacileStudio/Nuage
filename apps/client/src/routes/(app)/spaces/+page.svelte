<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import { backend, type Space, type UserProfile } from '$lib/backend';

	const app = getContext<{ token: string; user: UserProfile | null }>('app');

	let spaces = $state<Space[]>([]);
	let loading = $state(true);

	onMount(async () => {
		await loadSpaces();
	});

	async function loadSpaces() {
		loading = true;
		try {
			const res = await backend.listSpaces(app.token);
			spaces = res.spaces ?? [];
		} catch {
			spaces = [];
		}
		loading = false;
	}

	function roleBadgeClass(role: string): string {
		switch (role) {
			case 'owner': return 'bg-amber-500/10 text-amber-600';
			case 'admin': return 'bg-blue-500/10 text-blue-600';
			default: return 'bg-muted text-muted-foreground';
		}
	}
</script>

<svelte:head>
	<title>Spaces — Nuage</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="flex items-center justify-between border-b border-border px-4 py-4 md:px-8 md:py-5">
		<div>
			<h1 class="text-lg font-semibold">Spaces</h1>
			<p class="mt-1 text-sm text-muted-foreground">Collaborate with your team in shared spaces.</p>
		</div>
		<a
			href="/spaces/new"
			class="inline-flex h-9 items-center gap-2 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
		>
			<iconify-icon icon="solar:add-circle-linear" width="16"></iconify-icon>
			New space
		</a>
	</div>

	<div class="flex-1 overflow-auto px-4 py-6 md:px-8">
		{#if loading}
			<div class="flex items-center justify-center py-20">
				<div class="h-6 w-6 animate-spin rounded-full border-2 border-foreground border-t-transparent"></div>
			</div>
		{:else if spaces.length === 0}
			<div class="flex flex-col items-center justify-center py-20 text-center">
				<iconify-icon icon="solar:users-group-rounded-bold-duotone" width="48" class="text-muted-foreground/40"></iconify-icon>
				<p class="mt-4 text-sm font-medium text-muted-foreground">No spaces yet</p>
				<p class="mt-1 text-sm text-muted-foreground/70">Create a space to start collaborating with your team.</p>
				<a
					href="/spaces/new"
					class="mt-4 inline-flex h-9 items-center gap-2 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
				>
					Create your first space
				</a>
			</div>
		{:else}
			<div class="grid gap-3">
				{#each spaces as space}
					<a
						href="/spaces/{space.id}"
						class="flex items-center justify-between rounded-lg border border-border p-4 transition-colors hover:bg-muted/50"
					>
						<div class="flex items-center gap-3">
							<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10">
								<iconify-icon icon="solar:users-group-rounded-bold-duotone" width="20" class="text-primary"></iconify-icon>
							</div>
							<div>
								<p class="text-sm font-medium">{space.name}</p>
								{#if space.description}
									<p class="mt-0.5 text-xs text-muted-foreground line-clamp-1">{space.description}</p>
								{/if}
							</div>
						</div>
						<span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {roleBadgeClass(space.role)}">
							{space.role}
						</span>
					</a>
				{/each}
			</div>
		{/if}
	</div>
</div>
