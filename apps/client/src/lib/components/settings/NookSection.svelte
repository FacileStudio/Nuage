<script lang="ts">
	import { Button, Field, Input, SecretField, SettingsRow, SettingsSection, StatusDot, Switch } from '@facile/muse';
	import type { InstanceSettings } from '$lib/instance-settings.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { icons } from '$lib/icons';

	let { settings }: { settings: InstanceSettings } = $props();

	const status = $derived.by(() => {
		const result = settings.nookTestResult;
		if (!settings.nookEnabled) return { tone: 'neutral' as const, label: t('settings.nook.disabled'), pulse: false };
		if (settings.testing) return { tone: 'info' as const, label: t('settings.nook.testing'), pulse: true };
		if (!settings.nookWebhookUrl) return { tone: 'warning' as const, label: t('settings.nook.noUrl'), pulse: false };
		if (result?.success) return { tone: 'success' as const, label: t('settings.nook.reached'), pulse: false };
		if (result) return { tone: 'danger' as const, label: result.message ?? t('settings.nook.noAnswer'), pulse: false };
		return { tone: 'neutral' as const, label: t('settings.nook.untested'), pulse: false };
	});
</script>

<SettingsSection title="Nook" description={t('settings.nook.description')}>
	<SettingsRow label={t('settings.nook.status')}>
		<StatusDot tone={status.tone} label={status.label} pulse={status.pulse} />
	</SettingsRow>
	<SettingsRow label={t('settings.nook.enabled')} description={t('settings.nook.enabledText')}>
		<Switch bind:checked={settings.nookEnabled} aria-label={t('settings.nook.switchLabel')} />
	</SettingsRow>
	<SettingsRow label={t('settings.nook.url')} stacked>
		<Field>
			<Input bind:value={settings.nookWebhookUrl} type="url" placeholder="https://nook.example.com/hooks/nuage" />
		</Field>
	</SettingsRow>
	<SettingsRow label={t('settings.nook.secret')} description={t('settings.nook.secretText')} stacked>
		<SecretField bind:value={settings.nookSecret} editable placeholder={t('settings.nook.notSet')} />
	</SettingsRow>
	<SettingsRow>
		<div class="flex flex-wrap gap-2">
			<Button icon={icons.check} disabled={settings.saving} onclick={() => settings.save()}>
				{settings.saving ? t('common.saving') : t('common.save')}
			</Button>
			<Button
				variant="secondary"
				icon={icons.bolt}
				disabled={settings.testing || !settings.nookWebhookUrl}
				onclick={() => settings.testNook()}
			>
				{t('settings.nook.test')}
			</Button>
		</div>
	</SettingsRow>
</SettingsSection>
