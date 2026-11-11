<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { LandingFooter, LandingHeader, PricingPlans, SignInWithFacile } from '@facile/muse';
	import { backend } from '$lib/backend';
	import { nuage, icons } from '$lib/icons';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	let redirecting = $state(true);

	const signInHref = `${backend.baseUrl}/auth/oidc`;
	const heroLines = $derived(t('landing.heroTitle').split('|'));
	const features = [
		{ icon: icons.upload, id: 'upload' },
		{ icon: icons.folder, id: 'folders' },
		{ icon: nuage.lock, id: 'hosted' }
	] as const;

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
	<div class="min-h-screen bg-fc-page text-fc-fg">
		<LandingHeader appName="Nuage" icon={nuage.brand} skipHref="#main">
			{#snippet actions()}
				<SignInWithFacile href={signInHref} />
			{/snippet}
		</LandingHeader>

		<main id="main">
			<section class="mx-auto max-w-5xl px-6 py-24 text-center">
				<h1 class="font-heading text-5xl font-semibold tracking-tight">
					{heroLines[0]}<br />{heroLines[1]}
				</h1>
				<p class="mx-auto mt-6 max-w-xl text-lg text-fc-fg-muted">{t('landing.heroText')}</p>
				<div class="mt-10 flex justify-center">
					<SignInWithFacile href={signInHref} size="lg" />
				</div>
			</section>

			<section class="mx-auto max-w-5xl px-6 py-20">
				<div class="grid gap-6 md:grid-cols-3">
					{#each features as feature (feature.id)}
						<div class="rounded-fc-lg bg-fc-component p-6">
							<div class="mb-3 flex size-10 items-center justify-center rounded-fc-md bg-fc-surface">
								<iconify-icon icon={feature.icon} width="20" class="block"></iconify-icon>
							</div>
							<h3 class="text-base font-semibold">{t(`landing.feature.${feature.id}.title`)}</h3>
							<p class="mt-1.5 text-sm text-fc-fg-muted">{t(`landing.feature.${feature.id}.text`)}</p>
						</div>
					{/each}
				</div>
			</section>

			<div class="hidden has-[section]:block">
				<div id="pricing" class="mx-auto max-w-5xl px-6 py-20">
					<PricingPlans title={t('landing.pricing')} highlight={['nuage.storage_gb']} />
				</div>
			</div>

			<section class="mx-auto max-w-5xl px-6 py-20 text-center">
				<h2 class="font-heading text-3xl font-semibold tracking-tight">{t('landing.ctaTitle')}</h2>
				<p class="mt-4 text-fc-fg-muted">{t('landing.ctaText')}</p>
				<div class="mt-8 flex justify-center">
					<SignInWithFacile href={signInHref} size="lg" />
				</div>
			</section>
		</main>

		<LandingFooter
			appName="Nuage"
			icon={nuage.brand}
			tagline={t('login.tagline')}
			legal={{ onLocaleChange: rememberLocale }}
		/>
	</div>
{/if}
