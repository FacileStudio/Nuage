import type { Row } from '../row';

/** The activity feed. */
export const activity = {
	'activity.title': ['Activity — Nuage', 'Activité — Nuage', 'Actividad — Nuage', 'Aktivität — Nuage'],
	'activity.heading': ['Activity', 'Activité', 'Actividad', 'Aktivität'],
	'activity.description': [
		'Everything that has happened to your files and folders.',
		'Tout ce qui est arrivé à vos fichiers et dossiers.',
		'Todo lo que ha ocurrido con tus archivos y carpetas.',
		'Alles, was mit deinen Dateien und Ordnern passiert ist.'
	],
	'activity.emptyTitle': ['No activity yet', 'Aucune activité', 'Aún no hay actividad', 'Noch keine Aktivität'],
	'activity.emptyText': [
		'Uploads, renames, shares and deletions all land here as they happen.',
		'Dépôts, renommages, partages et suppressions apparaissent ici au fur et à mesure.',
		'Las subidas, los cambios de nombre, los enlaces compartidos y las eliminaciones llegan aquí a medida que ocurren.',
		'Uploads, Umbenennungen, Freigaben und Löschungen erscheinen hier, sobald sie passieren.'
	],
	'activity.loadFailed': [
		'Could not load the activity feed.',
		'Impossible de charger le fil d’activité.',
		'No se pudo cargar la actividad.',
		'Der Aktivitätsverlauf konnte nicht geladen werden.'
	],
	'activity.moreFailed': [
		'Could not load more activity.',
		'Impossible de charger plus d’activité.',
		'No se pudo cargar más actividad.',
		'Weitere Aktivität konnte nicht geladen werden.'
	],
	'activity.loadMore': [
		'Load more ({left} left)',
		'Voir plus ({left} restants)',
		'Cargar más (quedan {left})',
		'Mehr laden ({left} übrig)'
	],
	'activity.loading': ['Loading…', 'Chargement…', 'Cargando…', 'Wird geladen…'],
	'activity.today': ['Today', 'Aujourd’hui', 'Hoy', 'Heute'],
	'activity.yesterday': ['Yesterday', 'Hier', 'Ayer', 'Gestern'],
	'activity.file.uploaded': ['Uploaded', 'Déposé', 'Subido', 'Hochgeladen'],
	'activity.file.deleted': ['Deleted', 'Supprimé', 'Eliminado', 'Gelöscht'],
	'activity.file.updated': ['Updated', 'Modifié', 'Actualizado', 'Aktualisiert'],
	'activity.file.versioned': ['New version', 'Nouvelle version', 'Nueva versión', 'Neue Version'],
	'activity.file.restored': ['Restored', 'Restauré', 'Restaurado', 'Wiederhergestellt'],
	'activity.file.permanently_deleted': [
		'Permanently deleted',
		'Supprimé définitivement',
		'Eliminado para siempre',
		'Endgültig gelöscht'
	],
	'activity.file.expired': [
		'Expired from trash',
		'Expiré de la corbeille',
		'Caducado de la papelera',
		'Aus dem Papierkorb abgelaufen'
	],
	'activity.folder.created': ['Created folder', 'Dossier créé', 'Carpeta creada', 'Ordner erstellt'],
	'activity.folder.updated': ['Updated folder', 'Dossier modifié', 'Carpeta actualizada', 'Ordner aktualisiert'],
	'activity.folder.deleted': ['Deleted folder', 'Dossier supprimé', 'Carpeta eliminada', 'Ordner gelöscht'],
	'activity.folder.restored': ['Restored folder', 'Dossier restauré', 'Carpeta restaurada', 'Ordner wiederhergestellt'],
	'activity.folder.permanently_deleted': [
		'Permanently deleted folder',
		'Dossier supprimé définitivement',
		'Carpeta eliminada para siempre',
		'Ordner endgültig gelöscht'
	],
	'activity.folder.expired': [
		'Expired folder from trash',
		'Dossier expiré de la corbeille',
		'Carpeta caducada de la papelera',
		'Ordner aus dem Papierkorb abgelaufen'
	],
	'activity.share.created': ['Shared', 'Partagé', 'Compartido', 'Geteilt'],
	'activity.share.revoked': ['Revoked share', 'Partage révoqué', 'Enlace revocado', 'Freigabe widerrufen']
} satisfies Record<string, Row>;
