<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		DownloadCta,
		DownloadSection,
		Icon,
		LandingFooter,
		LandingHeader,
		PricingPlans,
		SignInWithFacile,
		icons
	} from '@facile/muse';
	import { backend } from '$lib/backend';
	import SyncDemo from '$lib/components/landing/SyncDemo.svelte';
	import { CHECKSUMS_URL, CLI_COMMANDS, RELEASES_URL } from '$lib/downloads';
	import { nuage } from '$lib/icons';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	let redirecting = $state(true);

	const signInHref = `${backend.baseUrl}/auth/oidc`;
	const heroLines = $derived(t('landing.heroTitle').split('|'));
	const features = $derived([
		{ icon: icons.refresh, title: t('landing.feature.sync.title'), body: t('landing.feature.sync.text') },
		{ icon: nuage.link, title: t('landing.feature.share.title'), body: t('landing.feature.share.text') },
		{ icon: nuage.team, title: t('landing.feature.spaces.title'), body: t('landing.feature.spaces.text') },
		{ icon: nuage.folderOpen, title: t('landing.feature.mount.title'), body: t('landing.feature.mount.text') }
	]);
	const links = $derived([
		{ href: '#features', label: t('landing.navFeatures') },
		{ href: '#download', label: t('landing.navDownload') },
		{ href: '#pricing', label: t('landing.pricing') }
	]);
	const columns = $derived([
		{
			title: 'Nuage',
			links: [
				{ href: '#features', label: t('landing.navFeatures') },
				{ href: '/downloads', label: t('landing.navDownload') },
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
				<div class="mt-8 flex flex-col items-center gap-3">
					<SignInWithFacile href={signInHref} size="lg" />
					<DownloadCta href="/#download" platforms={['macos', 'linux']} label={t('landing.downloadCta')} />
				</div>
				<div class="mt-12 w-full sm:mt-16"><SyncDemo /></div>
			</section>

			<section id="features" class="mx-auto max-w-6xl scroll-mt-20 px-4 py-16 sm:px-6 sm:py-20">
				<h2 class="text-center font-fc-title text-fc-2xl font-bold tracking-tight sm:text-fc-3xl">
					{t('landing.featuresTitle')}
				</h2>
				<div class="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
					{#each features as feature (feature.icon)}
						<div class="rounded-fc-lg bg-fc-component p-6">
							<span class="flex size-10 items-center justify-center rounded-fc-md bg-fc-surface">
								<Icon icon={feature.icon} size={20} />
							</span>
							<h3 class="mt-4 text-fc-md font-bold">{feature.title}</h3>
							<p class="mt-1.5 text-fc-sm text-fc-fg-muted">{feature.body}</p>
						</div>
					{/each}
				</div>
			</section>

			<section id="download" class="mx-auto max-w-4xl scroll-mt-20 px-4 py-16 sm:px-6 sm:py-20">
				<div class="mb-8 text-center">
					<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight sm:text-fc-3xl">{t('landing.downloadTitle')}</h2>
					<p class="mt-3 text-fc-fg-muted">{t('landing.downloadText')}</p>
				</div>
				<DownloadSection
					commands={[...CLI_COMMANDS]}
					releasesHref={RELEASES_URL}
					checksumsHref={CHECKSUMS_URL}
					headingLevel={3}
				/>
			</section>

			<div class="hidden has-[section]:block">
				<div id="pricing" class="mx-auto max-w-5xl scroll-mt-20 px-4 py-16 sm:px-6 sm:py-20">
					<PricingPlans title={t('landing.pricing')} highlight={['nuage.storage_gb']} />
				</div>
			</div>

			<section class="mx-auto max-w-5xl px-4 py-16 text-center sm:px-6 sm:py-24">
				<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight sm:text-fc-3xl">{t('landing.ctaTitle')}</h2>
				<p class="mt-3 text-fc-fg-muted">{t('landing.ctaText')}</p>
				<div class="mt-8 flex justify-center">
					<SignInWithFacile href={signInHref} size="lg" />
				</div>
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
