<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { PreferencesSection, ProfileSection, SettingsLayout, settingsRoute } from '@facile/muse';
	import type { QuotaResponse, UserProfile } from '$lib/backend';
	import { InstanceSettings } from '$lib/instance-settings.svelte';
	import { t, rememberLocale } from '$lib/i18n/index.svelte';
	import { theme } from '$lib/theme.svelte';
	import { icons } from '$lib/icons';
	import AppDataSection from '$lib/components/AppDataSection.svelte';
	import AdvancedSection from '$lib/components/settings/AdvancedSection.svelte';
	import NookSection from '$lib/components/settings/NookSection.svelte';
	import SubscriptionPanel from '$lib/components/settings/SubscriptionPanel.svelte';
	import TokensPanel from '$lib/components/settings/TokensPanel.svelte';

	const app = getContext<{ token: string; user: UserProfile | null; quota: QuotaResponse | null }>('app');

	const LEGACY: Record<string, string> = { appearance: 'preferences', api: 'tokens' };
	const instance = new InstanceSettings(app.token);

	const SECTIONS = [
		{ id: 'profile', icon: icons.userCircle },
		{ id: 'preferences', icon: icons.palette },
		{ id: 'apps', icon: icons.widget },
		{ id: 'nook', icon: icons.bolt },
		{ id: 'tokens', icon: icons.key },
		{ id: 'subscription', icon: icons.card },
		{ id: 'advanced', icon: icons.server }
	] as const;
	const sections = $derived(
		SECTIONS.map((s) => ({ ...s, label: t(`settings.tab.${s.id}`), href: `/settings/${s.id}` }))
	);

	const requested = $derived(page.params.section);

	$effect(() => {
		const legacy = requested ? LEGACY[requested] : undefined;
		if (legacy) void goto(`/settings/${legacy}`, { replaceState: true });
		else if (requested && !settingsRoute(requested, sections).known) void goto('/settings', { replaceState: true });
	});

	onMount(() => instance.load());
</script>

<svelte:head>
	<title>{t('settings.title')}</title>
</svelte:head>

<SettingsLayout
	class="mx-auto w-full max-w-5xl px-4 py-6 md:px-8"
	{sections}
	active={requested}
	backHref="/settings"
	title={t('settings.heading')}
	navLabel={t('settings.sections')}
>
	{#snippet children(id)}
		{#if id === 'profile'}
			<ProfileSection name={app.user?.name?.trim() || app.user?.email || ''} email={app.user?.email} avatar={app.user?.avatar_url || undefined} />
		{:else if id === 'preferences'}
			<PreferencesSection theme={theme.mode} onThemeChange={(next) => theme.set(next)} onLocaleChange={rememberLocale} />
		{:else if id === 'apps'}
			<AppDataSection token={app.token} />
		{:else if id === 'nook'}
			<NookSection settings={instance} />
		{:else if id === 'tokens'}
			<TokensPanel token={app.token} />
		{:else if id === 'subscription'}
			<SubscriptionPanel quota={app.quota} />
		{:else}
			<AdvancedSection settings={instance} user={app.user} />
		{/if}
	{/snippet}
</SettingsLayout>
