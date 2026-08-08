<script lang="ts">
	import { getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button, Card, Field, Input, Textarea, toast } from '@facile/muse';
	import { backend } from '$lib/backend';
	import { icons } from '$lib/icons';

	const app = getContext<{ token: string }>('app');

	let name = $state('');
	let description = $state('');
	let saving = $state(false);
	let error = $state('');

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!name.trim()) {
			error = 'Give the space a name.';
			return;
		}

		saving = true;
		error = '';
		try {
			const space = await backend.createSpace(app.token, {
				name: name.trim(),
				description: description.trim()
			});
			toast.success(`“${space.name}” is ready.`);
			goto(`/spaces/${space.id}`);
		} catch (e) {
			error = e instanceof Error && e.message ? e.message : 'Could not create that space.';
		}
		saving = false;
	}
</script>

<svelte:head>
	<title>New space — Nuage</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-xl flex-col gap-6 px-4 py-6 md:px-8">
	<div class="flex flex-col gap-4">
		<Button variant="ghost" size="sm" href="/spaces" icon={icons.chevronLeft} class="-ml-3 self-start">
			Spaces
		</Button>
		<div class="flex flex-col gap-1">
			<h1 class="text-fc-2xl font-semibold text-fc-fg">New space</h1>
			<p class="text-fc-sm text-fc-fg-muted">
				You start as its owner. Invite people once it exists.
			</p>
		</div>
	</div>

	<Card>
		<form class="flex flex-col gap-4" onsubmit={handleSubmit}>
			<Field label="Name" error={error || undefined}>
				<Input bind:value={name} placeholder="Design team" autocomplete="off" />
			</Field>
			<Field label="Description" helper="Optional — what belongs in here.">
				<Textarea bind:value={description} placeholder="Brand assets, mockups, exports." rows={3} />
			</Field>
			<div class="flex flex-wrap gap-2">
				<Button type="submit" icon={icons.plus} disabled={saving}>
					{saving ? 'Creating…' : 'Create space'}
				</Button>
				<Button variant="ghost" href="/spaces">Cancel</Button>
			</div>
		</form>
	</Card>
</div>
