<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { theme } from '$lib/theme.svelte';

	let { children } = $props();

	if (browser) {
		theme.restore();
		void import('iconify-icon');
	}
</script>

{@render children()}
