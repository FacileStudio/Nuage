import type { Row } from '../row';

/** The trash page. */
export const trash = {
	'trash.title': ['Trash — Nuage', 'Corbeille — Nuage', 'Papelera — Nuage', 'Papierkorb — Nuage'],
	'trash.heading': ['Trash', 'Corbeille', 'Papelera', 'Papierkorb'],
	'trash.keepFor': [
		{
			one: 'Deleted items are removed for good after {count} day.',
			other: 'Deleted items are removed for good after {count} days.'
		},
		{
			one: 'Les éléments supprimés sont effacés définitivement après {count} jour.',
			other: 'Les éléments supprimés sont effacés définitivement après {count} jours.'
		},
		{
			one: 'Los elementos eliminados se borran para siempre tras {count} día.',
			other: 'Los elementos eliminados se borran para siempre tras {count} días.'
		},
		{
			one: 'Gelöschte Elemente werden nach {count} Tag endgültig entfernt.',
			other: 'Gelöschte Elemente werden nach {count} Tagen endgültig entfernt.'
		}
	],
	'trash.keepForever': [
		'Deleted items stay here until you remove them.',
		'Les éléments supprimés restent ici jusqu’à ce que vous les retiriez.',
		'Los elementos eliminados permanecen aquí hasta que los quites.',
		'Gelöschte Elemente bleiben hier, bis du sie entfernst.'
	],
	'trash.reclaim': [
		'{keep} Emptying the trash would give back at least {size}.',
		'{keep} Vider la corbeille libérerait au moins {size}.',
		'{keep} Vaciar la papelera liberaría al menos {size}.',
		'{keep} Das Leeren des Papierkorbs würde mindestens {size} freigeben.'
	],
	'trash.stillCounts': [
		'{keep} Until then they still count against your storage.',
		'{keep} D’ici là, ils comptent toujours dans votre stockage.',
		'{keep} Hasta entonces siguen contando en tu almacenamiento.',
		'{keep} Bis dahin zählen sie weiterhin zu deinem Speicher.'
	],
	'trash.empty': ['Empty trash', 'Vider la corbeille', 'Vaciar la papelera', 'Papierkorb leeren'],
	'trash.emptyTitle': ['Trash is empty', 'La corbeille est vide', 'La papelera está vacía', 'Der Papierkorb ist leer'],
	'trash.emptyText': [
		'Deleted files and folders land here, so a wrong click is never the end of the story.',
		'Les fichiers et dossiers supprimés arrivent ici : un mauvais clic n’est jamais définitif.',
		'Los archivos y carpetas eliminados llegan aquí, así un clic equivocado nunca es el final.',
		'Gelöschte Dateien und Ordner landen hier, damit ein Fehlklick nie das Ende bedeutet.'
	],
	'trash.deleted': ['Deleted', 'Supprimé', 'Eliminado', 'Gelöscht'],
	'trash.restore': ['Restore', 'Restaurer', 'Restaurar', 'Wiederherstellen'],
	'trash.expiresNow': [
		'Removed for good within the hour',
		'Effacé définitivement d’ici une heure',
		'Se borrará para siempre en menos de una hora',
		'Wird innerhalb einer Stunde endgültig entfernt'
	],
	'trash.expiresIn': [
		{ one: 'Removed for good in {count} day', other: 'Removed for good in {count} days' },
		{ one: 'Effacé définitivement dans {count} jour', other: 'Effacé définitivement dans {count} jours' },
		{ one: 'Se borrará para siempre en {count} día', other: 'Se borrará para siempre en {count} días' },
		{ one: 'Wird in {count} Tag endgültig entfernt', other: 'Wird in {count} Tagen endgültig entfernt' }
	],
	'trash.loadFailed': [
		'Could not load the trash.',
		'Impossible de charger la corbeille.',
		'No se pudo cargar la papelera.',
		'Der Papierkorb konnte nicht geladen werden.'
	],
	'trash.restored': ['Restored “{name}”.', '« {name} » restauré.', '«{name}» restaurado.', '„{name}“ wiederhergestellt.'],
	'trash.restoreFailed': [
		'Could not restore “{name}”.',
		'Impossible de restaurer « {name} ».',
		'No se pudo restaurar «{name}».',
		'„{name}“ konnte nicht wiederhergestellt werden.'
	],
	'trash.deletedForGood': [
		'Deleted “{name}” for good.',
		'« {name} » supprimé définitivement.',
		'«{name}» eliminado para siempre.',
		'„{name}“ endgültig gelöscht.'
	],
	'trash.deleteFailed': [
		'Could not delete “{name}”.',
		'Impossible de supprimer « {name} ».',
		'No se pudo eliminar «{name}».',
		'„{name}“ konnte nicht gelöscht werden.'
	],
	'trash.emptied': ['Trash emptied.', 'Corbeille vidée.', 'Papelera vaciada.', 'Papierkorb geleert.'],
	'trash.emptyFailed': [
		'Could not empty the trash.',
		'Impossible de vider la corbeille.',
		'No se pudo vaciar la papelera.',
		'Der Papierkorb konnte nicht geleert werden.'
	],
	'trash.emptyConfirmTitle': [
		'Empty the trash?',
		'Vider la corbeille ?',
		'¿Vaciar la papelera?',
		'Papierkorb leeren?'
	],
	'trash.emptyConfirmText': [
		{
			one: 'The {count} item goes for good, and the file comes off the server. There is no undo.',
			other: 'All {count} items go for good, and the files come off the server. There is no undo.'
		},
		{
			one: 'L’élément ({count}) est effacé définitivement et le fichier quitte le serveur. Il n’y a pas d’annulation.',
			other: 'Les {count} éléments sont effacés définitivement et les fichiers quittent le serveur. Il n’y a pas d’annulation.'
		},
		{
			one: 'El elemento ({count}) se borra para siempre y el archivo sale del servidor. No hay deshacer.',
			other: 'Los {count} elementos se borran para siempre y los archivos salen del servidor. No hay deshacer.'
		},
		{
			one: 'Das Element ({count}) wird endgültig entfernt, und die Datei verlässt den Server. Es gibt kein Rückgängig.',
			other: 'Alle {count} Elemente werden endgültig entfernt, und die Dateien verlassen den Server. Es gibt kein Rückgängig.'
		}
	],
	'trash.deleteConfirmTitle': [
		'Delete permanently?',
		'Supprimer définitivement ?',
		'¿Eliminar para siempre?',
		'Endgültig löschen?'
	],
	'trash.deleteConfirmText': [
		'“{name}” comes off the server for good. There is no undo, and any share link pointing at it stops working.',
		'« {name} » quitte définitivement le serveur. Il n’y a pas d’annulation, et tout lien de partage qui y mène cesse de fonctionner.',
		'«{name}» sale del servidor para siempre. No hay deshacer y cualquier enlace compartido que apunte a él deja de funcionar.',
		'„{name}“ verlässt den Server endgültig. Es gibt kein Rückgängig, und jeder Freigabelink dorthin funktioniert nicht mehr.'
	],
	'trash.deleteForever': ['Delete forever', 'Supprimer définitivement', 'Eliminar para siempre', 'Endgültig löschen']
} satisfies Record<string, Row>;
