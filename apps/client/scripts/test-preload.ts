import { mock } from 'bun:test';

/** bun test runs outside SvelteKit, so the two $app modules @facile/client imports are stubbed. */
mock.module('$app/environment', () => ({ browser: false }));
mock.module('$app/navigation', () => ({ goto: async () => {} }));
