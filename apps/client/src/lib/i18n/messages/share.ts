import type { Row } from '../row';

/** The share dialog. */
export const share = {
	'share.title': ['Share “{name}”', 'Partager « {name} »', 'Compartir «{name}»', '„{name}“ teilen'],
	'share.checkFailed': [
		'Could not check for an existing link.',
		'Impossible de vérifier l’existence d’un lien.',
		'No se pudo comprobar si ya existe un enlace.',
		'Ein vorhandener Link konnte nicht geprüft werden.'
	],
	'share.created': [
		'Public link created.',
		'Lien public créé.',
		'Enlace público creado.',
		'Öffentlicher Link erstellt.'
	],
	'share.createFailed': [
		'Could not create that link.',
		'Impossible de créer ce lien.',
		'No se pudo crear ese enlace.',
		'Dieser Link konnte nicht erstellt werden.'
	],
	'share.revoked': ['Link revoked.', 'Lien révoqué.', 'Enlace revocado.', 'Link widerrufen.'],
	'share.revokeFailed': [
		'Could not revoke that link.',
		'Impossible de révoquer ce lien.',
		'No se pudo revocar ese enlace.',
		'Dieser Link konnte nicht widerrufen werden.'
	],
	'share.warnFile': [
		'Anyone with this link can open the file without signing in.',
		'Toute personne disposant de ce lien peut ouvrir le fichier sans se connecter.',
		'Cualquier persona con este enlace puede abrir el archivo sin iniciar sesión.',
		'Jeder mit diesem Link kann die Datei ohne Anmeldung öffnen.'
	],
	'share.warnFolder': [
		'Anyone with this link can open the folder without signing in.',
		'Toute personne disposant de ce lien peut ouvrir le dossier sans se connecter.',
		'Cualquier persona con este enlace puede abrir la carpeta sin iniciar sesión.',
		'Jeder mit diesem Link kann den Ordner ohne Anmeldung öffnen.'
	],
	'share.publicLink': ['Public link', 'Lien public', 'Enlace público', 'Öffentlicher Link'],
	'share.expiresAt': ['Expires {date}.', 'Expire le {date}.', 'Caduca el {date}.', 'Läuft am {date} ab.'],
	'share.neverExpires': [
		'This link never expires.',
		'Ce lien n’expire jamais.',
		'Este enlace nunca caduca.',
		'Dieser Link läuft nie ab.'
	],
	'share.noneFile': [
		'There is no public link for this file yet.',
		'Il n’existe pas encore de lien public pour ce fichier.',
		'Aún no hay un enlace público para este archivo.',
		'Für diese Datei gibt es noch keinen öffentlichen Link.'
	],
	'share.noneFolder': [
		'There is no public link for this folder yet.',
		'Il n’existe pas encore de lien public pour ce dossier.',
		'Aún no hay un enlace público para esta carpeta.',
		'Für diesen Ordner gibt es noch keinen öffentlichen Link.'
	],
	'share.expires': ['Expires', 'Expiration', 'Caducidad', 'Ablauf'],
	'share.expiresHelper': [
		'A link that never lapses is the one nobody remembers to revoke.',
		'Un lien qui n’expire jamais est celui que personne ne pense à révoquer.',
		'Un enlace que nunca caduca es el que nadie recuerda revocar.',
		'Ein Link, der nie abläuft, ist der, den niemand widerruft.'
	],
	'share.never': ['Never', 'Jamais', 'Nunca', 'Nie'],
	'share.in1d': ['In 1 day', 'Dans 1 jour', 'En 1 día', 'In 1 Tag'],
	'share.in7d': ['In 7 days', 'Dans 7 jours', 'En 7 días', 'In 7 Tagen'],
	'share.in30d': ['In 30 days', 'Dans 30 jours', 'En 30 días', 'In 30 Tagen'],
	'share.revokeLink': ['Revoke link', 'Révoquer le lien', 'Revocar enlace', 'Link widerrufen'],
	'share.create': ['Create public link', 'Créer un lien public', 'Crear enlace público', 'Öffentlichen Link erstellen'],
	'share.revokeTitle': ['Revoke this link?', 'Révoquer ce lien ?', '¿Revocar este enlace?', 'Diesen Link widerrufen?'],
	'share.revokeFile': [
		'Anyone still holding it gets a dead page. The file itself is untouched, and you would have to share it again to undo this.',
		'Toute personne qui le détient encore obtient une page morte. Le fichier n’est pas touché, et il faudrait le repartager pour annuler.',
		'Quien aún lo tenga verá una página inexistente. El archivo no se toca y tendrías que volver a compartirlo para deshacerlo.',
		'Wer ihn noch hat, sieht eine tote Seite. Die Datei selbst bleibt unberührt, und du müsstest sie erneut teilen, um das rückgängig zu machen.'
	],
	'share.revokeFolder': [
		'Anyone still holding it gets a dead page. The folder itself is untouched, and you would have to share it again to undo this.',
		'Toute personne qui le détient encore obtient une page morte. Le dossier n’est pas touché, et il faudrait le repartager pour annuler.',
		'Quien aún lo tenga verá una página inexistente. La carpeta no se toca y tendrías que volver a compartirla para deshacerlo.',
		'Wer ihn noch hat, sieht eine tote Seite. Der Ordner selbst bleibt unberührt, und du müsstest ihn erneut teilen, um das rückgängig zu machen.'
	]
} satisfies Record<string, Row>;
