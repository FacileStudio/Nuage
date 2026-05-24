<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { backend } from '$lib/backend';

	const TOKEN_KEY = 'nuage.token';

	let redirecting = $state(true);
	let ssoOnly = $state(false);

	onMount(async () => {
		const token = page.url.searchParams.get('token');
		if (token) {
			localStorage.setItem(TOKEN_KEY, token);
			goto('/drive');
			return;
		}
		if (localStorage.getItem(TOKEN_KEY)) {
			goto('/drive');
			return;
		}

		try {
			const cfg = await backend.getAuthConfig();
			ssoOnly = cfg.sso_only ?? false;
		} catch {}

		redirecting = false;
	});
</script>

<svelte:head>
	<title>Nuage — Cloud Storage</title>
	<meta name="description" content="Self-hosted file storage. Upload, organize, and share — your cloud, your rules." />
</svelte:head>

{#if !redirecting}
<div class="min-h-screen bg-background text-foreground">
	<header class="border-b border-border">
		<div class="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
			<div class="flex h-14 items-center gap-3">
				<iconify-icon icon="solar:cloud-bold-duotone" width="28" class="text-foreground"></iconify-icon>
				<span class="text-2xl font-bold font-heading tracking-tight">Nuage</span>
			</div>
			<div class="flex items-center gap-2">
				<a href="/login" class="inline-flex h-9 items-center justify-center rounded-md px-4 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground">
					Log in
				</a>
				<a
					href={ssoOnly ? '/login' : '/login?tab=register'}
					class="inline-flex h-9 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
				>
					{ssoOnly ? 'Continue with SSO' : 'Get started'}
				</a>
			</div>
		</div>
	</header>

	<main>
		<section class="mx-auto max-w-5xl px-6 py-24 text-center">
			<h1 class="text-5xl font-bold tracking-tight">
				Your files.<br />Your cloud.
			</h1>
			<p class="mx-auto mt-6 max-w-xl text-lg text-muted-foreground">
				Nuage is a self-hosted file storage platform.
				Upload, organize into folders, share with a link — done.
			</p>
			<div class="mt-10 flex justify-center gap-3">
				<a
					href={ssoOnly ? '/login' : '/login?tab=register'}
					class="inline-flex h-11 items-center justify-center rounded-md bg-primary px-6 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
				>
					{ssoOnly ? 'Continue with SSO' : 'Start storing'}
					<svg class="ml-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" /></svg>
				</a>
				<a
					href="/login"
					class="inline-flex h-11 items-center justify-center rounded-md border border-border bg-background px-6 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
				>
					Log in
				</a>
			</div>
		</section>

		<div class="mx-auto max-w-5xl"><div class="h-px bg-border"></div></div>

		<section class="mx-auto max-w-5xl px-6 py-20">
			<div class="grid gap-6 md:grid-cols-3">
				<div class="rounded-lg border border-border p-6">
					<div class="mb-3 flex h-10 w-10 items-center justify-center rounded-md border border-border">
						<iconify-icon icon="solar:upload-bold-duotone" width="20"></iconify-icon>
					</div>
					<h3 class="text-base font-semibold">Upload anything</h3>
					<p class="mt-1.5 text-sm text-muted-foreground">
						Drag and drop files or click to upload. Any file type, any size.
					</p>
				</div>

				<div class="rounded-lg border border-border p-6">
					<div class="mb-3 flex h-10 w-10 items-center justify-center rounded-md border border-border">
						<iconify-icon icon="solar:folder-bold-duotone" width="20"></iconify-icon>
					</div>
					<h3 class="text-base font-semibold">Organize with folders</h3>
					<p class="mt-1.5 text-sm text-muted-foreground">
						Create nested folders, move files around, keep everything tidy.
					</p>
				</div>

				<div class="rounded-lg border border-border p-6">
					<div class="mb-3 flex h-10 w-10 items-center justify-center rounded-md border border-border">
						<iconify-icon icon="solar:lock-bold-duotone" width="20"></iconify-icon>
					</div>
					<h3 class="text-base font-semibold">Self-hosted</h3>
					<p class="mt-1.5 text-sm text-muted-foreground">
						Your files stay on your server. No third-party cloud, no data harvesting.
					</p>
				</div>
			</div>
		</section>

		<div class="mx-auto max-w-5xl"><div class="h-px bg-border"></div></div>

		<section class="mx-auto max-w-5xl px-6 py-20 text-center">
			<h2 class="text-3xl font-bold tracking-tight">
				{ssoOnly ? 'Ready to sign in?' : 'Ready to start?'}
			</h2>
			<p class="mt-4 text-muted-foreground">
				{ssoOnly ? 'Use your organization SSO to access Nuage.' : 'Free to use. Self-hosted. No credit card required.'}
			</p>
			<a
				href={ssoOnly ? '/login' : '/login?tab=register'}
				class="mt-8 inline-flex h-11 items-center justify-center rounded-md bg-primary px-6 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
			>
				{ssoOnly ? 'Continue with SSO' : 'Create an account'}
			</a>
		</section>
	</main>

	<footer class="border-t border-border text-center text-muted">
		<div class="mx-auto max-w-5xl px-6 py-6 text-sm text-muted-foreground">
			&copy; {new Date().getFullYear()} Nuage by <a href="https://facile.studio" class="underline hover:cursor-pointer font-semibold">Facile.</a>
		</div>
	</footer>
</div>
{/if}
