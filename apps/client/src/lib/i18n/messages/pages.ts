import type { Row } from '../row';

/** App data, shared links, trash and activity pages. */
export const pages = {
	'appdata.title': ['App data', 'Données des apps', 'Datos de apps', 'App-Daten'],
	'appdata.description': [
		'Private storage other Facile apps keep for you. It counts toward your storage plan and is never shown as a folder or a space.',
		'Espace privé que d’autres apps Facile conservent pour vous. Il compte dans votre forfait de stockage et n’apparaît jamais comme un dossier ou un espace.',
		'Almacenamiento privado que otras apps de Facile guardan para ti. Cuenta en tu plan de almacenamiento y nunca se muestra como carpeta o espacio.',
		'Privater Speicher, den andere Facile-Apps für dich führen. Er zählt zu deinem Speichertarif und erscheint nie als Ordner oder Bereich.'
	],
	'appdata.emptyTitle': ['No app data', 'Aucune donnée d’app', 'Sin datos de apps', 'Keine App-Daten'],
	'appdata.emptyText': [
		'When an app such as Plume or Ardoise stores something for you, it shows here.',
		'Quand une app comme Plume ou Ardoise stocke quelque chose pour vous, cela apparaît ici.',
		'Cuando una app como Plume o Ardoise guarda algo para ti, aparece aquí.',
		'Wenn eine App wie Plume oder Ardoise etwas für dich speichert, erscheint es hier.'
	],
	'appdata.app': ['App', 'App', 'App', 'App'],
	'appdata.items': ['Items', 'Éléments', 'Elementos', 'Elemente'],
	'appdata.used': ['Used', 'Utilisé', 'Usado', 'Belegt'],
	'appdata.deleted': [
		'Deleted what {name} stored.',
		'Données de {name} supprimées.',
		'Se eliminó lo que guardaba {name}.',
		'Von {name} gespeicherte Daten gelöscht.'
	],
	'appdata.deleteFailed': [
		'Could not delete that app data.',
		'Impossible de supprimer ces données.',
		'No se pudieron eliminar esos datos.',
		'Diese App-Daten konnten nicht gelöscht werden.'
	],
	'appdata.confirmTitle': [
		'Delete this app’s data?',
		'Supprimer les données de cette app ?',
		'¿Eliminar los datos de esta app?',
		'Daten dieser App löschen?'
	],
	'appdata.confirmText': [
		'Everything {name} stored for you ({size}) is deleted for good. The app may lose documents it relies on, and this cannot be undone.',
		'Tout ce que {name} a stocké pour vous ({size}) est supprimé définitivement. L’app peut perdre des documents dont elle dépend, et c’est irréversible.',
		'Todo lo que {name} guardó para ti ({size}) se elimina para siempre. La app puede perder documentos de los que depende y no se puede deshacer.',
		'Alles, was {name} für dich gespeichert hat ({size}), wird endgültig gelöscht. Die App kann Dokumente verlieren, die sie braucht, und das lässt sich nicht rückgängig machen.'
	],
	'shared.title': ['Shared links — Nuage', 'Liens partagés — Nuage', 'Enlaces compartidos — Nuage', 'Geteilte Links — Nuage'],
	'shared.heading': ['Shared links', 'Liens partagés', 'Enlaces compartidos', 'Geteilte Links'],
	'shared.description': [
		'Every public link you have handed out. Anyone holding one can open it without an account.',
		'Tous les liens publics que vous avez distribués. Toute personne qui en détient un peut l’ouvrir sans compte.',
		'Todos los enlaces públicos que has repartido. Quien tenga uno puede abrirlo sin cuenta.',
		'Alle öffentlichen Links, die du weitergegeben hast. Wer einen besitzt, kann ihn ohne Konto öffnen.'
	],
	'shared.emptyTitle': ['No shared links yet', 'Aucun lien partagé', 'Aún no hay enlaces compartidos', 'Noch keine geteilten Links'],
	'shared.emptyText': [
		'Open a file’s menu in Files and choose Share to publish a link.',
		'Ouvrez le menu d’un fichier dans Fichiers et choisissez Partager pour publier un lien.',
		'Abre el menú de un archivo en Archivos y elige Compartir para publicar un enlace.',
		'Öffne in Dateien das Menü einer Datei und wähle Teilen, um einen Link zu veröffentlichen.'
	],
	'shared.goToFiles': ['Go to Files', 'Aller aux fichiers', 'Ir a Archivos', 'Zu den Dateien'],
	'shared.link': ['Link', 'Lien', 'Enlace', 'Link'],
	'shared.expiry': ['Expiry', 'Expiration', 'Caducidad', 'Ablauf'],
	'shared.created': ['Created', 'Créé', 'Creado', 'Erstellt'],
	'shared.expired': ['Expired', 'Expiré', 'Caducado', 'Abgelaufen'],
	'shared.never': ['Never', 'Jamais', 'Nunca', 'Nie'],
	'shared.copy': ['Copy', 'Copier', 'Copiar', 'Kopieren'],
	'shared.untitled': ['Untitled', 'Sans titre', 'Sin título', 'Ohne Titel'],
	'shared.loadFailed': [
		'Could not load your share links.',
		'Impossible de charger vos liens de partage.',
		'No se pudieron cargar tus enlaces compartidos.',
		'Deine Freigabelinks konnten nicht geladen werden.'
	],
	'shared.revoked': ['Link revoked.', 'Lien révoqué.', 'Enlace revocado.', 'Link widerrufen.'],
	'shared.revokeFailed': [
		'Could not revoke that link.',
		'Impossible de révoquer ce lien.',
		'No se pudo revocar ese enlace.',
		'Dieser Link konnte nicht widerrufen werden.'
	],
	'shared.copied': ['Link copied.', 'Lien copié.', 'Enlace copiado.', 'Link kopiert.'],
	'shared.clipboardFailed': [
		'Could not reach the clipboard.',
		'Impossible d’accéder au presse-papiers.',
		'No se pudo acceder al portapapeles.',
		'Zwischenablage nicht erreichbar.'
	],
	'shared.revokeTitle': ['Revoke this link?', 'Révoquer ce lien ?', '¿Revocar este enlace?', 'Diesen Link widerrufen?'],
	'shared.revokeText': [
		'Anyone who still has the link to “{name}” gets a dead page from now on. The file itself is untouched, and revoking cannot be undone — you would have to share it again.',
		'Toute personne qui possède encore le lien vers « {name} » obtiendra une page morte. Le fichier n’est pas touché, et la révocation est définitive : il faudrait le repartager.',
		'Quien aún tenga el enlace a «{name}» verá una página inexistente desde ahora. El archivo no se toca y revocar no se puede deshacer: habría que volver a compartirlo.',
		'Wer noch den Link zu „{name}“ hat, sieht ab jetzt eine tote Seite. Die Datei selbst bleibt unberührt, und das Widerrufen lässt sich nicht rückgängig machen: Du müsstest sie erneut teilen.'
	]
} satisfies Record<string, Row>;
