<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Alert, AuthScreen } from '@facile/muse';
	import { backend } from '$lib/backend';
	import { nuage } from '$lib/icons';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	let failed = $state(false);

	/**
	 * Already signed in? Ask the API, because the credential is the cookie an SSO callback left
	 * behind. This page is also where porte's failed-login redirect lands, so the error is read
	 * before the probe decides to leave.
	 */
	onMount(async () => {
		failed = page.url.searchParams.has('error');
		try {
			await backend.me();
			goto('/files');
		} catch {
			return;
		}
	});
</script>

<svelte:head>
	<title>{t('login.title')}</title>
</svelte:head>

<AuthScreen appName="Nuage" href="{backend.baseUrl}/auth/oidc" onLocaleChange={rememberLocale}>
	{#snippet logo()}
		<iconify-icon icon={nuage.brand} width="40" height="40" class="block text-fc-fg"></iconify-icon>
	{/snippet}
	{#snippet tagline()}
		{t('login.tagline')}
	{/snippet}
	{#snippet notice()}
		{#if failed}
			<Alert tone="danger">{t('login.failed')}</Alert>
		{/if}
	{/snippet}
</AuthScreen>
