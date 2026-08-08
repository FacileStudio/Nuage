<script lang="ts">
	import { onMount } from 'svelte';

	/*
	 * Scalar renders the OpenAPI spec. It is the one third-party script Nuage still executes,
	 * and it is pinned with an integrity hash on purpose.
	 *
	 * The URL used to be versionless (`@scalar/api-reference`, i.e. `@latest`), which meant this
	 * origin auto-executed whatever Scalar published next. That is the same origin that holds
	 * `nuage.token` in localStorage, so a bad release — or a compromised CDN — would have read
	 * every visitor's session token. Pinning plus `integrity` makes the browser refuse anything
	 * whose bytes are not the ones reviewed here.
	 *
	 * To upgrade: bump VERSION, then re-derive the hash from the exact file —
	 *   curl -sL <url> | openssl dgst -sha384 -binary | openssl base64 -A
	 * Never bump one without the other; a stale hash fails closed (the script is blocked),
	 * which is the safe direction but does render an empty page.
	 */
	const VERSION = '1.64.1';
	const INTEGRITY = 'sha384-yNQdqLDpE2fst+aUqSHXcquVibo90vCkT+zBMLgYfCejLv85GXAR3tFg9lXDUJAd';
	const SRC = `https://cdn.jsdelivr.net/npm/@scalar/api-reference@${VERSION}/dist/browser/standalone.js`;

	let failed = $state(false);

	onMount(() => {
		const container = document.getElementById('scalar-api');
		if (!container) return;

		const ref = document.createElement('script');
		ref.id = 'api-reference';
		ref.dataset.url = '/api/docs/openapi.yaml';
		container.appendChild(ref);

		const loader = document.createElement('script');
		loader.src = SRC;
		loader.integrity = INTEGRITY;
		/* Required for SRI to be checked at all on a cross-origin script: without CORS the
		   response is opaque and the browser cannot hash it. */
		loader.crossOrigin = 'anonymous';
		loader.onerror = () => (failed = true);
		container.appendChild(loader);
	});
</script>

<svelte:head>
	<title>Nuage API</title>
	<style>
		body { margin: 0; }
	</style>
</svelte:head>

{#if failed}
	<div class="mx-auto flex max-w-md flex-col gap-2 p-8 text-center">
		<p class="text-fc-sm font-medium text-fc-fg">The API reference could not load.</p>
		<p class="text-fc-sm text-fc-fg-muted">
			It renders through a pinned third-party script, so this instance may be offline or the
			pin may be stale. The spec itself is always available at
			<a class="underline" href="/api/docs/openapi.yaml">/api/docs/openapi.yaml</a>.
		</p>
	</div>
{/if}

<div id="scalar-api"></div>
