<script lang="ts">
	import { page } from '$app/state';
	import type { UserProfile } from '$lib/backend';

	let { user = null }: { user?: UserProfile | null } = $props();

	let avatarFailed = $state(false);

	// Reset the fallback when the avatar URL changes (e.g. after profile sync).
	$effect(() => {
		void user?.avatar_url;
		avatarFailed = false;
	});

	function getInitials(value: string) {
		const parts = value.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase();
	}

	const userLabel = $derived(user?.name?.trim() || user?.email || '');
	const settingsActive = $derived(isActive('/settings'));

	const items = [
		{ href: '/files', label: 'Files', icon: 'solar:folder-linear' },
		{ href: '/shared', label: 'Shared', icon: 'solar:share-linear' },
		{ href: '/trash', label: 'Trash', icon: 'solar:trash-bin-2-linear' },
		{ href: '/activity', label: 'Activity', icon: 'solar:history-linear' }
	];

	function isActive(href: string) {
		return page.url.pathname === href || page.url.pathname.startsWith(href + '/');
	}
</script>

<nav
	class="fixed inset-x-0 z-50 flex justify-center px-4 md:hidden"
	style="bottom: max(0.75rem, env(safe-area-inset-bottom))"
>
	<div
		class="flex items-center gap-1 rounded-full border border-border/40 bg-background/55 p-1.5 shadow-lg shadow-black/10 ring-1 ring-white/10 backdrop-blur-2xl backdrop-saturate-150"
	>
		{#each items as item (item.href)}
			{@const active = isActive(item.href)}
			<a
				href={item.href}
				aria-label={item.label}
				title={item.label}
				class="flex items-center justify-center rounded-full px-3.5 py-2 transition-all duration-200 {active
					? 'bg-foreground text-background shadow-sm'
					: 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'}"
			>
				<iconify-icon icon={item.icon} width="22"></iconify-icon>
			</a>
		{/each}

		<a
			href="/settings"
			aria-label="Settings"
			title="Settings"
			class="flex items-center justify-center rounded-full px-2.5 py-1.5 transition-all duration-200 {settingsActive
				? 'bg-foreground shadow-sm'
				: 'hover:bg-muted/60'}"
		>
			{#if user?.avatar_url && !avatarFailed}
				<img
					src={user.avatar_url}
					alt={userLabel}
					class="h-7 w-7 rounded-full border border-border object-cover"
					onerror={() => (avatarFailed = true)}
				/>
			{:else}
				<span
					class="flex h-7 w-7 items-center justify-center rounded-full border border-border bg-foreground text-[10px] font-semibold text-background"
				>
					{getInitials(userLabel)}
				</span>
			{/if}
		</a>
	</div>
</nav>
