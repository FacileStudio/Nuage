export type RoleTone = 'owner' | 'admin' | 'neutral';

/**
 * Maps a membership role onto muse's shared tone vocabulary. `owner` and `admin` have their
 * own tokens; everything else is a plain `neutral` pill.
 *
 * This exists because the role pill was being spelled by hand at every call site, and it had
 * drifted to stock Tailwind amber and blue — palette classes that CHARTE §10 bans by name and
 * that do not follow the theme into dark mode.
 */
export function roleTone(role: string): RoleTone {
	const normalized = role.trim().toLowerCase();
	return normalized === 'owner' || normalized === 'admin' ? normalized : 'neutral';
}
