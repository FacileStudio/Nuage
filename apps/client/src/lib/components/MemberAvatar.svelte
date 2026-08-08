<script lang="ts">
	import { Avatar } from '@facile/muse';

	let {
		name,
		src,
		color,
		size = 'sm'
	}: {
		name: string;
		src?: string;
		color?: string;
		size?: 'sm' | 'md' | 'lg';
	} = $props();
</script>

<span class="relative inline-flex shrink-0">
	<Avatar {name} {src} {size} />
	{#if color}
		<!--
		  The identity colour is a value out of the user's row, not a design token, so it is
		  rendered inline — the one sanctioned exception to the no-hardcoded-colour rule
		  (CHARTE §13). Ringed in the surface it sits on so it reads as a dot on the avatar
		  rather than a notch cut out of it.
		-->
		<span
			class="absolute -right-0.5 -bottom-0.5 size-2.5 rounded-fc-pill ring-2 ring-fc-component"
			style:background-color={color}
			aria-hidden="true"
		></span>
	{/if}
</span>
