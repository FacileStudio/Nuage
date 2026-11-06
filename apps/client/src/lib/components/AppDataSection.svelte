<script lang="ts">
	import { Button, ConfirmModal, EmptyState, SettingsSection, Table, toast } from '@facile/muse';
	import { onMount } from 'svelte';
	import { backend, type AppData } from '$lib/backend';
	import { formatSize } from '$lib/format';
	import { icons } from '$lib/icons';

	let { token }: { token: string } = $props();

	let appData = $state<AppData[]>([]);
	let deletingApp = $state<AppData | null>(null);

	onMount(loadAppData);

	async function loadAppData() {
		try {
			const res = await backend.listAppData(token);
			appData = res.apps ?? [];
		} catch {
			appData = [];
		}
	}

	async function deleteAppData() {
		const target = deletingApp;
		if (!target) return;
		try {
			await backend.deleteAppData(token, target.app);
			toast.success(`Deleted what ${target.name} stored.`);
		} catch (e) {
			toast.danger(e instanceof Error && e.message ? e.message : 'Could not delete that app data.');
		}
		deletingApp = null;
		await loadAppData();
	}
</script>

<SettingsSection
	title="App data"
	description="Private storage other Facile apps keep for you. It counts toward your storage plan and is never shown as a folder or a space."
>
	{#if appData.length === 0}
		<EmptyState
			bare
			icon={icons.widget}
			title="No app data"
			description="When an app such as Plume or Ardoise stores something for you, it shows here."
		/>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">App</th>
					<th scope="col">Items</th>
					<th scope="col">Used</th>
					<th scope="col" class="text-right"><span class="sr-only">Actions</span></th>
				</tr>
			</thead>
			<tbody>
				{#each appData as entry (entry.app)}
					<tr>
						<td class="font-medium">{entry.name}</td>
						<td class="text-fc-fg-muted">{entry.objects}</td>
						<td class="text-fc-fg-muted">{formatSize(entry.used_bytes)}</td>
						<td class="text-right">
							<Button
								variant="ghost-danger"
								size="sm"
								icon={icons.remove}
								onclick={() => (deletingApp = entry)}
							>
								Delete
							</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</SettingsSection>

<ConfirmModal
	open={deletingApp !== null}
	title="Delete this app's data?"
	description={deletingApp
		? `Everything ${deletingApp.name} stored for you (${formatSize(deletingApp.used_bytes)}) is deleted for good. The app may lose documents it relies on, and this cannot be undone.`
		: ''}
	confirmLabel="Delete"
	tone="danger"
	onConfirm={deleteAppData}
	onCancel={() => (deletingApp = null)}
/>
