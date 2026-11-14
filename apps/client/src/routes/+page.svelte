<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		FeatureSplit,
		ForDevelopers,
		LANDING_HEADING,
		LandingCta,
		LandingFooter,
		LandingHeader,
		PricingPlans,
		SignInWithFacile,
		WorksWith,
		icons,
		type DeveloperRow,
		type FeatureItem,
		type WorksWithApp
	} from '@facile/muse';
	import { backend } from '$lib/backend';
	import AlbumDemo from '$lib/components/landing/AlbumDemo.svelte';
	import PhoneTile from '$lib/components/landing/PhoneTile.svelte';
	import SpacesTile from '$lib/components/landing/SpacesTile.svelte';
	import AppsTile from '$lib/components/landing/AppsTile.svelte';
	import { nuage, suite } from '$lib/icons';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	let redirecting = $state(true);

	const signInHref = `${backend.baseUrl}/auth/oidc`;
	const heroLines = $derived(t('landing.heroTitle').split('|'));
	const features = $derived<FeatureItem[]>([
		{ icon: nuage.share, title: t('landing.feature.share.title'), text: t('landing.feature.share.text') },
		{ icon: nuage.team, title: t('landing.feature.spaces.title'), text: t('landing.feature.spaces.text') },
		{ icon: suite.eye, title: t('landing.feature.view.title'), text: t('landing.feature.view.text') },
		{ icon: nuage.restore, title: t('landing.feature.trash.title'), text: t('landing.feature.trash.text') }
	]);
	const apps = $derived<WorksWithApp[]>([
		{ name: 'Plume', icon: suite.plume, line: t('landing.with.plume'), href: 'https://plume.facile.studio' },
		{ name: 'Ardoise', icon: suite.ardoise, line: t('landing.with.ardoise'), href: 'https://ardoise.facile.studio' },
		{ name: 'Toile', icon: suite.toile, line: t('landing.with.toile'), href: 'https://toile.facile.studio' },
		{ name: 'Bulle', icon: suite.bulle, line: t('landing.with.bulle'), href: 'https://bulle.facile.studio' }
	]);
	const developers = $derived<DeveloperRow[]>([
		{ kind: 'cli', text: t('landing.dev.cli'), command: 'brew install facilestudio/tap/nuage', href: '/downloads' },
		{ icon: suite.webdav, title: 'WebDAV', text: t('landing.dev.webdav') },
		{ kind: 'api', text: t('landing.dev.api'), href: 'https://nuage.facile.studio/docs' },
		{ kind: 'mcp', text: t('landing.dev.mcp') },
		{ kind: 'source', text: t('landing.dev.source'), href: 'https://github.com/FacileStudio/Nuage' }
	]);
	const links = $derived([
		{ href: '#features', label: t('landing.navFeatures') },
		{ href: '#pricing', label: t('landing.pricing') }
	]);
	const columns = $derived([
		{
			title: 'Nuage',
			links: [
				{ href: '#features', label: t('landing.navFeatures') },
				{ href: '#pricing', label: t('landing.pricing') }
			]
		},
		{ title: 'Facile', links: [{ href: 'https://facile.studio', label: 'facile.studio', external: true }] }
	]);

	/**
	 * This is where the SSO callback lands: OIDC_SUCCESS_URL is the app's origin, and porte hands
	 * back a session cookie and a 302, nothing in the URL. So the only way to tell an arriving
	 * SSO user from a stranger is to ask the API.
	 */
	onMount(async () => {
		try {
			await backend.me();
			goto('/files');
			return;
		} catch {
			redirecting = false;
		}
	});
</script>

<svelte:head>
	<title>{t('landing.metaTitle')}</title>
	<meta name="description" content={t('landing.metaDescription')} />
</svelte:head>

{#if !redirecting}
	<div class="min-h-dvh bg-fc-page text-fc-fg">
		<LandingHeader appName="Nuage" icon={nuage.brand} {links} skipHref="#main">
			{#snippet actions()}
				<SignInWithFacile href={signInHref} />
			{/snippet}
		</LandingHeader>

		<main id="main">
			<section class="mx-auto flex max-w-6xl flex-col items-center px-4 pt-16 pb-12 text-center sm:px-6 sm:pt-24">
				<h1 class="font-fc-title text-fc-3xl leading-tight font-bold tracking-tight sm:text-5xl">
					{heroLines[0]}<br />{heroLines[1]}
				</h1>
				<p class="mt-5 max-w-xl text-fc-lg text-fc-fg-muted">{t('landing.heroText')}</p>
				<LandingCta {signInHref} align="center" class="mt-8 w-full sm:w-auto" />
				<div class="mt-12 w-full sm:mt-16"><AlbumDemo /></div>
			</section>

			<section class="relative mx-auto max-w-5xl px-4 py-16 sm:px-6 sm:py-20">
				<h2 class="mb-10 text-center {LANDING_HEADING}">{t('landing.cardsTitle')}</h2>
				<div class="grid gap-6 md:grid-cols-3">
					<SpacesTile />
					<PhoneTile />
					<AppsTile />
				</div>
			</section>

			<FeatureSplit
				id="features"
				class="mx-auto w-full max-w-5xl scroll-mt-20 px-4 py-16 sm:px-6 sm:py-20"
				title={t('landing.featuresTitle')}
				description={t('landing.featuresLine')}
				{features}
			/>

			<div class="hidden has-[section]:block">
				<div id="pricing" class="mx-auto max-w-5xl scroll-mt-20 px-4 py-16 sm:px-6 sm:py-20">
					<PricingPlans
						title={t('landing.pricing')}
						headingClass={LANDING_HEADING}
						highlight={['nuage.storage_gb', 'ai.access']}
						highlightOnly
						recommended="pro"
						suiteHref="https://suite.facile.studio/#tarifs"
					/>
				</div>
			</div>

			<WorksWith class="mx-auto w-full max-w-5xl px-4 py-16 sm:px-6 sm:py-20" {apps} />

			<div id="developers" class="mx-auto w-full max-w-5xl scroll-mt-20 px-4 py-16 sm:px-6 sm:py-20">
				<ForDevelopers rows={developers} description={t('landing.dev.text')} />
			</div>
			<section class="mx-auto max-w-5xl px-4 py-16 text-center sm:px-6 sm:py-24">
				<h2 class={LANDING_HEADING}>{t('landing.ctaTitle')}</h2>
				<p class="mt-3 text-fc-fg-muted">{t('landing.ctaText')}</p>
				<LandingCta {signInHref} align="center" class="mt-8" />
			</section>

		</main>

		<LandingFooter
			appName="Nuage"
			icon={nuage.brand}
			tagline={t('login.tagline')}
			{columns}
			legal={{ onLocaleChange: rememberLocale }}
		/>
	</div>
{/if}
