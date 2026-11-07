import { t } from '$lib/i18n/index.svelte';

/** The address the API gives the placeholder account that stands for every erased person. */
export const DELETED_USER_EMAIL = 'deleted-user@nuage.invalid';

/** A person's display name, localised for the placeholder that replaced an erased account. */
export function displayName(person: { name?: string; email?: string }): string {
	if (person.email === DELETED_USER_EMAIL) return t('common.deletedUser');
	return person.name?.trim() || person.email || t('common.unknownUser');
}

/** A membership role in the reader's language; an unknown role is shown as the server wrote it. */
export function roleLabel(role: string): string {
	const key = role.trim().toLowerCase();
	if (key === 'owner') return t('role.owner');
	if (key === 'admin') return t('role.admin');
	if (key === 'member') return t('role.member');
	return role;
}
