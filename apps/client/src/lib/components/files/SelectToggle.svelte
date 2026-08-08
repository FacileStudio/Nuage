<script lang="ts">
	import { icons, nuage } from '$lib/icons';

	let {
		checked = false,
		indeterminate = false,
		label,
		onToggle
	}: {
		checked?: boolean;
		indeterminate?: boolean;
		label: string;
		onToggle: (e: MouseEvent) => void;
	} = $props();
</script>

<!--
  A real <button role="checkbox">, not a <div> with an onclick. The version this replaces was
  a div carrying `role="checkbox"` and `aria-checked` with no tabindex and no key handler, so
  it announced itself as a checkbox to a screen reader and then could not be reached or
  operated by one — the worst of both, and what svelte-check was warning about on nine lines.
-->
<button
	type="button"
	role="checkbox"
	aria-checked={indeterminate ? 'mixed' : checked}
	aria-label={label}
	onclick={onToggle}
	class="flex size-5 shrink-0 items-center justify-center rounded-fc-pill border-2 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring {checked ||
	indeterminate
		? 'border-fc-accent bg-fc-accent text-fc-accent-fg'
		: 'border-fc-border bg-fc-bg text-transparent hover:border-fc-fg-muted'}"
>
	{#if checked}
		<iconify-icon icon={nuage.tick} width="14" height="14" class="block"></iconify-icon>
	{:else if indeterminate}
		<iconify-icon icon={icons.minus} width="14" height="14" class="block"></iconify-icon>
	{/if}
</button>
