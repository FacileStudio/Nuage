let source: () => string = () => 'en';

/** Points the Intl helpers at the locale the root layout resolved. */
export function bindActive(next: () => string) {
	source = next;
}

/** The resolved locale (fr, en, es or de) every Intl formatter in the app uses. */
export function activeLocale(): string {
	return source();
}
