<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { Toaster } from '@facile/muse';
	import { addCollection } from 'iconify-icon';
	import { theme } from '$lib/theme.svelte';
	import mdi from '$lib/icons/mdi.json';
	import solar from '$lib/icons/solar.json';

	let { children } = $props();

	if (browser) {
		theme.restore();
		/*
		 * The collections are registered before the first component renders, so no icon is ever
		 * resolved through `api.iconify.design`. Without this every glyph in a self-hosted Nuage
		 * is fetched from a third party at runtime: no icons at all on an air-gapped instance,
		 * someone else's uptime in the critical path, and a request naming the glyphs — so
		 * effectively the page — each user is looking at.
		 *
		 * The import is static on purpose. A dynamic one defined the custom element after the
		 * first paint, so every icon on the page was empty for a beat and then popped in. The app
		 * is a client-only SPA (`ssr = false`), so nothing here ever runs on a server.
		 *
		 * Regenerate with `bun run icons` after adding an icon or bumping @facile/muse.
		 */
		addCollection(solar);
		addCollection(mdi);
	}
</script>

{@render children()}

<!-- One queue for the whole app, outside the router so a toast survives navigation. The
     bottom padding clears MobileNav's floating pill, which only exists below `md`. -->
<Toaster class="pb-28 md:pb-6" />
