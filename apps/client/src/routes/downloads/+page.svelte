<script lang="ts">
	import { DownloadSection, LandingFooter, LandingHeader } from '@facile/muse';
	import { CHECKSUMS_URL, CLI_COMMANDS, RELEASES_URL } from '$lib/downloads';
	import { nuage } from '$lib/icons';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	const links = $derived([
		{ href: '/#features', label: t('landing.navFeatures') },
		{ href: '/downloads', label: t('landing.navDownload'), active: true },
		{ href: '/#pricing', label: t('landing.pricing') }
	]);
	const columns = $derived([
		{ title: 'Nuage', links },
		{ title: 'Facile', links: [{ href: 'https://facile.studio', label: 'facile.studio', external: true }] }
	]);
</script>

<svelte:head>
	<title>{t('downloads.title')}</title>
	<meta name="description" content={t('downloads.description')} />
</svelte:head>

<div class="min-h-dvh bg-fc-page text-fc-fg">
	<LandingHeader appName="Nuage" icon={nuage.brand} {links} skipHref="#main" />

	<main id="main" class="mx-auto flex max-w-4xl flex-col gap-10 px-4 py-12 sm:px-6 sm:py-16">
		<section class="flex flex-col items-center gap-4 text-center">
			<h1 class="font-fc-title text-fc-3xl font-bold tracking-tight sm:text-5xl">{t('downloads.heading')}</h1>
			<p class="max-w-md text-fc-md text-fc-fg-muted">{t('landing.downloadText')}</p>
		</section>

		<DownloadSection commands={[...CLI_COMMANDS]} releasesHref={RELEASES_URL} checksumsHref={CHECKSUMS_URL} />
	</main>

	<LandingFooter
		appName="Nuage"
		icon={nuage.brand}
		tagline={t('login.tagline')}
		{columns}
		legal={{ onLocaleChange: rememberLocale }}
	/>
</div>
