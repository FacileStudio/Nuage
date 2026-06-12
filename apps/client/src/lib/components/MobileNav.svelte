<script lang="ts">
	import { page } from '$app/state';

	const items = [
		{ href: '/files', label: 'Files', icon: 'solar:folder-linear' },
		{ href: '/shared', label: 'Shared', icon: 'solar:share-linear' },
		{ href: '/trash', label: 'Trash', icon: 'solar:trash-bin-2-linear' },
		{ href: '/activity', label: 'Activity', icon: 'solar:history-linear' },
		{ href: '/settings', label: 'Settings', icon: 'solar:settings-linear' }
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
	</div>
</nav>
