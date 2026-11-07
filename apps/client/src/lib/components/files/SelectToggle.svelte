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
	class="relative flex size-5 shrink-0 items-center justify-center rounded-fc-pill transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring [@media(pointer:coarse)]:before:absolute [@media(pointer:coarse)]:before:-inset-3.5 [@media(pointer:coarse)]:before:content-[''] {checked ||
	indeterminate
		? 'bg-fc-accent text-fc-accent-fg'
		: 'bg-fc-field text-transparent hover:bg-fc-surface'}"
>
	{#if checked}
		<iconify-icon icon={nuage.tick} width="14" height="14" class="block"></iconify-icon>
	{:else if indeterminate}
		<iconify-icon icon={icons.minus} width="14" height="14" class="block"></iconify-icon>
	{/if}
</button>
