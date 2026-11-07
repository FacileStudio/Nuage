<script lang="ts">
	import { Button, Field, Input, SettingsRow, SettingsSection } from '@facile/muse';
	import type { UserProfile } from '$lib/backend';
	import { formatDate } from '$lib/format';
	import type { InstanceSettings } from '$lib/instance-settings.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { icons } from '$lib/icons';

	let { settings, user }: { settings: InstanceSettings; user: UserProfile | null } = $props();
</script>

<SettingsSection title={t('settings.instance.title')} description={t('settings.instance.description')}>
	<SettingsRow label={t('settings.instance.name')} stacked>
		<Field>
			<Input bind:value={settings.instanceName} placeholder={t('settings.instancePlaceholder')} />
		</Field>
	</SettingsRow>
	{#if settings.isAdmin}
		<SettingsRow label={t('settings.retention')} description={t('settings.retention.text')} stacked>
			<Field>
				<Input
					bind:value={settings.trashRetention}
					type="number"
					min="0"
					max="3650"
					step="1"
					inputmode="numeric"
					aria-label={t('settings.retention.label')}
				/>
			</Field>
		</SettingsRow>
	{/if}
	<SettingsRow>
		<Button icon={icons.check} disabled={settings.saving} onclick={() => settings.save()}>
			{settings.saving ? t('common.saving') : t('common.save')}
		</Button>
	</SettingsRow>
</SettingsSection>

<SettingsSection title={t('settings.about.title')} description={t('settings.about.description')}>
	<SettingsRow label={t('settings.about.docs')} description={t('settings.about.docsText')}>
		<Button variant="outline" href="/docs" data-sveltekit-reload iconRight={icons.arrow}>{t('settings.about.open')}</Button>
	</SettingsRow>
	<SettingsRow label={t('settings.about.created')}>
		<span class="text-fc-sm text-fc-fg-muted">{user?.created_at ? formatDate(user.created_at) : '—'}</span>
	</SettingsRow>
</SettingsSection>
