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
		class="flex items-center gap-0.5 rounded-2xl border border-border/70 bg-background/90 p-1.5 shadow-xl backdrop-blur-md"
	>
		{#each items as item (item.href)}
			{@const active = isActive(item.href)}
			<a
				href={item.href}
				class="flex flex-col items-center gap-0.5 rounded-xl px-3 py-1.5 text-[10px] font-medium transition-colors {active
					? 'bg-foreground text-background'
					: 'text-muted-foreground hover:text-foreground'}"
			>
				<iconify-icon icon={item.icon} width="20"></iconify-icon>
				<span>{item.label}</span>
			</a>
		{/each}
	</div>
</nav>
