/** Registre's origin: the OIDC issuer, and the only host plan upgrade links may point at. */
export const REGISTRE_ORIGIN = 'https://sso.facile.studio';

/** Where shared spaces are created, renamed and staffed. Nuage only mirrors them. */
export const REGISTRE_SPACES_URL = `${REGISTRE_ORIGIN}/spaces`;

/**
 * Returns the Registre page for a space, or the list of spaces when it is not linked.
 */
export function registreSpaceUrl(registreSpaceId?: string | null): string {
	return registreSpaceId ? `${REGISTRE_SPACES_URL}/${registreSpaceId}` : REGISTRE_SPACES_URL;
}
