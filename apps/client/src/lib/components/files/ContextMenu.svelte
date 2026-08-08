<script module lang="ts">
	export type MenuItem = {
		label: string;
		icon: string;
		tone?: 'danger';
		onSelect: () => void;
	};
</script>

<script lang="ts">
	import { tick } from 'svelte';

	let {
		x,
		y,
		items,
		heading,
		onClose
	}: {
		x: number;
		y: number;
		items: MenuItem[];
		heading?: { title: string; detail?: string };
		onClose: () => void;
	} = $props();

	let menu = $state<HTMLDivElement | null>(null);
	/* Placed only once it has been measured — painting at the raw pointer position first would
	   show a frame of the menu hanging off the edge before it snapped back inside. */
	let position = $state<{ left: number; top: number } | null>(null);

	const MARGIN = 8;

	/*
	 * A menu opened near the right or bottom edge would otherwise render off screen — the same
	 * obligation muse's SpaceSwitcher documents for its dropdown. It is measured after mount
	 * rather than estimated, because the item count varies with selection and file type.
	 */
	$effect(() => {
		const el = menu;
		if (!el) return;
		void x;
		void y;

		tick().then(() => {
			const { width, height } = el.getBoundingClientRect();
			position = {
				left: Math.max(MARGIN, Math.min(x, window.innerWidth - width - MARGIN)),
				top: Math.max(MARGIN, Math.min(y, window.innerHeight - height - MARGIN))
			};
			el.querySelector<HTMLButtonElement>('button')?.focus();
		});
	});

	function onKeydown(e: KeyboardEvent) {
		if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
		e.preventDefault();
		const buttons = [...(menu?.querySelectorAll<HTMLButtonElement>('button') ?? [])];
		const current = buttons.indexOf(document.activeElement as HTMLButtonElement);
		const next = (current + (e.key === 'ArrowDown' ? 1 : -1) + buttons.length) % buttons.length;
		buttons[next]?.focus();
	}

	function choose(item: MenuItem) {
		onClose();
		item.onSelect();
	}
</script>

<!-- A floating surface, so it pairs its border with a shadow (CHARTE §5). Clicks inside must
     not reach the document listener that closes it. -->
<div
	bind:this={menu}
	role="menu"
	tabindex="-1"
	class="fixed z-40 min-w-48 rounded-fc-md border border-fc-border bg-fc-component py-1 shadow-lg {position
		? ''
		: 'invisible'}"
	style="left: {position?.left ?? 0}px; top: {position?.top ?? 0}px"
	onclick={(e) => e.stopPropagation()}
	oncontextmenu={(e) => e.preventDefault()}
	onkeydown={onKeydown}
>
	{#if heading}
		<div class="px-3 py-2">
			<p class="truncate text-fc-sm font-medium text-fc-fg">{heading.title}</p>
			{#if heading.detail}
				<p class="truncate text-fc-xs text-fc-fg-muted">{heading.detail}</p>
			{/if}
		</div>
		<hr class="my-1 border-0 border-t border-fc-border" />
	{/if}

	{#each items as item (item.label)}
		<button
			type="button"
			role="menuitem"
			onclick={() => choose(item)}
			class="flex w-full items-center gap-2.5 px-3 py-2 text-left text-fc-sm transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-fc-ring {item.tone ===
			'danger'
				? 'text-fc-fg-muted hover:bg-fc-danger/10 hover:text-fc-danger'
				: 'text-fc-fg hover:bg-fc-surface'}"
		>
			<iconify-icon icon={item.icon} width="16" height="16" class="block shrink-0"></iconify-icon>
			{item.label}
		</button>
	{/each}
</div>
