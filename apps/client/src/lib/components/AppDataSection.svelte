<script lang="ts">
	import { Button, ConfirmModal, EmptyState, SettingsSection, Table, toast } from '@facile/muse';
	import { onMount } from 'svelte';
	import { backend, type AppData } from '$lib/backend';
	import { formatSize } from '$lib/format';
	import { icons } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

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
			toast.success(t('appdata.deleted', { name: target.name }));
		} catch (e) {
			toast.danger(e instanceof Error && e.message ? e.message : t('appdata.deleteFailed'));
		}
		deletingApp = null;
		await loadAppData();
	}
</script>

<SettingsSection
	title={t('appdata.title')}
	description={t('appdata.description')}
>
	{#if appData.length === 0}
		<EmptyState
			bare
			icon={icons.widget}
			title={t('appdata.emptyTitle')}
			description={t('appdata.emptyText')}
		/>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">{t('appdata.app')}</th>
					<th scope="col">{t('appdata.items')}</th>
					<th scope="col">{t('appdata.used')}</th>
					<th scope="col" class="text-right"><span class="sr-only">{t('settings.tokens.actions')}</span></th>
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
								{t('common.delete')}
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
	title={t('appdata.confirmTitle')}
	description={deletingApp
		? t('appdata.confirmText', { name: deletingApp.name, size: formatSize(deletingApp.used_bytes) })
		: ''}
	confirmLabel={t('common.delete')}
	tone="danger"
	onConfirm={deleteAppData}
	onCancel={() => (deletingApp = null)}
/>
