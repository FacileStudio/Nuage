import type { Row } from '../row';

/** The file browser, its menus and dialogs, the viewers, sharing and the public link page. */
export const files = {
	'files.title': ['Files — Nuage', 'Fichiers — Nuage', 'Archivos — Nuage', 'Dateien — Nuage'],
	'files.search': ['Search files', 'Rechercher des fichiers', 'Buscar archivos', 'Dateien suchen'],
	'files.viewList': ['List view', 'Vue en liste', 'Vista de lista', 'Listenansicht'],
	'files.viewGrid': ['Grid view', 'Vue en grille', 'Vista de cuadrícula', 'Rasteransicht'],
	'files.selected': [
		{ one: '{count} selected', other: '{count} selected' },
		{ one: '{count} sélectionné', other: '{count} sélectionnés' },
		{ one: '{count} seleccionado', other: '{count} seleccionados' },
		{ one: '{count} ausgewählt', other: '{count} ausgewählt' }
	],
	'files.selectAll': ['Select all', 'Tout sélectionner', 'Seleccionar todo', 'Alle auswählen'],
	'files.deselectAll': ['Deselect all', 'Tout désélectionner', 'Deseleccionar todo', 'Auswahl aufheben'],
	'files.move': ['Move', 'Déplacer', 'Mover', 'Verschieben'],
	'files.moveToTrash': ['Move to trash', 'Mettre à la corbeille', 'Mover a la papelera', 'In den Papierkorb'],
	'files.upload': ['Upload', 'Envoyer', 'Subir', 'Hochladen'],
	'files.uploadFiles': ['Upload files', 'Envoyer des fichiers', 'Subir archivos', 'Dateien hochladen'],
	'files.newFolder': ['New folder', 'Nouveau dossier', 'Nueva carpeta', 'Neuer Ordner'],
	'files.select': ['Select', 'Sélectionner', 'Seleccionar', 'Auswählen'],
	'files.emptyTitle': ['This folder is empty', 'Ce dossier est vide', 'Esta carpeta está vacía', 'Dieser Ordner ist leer'],
	'files.emptyText': [
		'Drop files anywhere on this page, or use the buttons above.',
		'Déposez des fichiers n’importe où sur cette page, ou utilisez les boutons ci-dessus.',
		'Suelta archivos en cualquier parte de esta página o usa los botones de arriba.',
		'Ziehe Dateien irgendwo auf diese Seite oder nutze die Schaltflächen oben.'
	],
	'files.noMatchTitle': ['Nothing matches that', 'Aucun résultat', 'Nada coincide', 'Keine Treffer'],
	'files.noMatchText': [
		'Try a shorter term — search looks at file names only.',
		'Essayez un terme plus court : la recherche porte uniquement sur les noms de fichiers.',
		'Prueba con un término más corto: la búsqueda solo mira los nombres de archivo.',
		'Versuche einen kürzeren Begriff: Die Suche betrachtet nur Dateinamen.'
	],
	'files.dropToUpload': ['Drop to upload', 'Déposez pour envoyer', 'Suelta para subir', 'Zum Hochladen ablegen'],
	'files.loadFailed': [
		'Could not load this folder.',
		'Impossible de charger ce dossier.',
		'No se pudo cargar esta carpeta.',
		'Dieser Ordner konnte nicht geladen werden.'
	],
	'files.uploadsFailed': [
		'{failed} of {total} uploads failed.',
		'{failed} envois sur {total} ont échoué.',
		'Fallaron {failed} de {total} subidas.',
		'{failed} von {total} Uploads sind fehlgeschlagen.'
	],
	'files.uploadedOne': ['Uploaded.', 'Envoyé.', 'Subido.', 'Hochgeladen.'],
	'files.uploadedMany': [
		'Uploaded {count} files.',
		'{count} fichiers envoyés.',
		'Se subieron {count} archivos.',
		'{count} Dateien hochgeladen.'
	],
	'files.uploadFailed': ['Upload failed', 'Échec de l’envoi', 'Falló la subida', 'Upload fehlgeschlagen'],
	'files.createFailed': [
		'Could not create that folder.',
		'Impossible de créer ce dossier.',
		'No se pudo crear esa carpeta.',
		'Dieser Ordner konnte nicht erstellt werden.'
	],
	'files.renameFailed': [
		'Could not rename that.',
		'Impossible de renommer cet élément.',
		'No se pudo cambiar el nombre.',
		'Das konnte nicht umbenannt werden.'
	],
	'files.renamedTo': [
		'Renamed to {name}',
		'Renommé en {name}',
		'Renombrado a {name}',
		'Umbenannt in {name}'
	],
	'files.moveFailed': [
		{ one: '{count} item could not be moved.', other: '{count} items could not be moved.' },
		{ one: '{count} élément n’a pas pu être déplacé.', other: '{count} éléments n’ont pas pu être déplacés.' },
		{ one: 'No se pudo mover {count} elemento.', other: 'No se pudieron mover {count} elementos.' },
		{ one: '{count} Element konnte nicht verschoben werden.', other: '{count} Elemente konnten nicht verschoben werden.' }
	],
	'files.folderFallback': ['folder', 'le dossier', 'la carpeta', 'Ordner'],
	'files.movedTo': [
		{ one: 'Moved to {where}', other: '{count} items moved to {where}' },
		{ one: 'Déplacé vers {where}', other: '{count} éléments déplacés vers {where}' },
		{ one: 'Movido a {where}', other: '{count} elementos movidos a {where}' },
		{ one: 'Verschoben nach {where}', other: '{count} Elemente nach {where} verschoben' }
	],
	'files.deleteFailed': [
		{ one: '{count} item could not be deleted.', other: '{count} items could not be deleted.' },
		{ one: '{count} élément n’a pas pu être supprimé.', other: '{count} éléments n’ont pas pu être supprimés.' },
		{ one: 'No se pudo eliminar {count} elemento.', other: 'No se pudieron eliminar {count} elementos.' },
		{ one: '{count} Element konnte nicht gelöscht werden.', other: '{count} Elemente konnten nicht gelöscht werden.' }
	],
	'files.trashedOne': [
		'“{name}” moved to trash',
		'« {name} » mis à la corbeille',
		'«{name}» movido a la papelera',
		'„{name}“ in den Papierkorb verschoben'
	],
	'files.trashedMany': [
		'{count} items moved to trash',
		'{count} éléments mis à la corbeille',
		'{count} elementos movidos a la papelera',
		'{count} Elemente in den Papierkorb verschoben'
	],
	'files.undo': ['Undo', 'Annuler', 'Deshacer', 'Rückgängig'],
	'files.undoFailed': [
		'Could not undo that.',
		'Impossible d’annuler.',
		'No se pudo deshacer.',
		'Das konnte nicht rückgängig gemacht werden.'
	],
	'files.menu.download': ['Download', 'Télécharger', 'Descargar', 'Herunterladen'],
	'files.menu.rename': ['Rename', 'Renommer', 'Cambiar nombre', 'Umbenennen'],
	'files.menu.share': ['Share', 'Partager', 'Compartir', 'Teilen'],
	'files.menu.moveTo': ['Move to…', 'Déplacer vers…', 'Mover a…', 'Verschieben nach…'],
	'files.menu.moveMany': [
		'Move {count} items…',
		'Déplacer {count} éléments…',
		'Mover {count} elementos…',
		'{count} Elemente verschieben…'
	],
	'files.menu.moveParent': ['Move to {name}', 'Déplacer vers {name}', 'Mover a {name}', 'Nach {name} verschieben'],
	'files.menu.trashMany': [
		'Move {count} items to trash',
		'Mettre {count} éléments à la corbeille',
		'Mover {count} elementos a la papelera',
		'{count} Elemente in den Papierkorb'
	],
	'files.menu.folder': ['Folder', 'Dossier', 'Carpeta', 'Ordner'],
	'files.newFolderTitle': ['New folder', 'Nouveau dossier', 'Nueva carpeta', 'Neuer Ordner'],
	'files.newFolderPlaceholder': ['Invoices', 'Factures', 'Facturas', 'Rechnungen'],
	'files.trashTitle': [
		'Move to trash?',
		'Mettre à la corbeille ?',
		'¿Mover a la papelera?',
		'In den Papierkorb verschieben?'
	],
	'files.trashOne': [
		'“{name}” goes to the trash. You can restore it from there, and nothing is removed from the server yet.',
		'« {name} » va à la corbeille. Vous pouvez le restaurer depuis là, et rien n’est encore supprimé du serveur.',
		'«{name}» va a la papelera. Puedes restaurarlo desde allí y todavía no se borra nada del servidor.',
		'„{name}“ kommt in den Papierkorb. Du kannst es dort wiederherstellen, und vom Server wird noch nichts entfernt.'
	],
	'files.trashMany': [
		'{count} items go to the trash. You can restore them from there, and nothing is removed from the server yet.',
		'{count} éléments vont à la corbeille. Vous pouvez les restaurer depuis là, et rien n’est encore supprimé du serveur.',
		'{count} elementos van a la papelera. Puedes restaurarlos desde allí y todavía no se borra nada del servidor.',
		'{count} Elemente kommen in den Papierkorb. Du kannst sie dort wiederherstellen, und vom Server wird noch nichts entfernt.'
	],
	'files.breadcrumb': ['Breadcrumb', 'Fil d’Ariane', 'Ruta de navegación', 'Brotkrümelnavigation'],
	'files.upTo': ['Up to {name}', 'Remonter vers {name}', 'Subir a {name}', 'Hoch zu {name}'],
	'files.upToDrop': [
		'Up to {name} — drop here to move',
		'Remonter vers {name} : déposez ici pour déplacer',
		'Subir a {name}: suelta aquí para mover',
		'Hoch zu {name}: hier ablegen zum Verschieben'
	],
	'files.newNameFor': ['New name for {name}', 'Nouveau nom pour {name}', 'Nuevo nombre para {name}', 'Neuer Name für {name}'],
	'files.selectItem': ['Select {name}', 'Sélectionner {name}', 'Seleccionar {name}', '{name} auswählen'],
	'files.selectEverything': [
		'Select everything in this folder',
		'Sélectionner tout le contenu de ce dossier',
		'Seleccionar todo en esta carpeta',
		'Alles in diesem Ordner auswählen'
	],
	'files.actionsFor': ['Actions for {name}', 'Actions pour {name}', 'Acciones para {name}', 'Aktionen für {name}'],
	'files.open': ['Open {name}', 'Ouvrir {name}', 'Abrir {name}', '{name} öffnen'],
	'files.folders': ['Folders', 'Dossiers', 'Carpetas', 'Ordner'],
	'files.files': ['Files', 'Fichiers', 'Archivos', 'Dateien'],
	'files.actions': ['Actions', 'Actions', 'Acciones', 'Aktionen'],
	'move.single': ['Move “{name}”', 'Déplacer « {name} »', 'Mover «{name}»', '„{name}“ verschieben'],
	'move.many': [
		'Move {count} items',
		'Déplacer {count} éléments',
		'Mover {count} elementos',
		'{count} Elemente verschieben'
	],
	'move.destination': ['Destination', 'Destination', 'Destino', 'Ziel'],
	'move.loadFailed': [
		'Could not load folders.',
		'Impossible de charger les dossiers.',
		'No se pudieron cargar las carpetas.',
		'Ordner konnten nicht geladen werden.'
	],
	'move.none': ['No subfolders here', 'Aucun sous-dossier ici', 'No hay subcarpetas aquí', 'Hier gibt es keine Unterordner'],
	'move.here': ['Already here', 'Déjà ici', 'Ya está aquí', 'Bereits hier'],
	'move.to': ['Move to {name}', 'Déplacer vers {name}', 'Mover a {name}', 'Nach {name} verschieben']
} satisfies Record<string, Row>;
