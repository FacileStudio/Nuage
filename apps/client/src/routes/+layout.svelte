<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { Toaster } from '@facile/muse';
	import { theme } from '$lib/theme.svelte';

	let { children } = $props();

	if (browser) {
		theme.restore();
		void import('iconify-icon');
	}
</script>

{@render children()}

<!-- One queue for the whole app, outside the router so a toast survives navigation. The
     bottom padding clears MobileNav's floating pill, which only exists below `md`. -->
<Toaster class="pb-28 md:pb-6" />
