<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { Toaster } from '@facile/muse';
	import { theme } from '$lib/theme.svelte';
	import mdi from '$lib/icons/mdi.json';
	import solar from '$lib/icons/solar.json';

	let { children } = $props();

	if (browser) {
		theme.restore();
		/*
		 * The collections are registered as soon as the custom element defines itself, so no
		 * icon is ever resolved through `api.iconify.design`. Without this every glyph in a
		 * self-hosted Nuage is fetched from a third party at runtime: no icons at all on an
		 * air-gapped instance, someone else's uptime in the critical path, and a request
		 * naming the glyphs — so effectively the page — each user is looking at.
		 *
		 * Regenerate with `bun run icons` after adding an icon or bumping @facile/muse.
		 */
		void import('iconify-icon').then(({ addCollection }) => {
			addCollection(solar);
			addCollection(mdi);
		});
	}
</script>

{@render children()}

<!-- One queue for the whole app, outside the router so a toast survives navigation. The
     bottom padding clears MobileNav's floating pill, which only exists below `md`. -->
<Toaster class="pb-28 md:pb-6" />
