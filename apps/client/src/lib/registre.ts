/** Where shared spaces are created, renamed and staffed. Nuage only mirrors them. */
const REGISTRE_SPACES_URL = 'https://sso.facile.studio/spaces';

/**
 * Returns the Registre page for a space, or the list of spaces when it is not linked.
 */
export function registreSpaceUrl(registreSpaceId?: string | null): string {
	return registreSpaceId ? `${REGISTRE_SPACES_URL}/${registreSpaceId}` : REGISTRE_SPACES_URL;
}
