<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		Button,
		ConfirmModal,
		EmptyState,
		Field,
		Input,
		SettingsRow,
		SettingsSection,
		SecretField,
		Spinner,
		Textarea,
		toast
	} from '@facile/muse';
	import { backend, type Space } from '$lib/backend';
	import { getSpaceStore } from '$lib/space.svelte';
	import { icons } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';

	const app = getContext<{ token: string }>('app');
	const spaceStore = getSpaceStore();

	let space = $state<Space | null>(null);
	let loading = $state(true);
	let name = $state('');
	let description = $state('');
	let saving = $state(false);
	let deleteOpen = $state(false);

	let spaceId = $derived(Number(page.params.id));

	onMount(loadSpace);

	async function loadSpace() {
		loading = true;
		try {
			space = await backend.getSpace(app.token, spaceId);
			name = space.name;
			description = space.description ?? '';
		} catch {
			space = null;
		}
		loading = false;
	}

	async function saveSettings() {
		saving = true;
		try {
			space = await backend.updateSpace(app.token, spaceId, {
				name: name.trim(),
				description: description.trim()
			});
			if (spaceStore.current?.id === spaceId) spaceStore.set(space);
			toast.success(t('spaceSettings.updated'));
		} catch (e) {
			toast.danger(e instanceof Error && e.message ? e.message : t('spaceSettings.saveFailed'));
		}
		saving = false;
	}

	async function confirmDelete() {
		try {
			await backend.deleteSpace(app.token, spaceId);
		} catch (e) {
			toast.danger(e instanceof Error && e.message ? e.message : t('spaceSettings.deleteFailed'));
			throw e;
		}
		if (spaceStore.current?.id === spaceId) spaceStore.clear();
		toast.success(t('spaceSettings.deleted'));
		goto('/spaces');
	}

	const isOwner = $derived(space?.role === 'owner');
	const linked = $derived(Boolean(space?.registre_space_id));
</script>

<svelte:head>
	<title>{t('spaceSettings.title', { name: space?.name ?? t('space.fallbackName') })}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-10 px-4 py-6 md:px-8">
	<div class="flex flex-col gap-4">
		<Button
			variant="ghost"
			size="sm"
			href="/spaces/{spaceId}"
			icon={icons.chevronLeft}
			class="-ml-3 self-start"
		>
			{space?.name ?? t('space.fallbackName')}
		</Button>
		<PageHeader title={t('spaceSettings.heading')} />
	</div>

	{#if loading}
		<div class="flex h-64 items-center justify-center"><Spinner /></div>
	{:else if !space}
		<EmptyState
			icon={icons.space}
			title={t('space.notFound')}
			description={t('space.notFoundText')}
		>
			<Button href="/spaces" icon={icons.chevronLeft}>{t('space.backToSpaces')}</Button>
		</EmptyState>
	{:else}
		<SettingsSection title={t('spaceSettings.general')} description={t('spaceSettings.generalText')}>
			<SettingsRow label={t('common.name')} stacked>
				<Field helper={linked ? t('spaceSettings.renameInRegistre') : undefined}>
					<Input bind:value={name} placeholder={t('spaceSettings.namePlaceholder')} disabled={linked} />
				</Field>
			</SettingsRow>
			<SettingsRow label={t('common.description')} stacked>
				<Field>
					<Textarea bind:value={description} rows={3} placeholder={t('spaceSettings.descriptionPlaceholder')} />
				</Field>
			</SettingsRow>
			<SettingsRow>
				<Button icon={icons.check} disabled={saving} onclick={saveSettings}>
					{saving ? t('common.saving') : t('spaceSettings.save')}
				</Button>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection
			title="WebDAV"
			description={t('spaceSettings.webdavText')}
		>
			<SettingsRow label={t('spaceSettings.mountUrl')} description={t('settings.webdav.personalText')} stacked>
				<SecretField value={`${page.url.origin}/webdav/spaces/${spaceId}`} sensitive={false} />
			</SettingsRow>
		</SettingsSection>

		{#if isOwner && !linked}
			<!-- Danger zone sits last and never gets a heading of its own above the fold: a
			     destructive action should cost a scroll (CHARTE §14). -->
			<SettingsSection
				title={t('spaceSettings.danger')}
				description={t('spaceSettings.dangerText')}
			>
				<SettingsRow label={t('spaceSettings.deleteThis')} description={t('spaceSettings.deleteThisText')}>
					<Button variant="danger" icon={icons.remove} onclick={() => (deleteOpen = true)}>
						{t('spaceSettings.delete')}
					</Button>
				</SettingsRow>
			</SettingsSection>
		{/if}
	{/if}
</div>

<ConfirmModal
	bind:open={deleteOpen}
	title={t('spaceSettings.deleteTitle', { name: space?.name ?? t('space.leaveFallback') })}
	description={t('spaceSettings.deleteText')}
	confirmLabel={t('spaceSettings.delete')}
	tone="danger"
	onConfirm={confirmDelete}
/>
