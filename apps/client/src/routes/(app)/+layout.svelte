<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { backend, type UserProfile, type QuotaResponse } from '$lib/backend';

	let { children } = $props();

	const TOKEN_KEY = 'nuage.token';

	let token = $state('');
	let user = $state<UserProfile | null>(null);
	let loaded = $state(false);
	let mobileMenuOpen = $state(false);
	let quota = $state<QuotaResponse | null>(null);

	function setUser(nextUser: UserProfile) {
		user = nextUser;
	}

	async function refreshQuota() {
		if (!token) return;
		try {
			quota = await backend.getQuota(token);
		} catch {
			quota = null;
		}
	}

	setContext('app', {
		get token() { return token; },
		get user() { return user; },
		setUser,
		refreshQuota
	});

	onMount(async () => {
		const stored = localStorage.getItem(TOKEN_KEY) ?? '';
		if (!stored) {
			goto('/login');
			return;
		}
		try {
			const result = await backend.me(stored);
			token = stored;
			user = result.user;
			loaded = true;
			refreshQuota();
		} catch {
			goto('/login');
		}
	});

	function getInitials(value: string) {
		const parts = value.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase();
	}

	function userLabel(u: UserProfile | null) {
		return u?.name?.trim() || u?.email || '';
	}

	function logout() {
		localStorage.removeItem(TOKEN_KEY);
		goto('/login');
	}

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(1024));
		return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
	}

	let quotaPercentage = $derived(quota ? Math.min(quota.percentage, 100) : 0);
	let quotaBarColor = $derived(
		quotaPercentage >= 90 ? 'bg-red-500' :
		quotaPercentage >= 80 ? 'bg-amber-500' :
		'bg-primary'
	);
	let hasLimit = $derived(quota != null && quota.storage_limit > 0);

	const navLinks: { href: string; label: string; icon: string; disabled: boolean }[] = [
		{ href: '/files', label: 'Files', icon: 'solar:folder-linear', disabled: false },
		{ href: '/shared', label: 'Shared', icon: 'solar:share-linear', disabled: false },
		{ href: '/trash', label: 'Trash', icon: 'solar:trash-bin-2-linear', disabled: false },
		{ href: '/activity', label: 'Activity', icon: 'solar:history-linear', disabled: false },
		{ href: '/settings', label: 'Settings', icon: 'solar:settings-linear', disabled: false }
	];
</script>

{#if loaded}
	<div class="flex h-[100dvh] w-full overflow-hidden">
		<button
			class="fixed top-4 left-4 z-50 flex h-10 w-10 items-center justify-center rounded-md border border-border bg-background md:hidden"
			onclick={() => mobileMenuOpen = !mobileMenuOpen}
			aria-label="Toggle menu"
		>
			<iconify-icon icon={mobileMenuOpen ? 'solar:close-circle-linear' : 'solar:hamburger-menu-linear'} width="20"></iconify-icon>
		</button>

		{#if mobileMenuOpen}
			<button
				class="fixed inset-0 z-30 bg-black/40 md:hidden"
				onclick={() => mobileMenuOpen = false}
				aria-label="Close menu"
			></button>
		{/if}

		<aside class="fixed z-40 top-0 left-0 flex h-[100dvh] w-60 flex-col border-r bg-background transition-transform md:sticky md:translate-x-0 {mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}">
			<div class="flex items-center gap-3 px-5 pt-8 pb-6">
				<iconify-icon icon="solar:cloud-bold-duotone" width="28" class="text-foreground"></iconify-icon>
				<span class="text-2xl font-bold font-heading tracking-tight">Nuage</span>
			</div>

			<nav class="flex flex-1 flex-col gap-1 px-3">
				{#each navLinks as link}
					{@const active = page.url.pathname === link.href || page.url.pathname.startsWith(link.href + '/')}
					{#if link.disabled}
						<span
							class="flex items-center gap-3 rounded-md px-3 py-2.5 text-sm text-muted-foreground/50 cursor-not-allowed"
						>
							<iconify-icon icon={link.icon} width="16"></iconify-icon>
							{link.label}
						</span>
					{:else}
						<a
							href={link.href}
							onclick={() => mobileMenuOpen = false}
							class="flex items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors {active
								? 'bg-foreground text-background font-medium'
								: 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
						>
							<iconify-icon icon={link.icon} width="16"></iconify-icon>
							{link.label}
						</a>
					{/if}
				{/each}
			</nav>

			<div class="h-px bg-border"></div>

			{#if quota}
				<div class="px-4 pt-3">
					{#if hasLimit}
						<div class="flex items-center justify-between text-[11px] text-muted-foreground">
							<span>{formatSize(quota.storage_used)} / {formatSize(quota.storage_limit)}</span>
							<span>{Math.round(quotaPercentage)}%</span>
						</div>
						<div class="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-muted">
							<div
								class="h-full rounded-full transition-all duration-300 {quotaBarColor}"
								style="width: {quotaPercentage}%"
							></div>
						</div>
					{:else}
						<div class="flex items-center gap-1.5 text-[11px] text-muted-foreground">
							<iconify-icon icon="solar:cloud-bold-duotone" width="14"></iconify-icon>
							<span>{formatSize(quota.storage_used)} used</span>
						</div>
					{/if}
				</div>
			{/if}

			<div class="flex flex-col gap-2 p-4">
				<div class="flex items-center gap-3 rounded-xl border border-border/70 bg-muted/40 p-2.5">
					<div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-border bg-foreground text-xs font-semibold text-background">
						{getInitials(userLabel(user))}
					</div>
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm font-medium">{user?.name || 'Set your profile'}</p>
						<p class="truncate text-xs text-muted-foreground">{user?.email ?? ''}</p>
					</div>
				</div>
				<button
					onclick={logout}
					class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:text-destructive hover:bg-destructive/10"
				>
					<iconify-icon icon="solar:logout-2-linear" width="16"></iconify-icon>
					Logout
				</button>
			</div>
		</aside>

		<main class="flex-1 overflow-auto">
			{@render children()}
		</main>
	</div>
{/if}
