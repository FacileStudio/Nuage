<script lang="ts">
	import { getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend, type UserProfile } from '$lib/backend';

	const app = getContext<{ token: string; user: UserProfile | null }>('app');

	let name = $state('');
	let description = $state('');
	let saving = $state(false);
	let error = $state('');

	async function handleSubmit() {
		if (!name.trim()) {
			error = 'Name is required';
			return;
		}

		saving = true;
		error = '';
		try {
			const space = await backend.createSpace(app.token, {
				name: name.trim(),
				description: description.trim()
			});
			goto(`/spaces/${space.id}`);
		} catch (e: any) {
			error = e.message || 'Failed to create space';
		}
		saving = false;
	}
</script>

<svelte:head>
	<title>New Space — Nuage</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<div class="flex items-center gap-3">
			<a href="/spaces" class="text-muted-foreground transition-colors hover:text-foreground" aria-label="Back to spaces">
				<iconify-icon icon="solar:arrow-left-linear" width="20"></iconify-icon>
			</a>
			<h1 class="text-lg font-semibold">New Space</h1>
		</div>
	</div>

	<div class="flex-1 overflow-auto px-4 py-6 md:px-8">
		<div class="max-w-xl space-y-6">
			<div class="space-y-4">
				<div>
					<label for="space-name" class="mb-1.5 block text-sm font-medium">Name</label>
					<input
						id="space-name"
						type="text"
						bind:value={name}
						placeholder="e.g. Design Team"
						class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
					/>
				</div>
				<div>
					<label for="space-description" class="mb-1.5 block text-sm font-medium">Description</label>
					<textarea
						id="space-description"
						bind:value={description}
						placeholder="What is this space for?"
						rows="3"
						class="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
					></textarea>
				</div>
			</div>

			{#if error}
				<p class="text-sm text-destructive">{error}</p>
			{/if}

			<div class="flex items-center gap-3">
				<button
					class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
					onclick={handleSubmit}
					disabled={saving}
				>
					{saving ? 'Creating...' : 'Create space'}
				</button>
				<a
					href="/spaces"
					class="inline-flex h-9 items-center rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
				>
					Cancel
				</a>
			</div>
		</div>
	</div>
</div>
