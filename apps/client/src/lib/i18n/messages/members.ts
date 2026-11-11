import type { Row } from '../row';

/** A space's members and settings pages. */
export const members = {
	'members.title': [
		'Members — {name} — Nuage',
		'Membres — {name} — Nuage',
		'Miembros — {name} — Nuage',
		'Mitglieder — {name} — Nuage'
	],
	'members.description': [
		'Everyone here can read and write every file in the space.',
		'Toutes les personnes ici peuvent lire et modifier chaque fichier de l’espace.',
		'Todas las personas de aquí pueden leer y escribir todos los archivos del espacio.',
		'Alle hier können jede Datei im Bereich lesen und schreiben.'
	],
	'members.add': ['Add member', 'Ajouter un membre', 'Añadir miembro', 'Mitglied hinzufügen'],
	'members.roleFor': ['Role for {name}', 'Rôle de {name}', 'Rol de {name}', 'Rolle von {name}'],
	'members.removeLabel': ['Remove {name}', 'Retirer {name}', 'Quitar a {name}', '{name} entfernen'],
	'members.drawerTitle': ['Add a member', 'Ajouter un membre', 'Añadir un miembro', 'Mitglied hinzufügen'],
	'members.who': ['Who', 'Qui', 'Quién', 'Wer'],
	'members.whoHelper': [
		'People you share a space with. For anyone else, type their full email address.',
		'Les personnes de vos espaces. Pour quelqu’un d’autre, saisissez son adresse email complète.',
		'Las personas con las que compartes un espacio. Para otra persona, escribe su correo completo.',
		'Personen aus deinen Bereichen. Für alle anderen gib die vollständige E-Mail-Adresse ein.'
	],
	'members.search': [
		'Search by name or email',
		'Rechercher par nom ou email',
		'Buscar por nombre o correo',
		'Nach Name oder E-Mail suchen'
	],
	'members.nobody': ['Nobody matches that.', 'Personne ne correspond.', 'Nadie coincide.', 'Niemand passt dazu.'],
	'members.everyone': [
		'Everyone you share a space with is already a member. Type an email address to add someone else.',
		'Toutes les personnes de vos espaces sont déjà membres. Saisissez une adresse email pour ajouter quelqu’un d’autre.',
		'Todas las personas de tus espacios ya son miembros. Escribe un correo para añadir a otra persona.',
		'Alle aus deinen Bereichen sind bereits Mitglied. Gib eine E-Mail-Adresse ein, um jemand anderen hinzuzufügen.'
	],
	'members.byEmail': [
		'{email} will be added if they have an account on this Nuage.',
		'{email} sera ajouté si cette personne a un compte sur ce Nuage.',
		'Se añadirá a {email} si tiene una cuenta en este Nuage.',
		'{email} wird hinzugefügt, wenn die Person ein Konto auf diesem Nuage hat.'
	],
	'members.choose': [
		'Choose a person to add',
		'Choisissez la personne à ajouter',
		'Elige a la persona que añadir',
		'Person zum Hinzufügen auswählen'
	],
	'members.role': ['Role', 'Rôle', 'Rol', 'Rolle'],
	'members.role.member': [
		'Member — read and write files',
		'Membre : lit et écrit les fichiers',
		'Miembro: lee y escribe archivos',
		'Mitglied: liest und schreibt Dateien'
	],
	'members.role.admin': [
		'Admin — also manages members',
		'Administrateur : gère aussi les membres',
		'Administrador: también gestiona miembros',
		'Administrator: verwaltet auch Mitglieder'
	],
	'members.adding': ['Adding…', 'Ajout…', 'Añadiendo…', 'Wird hinzugefügt…'],
	'members.addPerson': ['Add {name}', 'Ajouter {name}', 'Añadir a {name}', '{name} hinzufügen'],
	'members.pickFirst': ['Pick someone first', 'Choisissez d’abord quelqu’un', 'Elige a alguien primero', 'Wähle zuerst jemanden aus'],
	'members.directoryFailed': [
		'Could not load the user directory.',
		'Impossible de charger l’annuaire.',
		'No se pudo cargar el directorio de usuarios.',
		'Das Nutzerverzeichnis konnte nicht geladen werden.'
	],
	'members.badId': [
		'That account has an id this API will not accept.',
		'Ce compte a un identifiant que l’API n’accepte pas.',
		'Esa cuenta tiene un id que esta API no acepta.',
		'Dieses Konto hat eine ID, die diese API nicht akzeptiert.'
	],
	'members.noAccount': [
		'No account uses that email address yet.',
		'Aucun compte n’utilise encore cette adresse email.',
		'Ninguna cuenta usa todavía esa dirección de correo.',
		'Noch kein Konto nutzt diese E-Mail-Adresse.'
	],
	'members.added': ['Added {name}.', '{name} ajouté.', 'Se añadió a {name}.', '{name} hinzugefügt.'],
	'members.addFailed': [
		'Could not add that member.',
		'Impossible d’ajouter ce membre.',
		'No se pudo añadir a ese miembro.',
		'Dieses Mitglied konnte nicht hinzugefügt werden.'
	],
	'members.roleChanged': [
		'{name} is now {role}.',
		'{name} est maintenant {role}.',
		'{name} ahora es {role}.',
		'{name} ist jetzt {role}.'
	],
	'members.roleFailed': [
		'Could not change that role.',
		'Impossible de changer ce rôle.',
		'No se pudo cambiar ese rol.',
		'Diese Rolle konnte nicht geändert werden.'
	],
	'members.removed': ['Removed {name}.', '{name} retiré.', 'Se quitó a {name}.', '{name} entfernt.'],
	'members.removeFailed': [
		'Could not remove that member.',
		'Impossible de retirer ce membre.',
		'No se pudo quitar a ese miembro.',
		'Dieses Mitglied konnte nicht entfernt werden.'
	],
	'members.removeTitle': ['Remove this member?', 'Retirer ce membre ?', '¿Quitar a este miembro?', 'Dieses Mitglied entfernen?'],
	'members.removeText': [
		'{name} loses access to every file in this space immediately. Files they uploaded stay in the space.',
		'{name} perd aussitôt l’accès à tous les fichiers de cet espace. Les fichiers déposés restent dans l’espace.',
		'{name} pierde de inmediato el acceso a todos los archivos de este espacio. Los archivos que subió se quedan en el espacio.',
		'{name} verliert sofort den Zugriff auf alle Dateien in diesem Bereich. Hochgeladene Dateien bleiben im Bereich.'
	],
	'spaceSettings.title': [
		'Settings — {name} — Nuage',
		'Paramètres — {name} — Nuage',
		'Ajustes — {name} — Nuage',
		'Einstellungen — {name} — Nuage'
	],
	'spaceSettings.heading': ['Space settings', 'Paramètres de l’espace', 'Ajustes del espacio', 'Bereichseinstellungen'],
	'spaceSettings.general': ['General', 'Général', 'General', 'Allgemein'],
	'spaceSettings.generalText': [
		'How this space is labelled everywhere it appears.',
		'Comment cet espace est nommé partout où il apparaît.',
		'Cómo se etiqueta este espacio dondequiera que aparezca.',
		'Wie dieser Bereich überall bezeichnet wird, wo er erscheint.'
	],
	'spaceSettings.renameInRegistre': [
		'Renamed in Registre, for every Facile app at once.',
		'Se renomme dans Registre, pour toutes les apps Facile à la fois.',
		'Se renombra en Registre, para todas las apps de Facile a la vez.',
		'Wird in Registre umbenannt, für alle Facile-Apps gleichzeitig.'
	],
	'spaceSettings.namePlaceholder': ['Design team', 'Équipe design', 'Equipo de diseño', 'Design-Team'],
	'spaceSettings.descriptionPlaceholder': [
		'What belongs in here.',
		'Ce qui a sa place ici.',
		'Qué va aquí.',
		'Was hier hineingehört.'
	],
	'spaceSettings.save': ['Save changes', 'Enregistrer les modifications', 'Guardar cambios', 'Änderungen speichern'],
	'spaceSettings.updated': ['Space updated.', 'Espace mis à jour.', 'Espacio actualizado.', 'Bereich aktualisiert.'],
	'spaceSettings.saveFailed': [
		'Could not save that.',
		'Impossible d’enregistrer.',
		'No se pudo guardar.',
		'Das konnte nicht gespeichert werden.'
	],
	'spaceSettings.webdavText': [
		'Mount this space on its own in Finder or any WebDAV client.',
		'Montez cet espace séparément dans le Finder ou n’importe quel client WebDAV.',
		'Monta este espacio por separado en el Finder o en cualquier cliente WebDAV.',
		'Binde diesen Bereich einzeln im Finder oder in einem beliebigen WebDAV-Client ein.'
	],
	'spaceSettings.mountUrl': ['Mount URL', 'URL de montage', 'URL de montaje', 'Einbindungs-URL'],
	'spaceSettings.danger': ['Danger zone', 'Zone dangereuse', 'Zona de peligro', 'Gefahrenzone'],
	'spaceSettings.dangerText': [
		'Deleting the space removes every member. Its files and folders survive, but they stop being scoped to a space and only the uploader keeps them.',
		'Supprimer l’espace retire tous les membres. Ses fichiers et dossiers survivent, mais ne sont plus rattachés à un espace et seule la personne qui les a déposés les conserve.',
		'Eliminar el espacio quita a todos los miembros. Sus archivos y carpetas sobreviven, pero dejan de pertenecer a un espacio y solo los conserva quien los subió.',
		'Das Löschen des Bereichs entfernt alle Mitglieder. Seine Dateien und Ordner bleiben erhalten, gehören aber nicht mehr zu einem Bereich, und nur wer sie hochgeladen hat, behält sie.'
	],
	'spaceSettings.deleteThis': ['Delete this space', 'Supprimer cet espace', 'Eliminar este espacio', 'Diesen Bereich löschen'],
	'spaceSettings.deleteThisText': [
		'There is no undo, and the name is not reserved afterwards.',
		'Il n’y a pas d’annulation, et le nom n’est pas réservé ensuite.',
		'No se puede deshacer y el nombre no queda reservado después.',
		'Es gibt kein Rückgängig, und der Name wird danach nicht reserviert.'
	],
	'spaceSettings.delete': ['Delete space', 'Supprimer l’espace', 'Eliminar espacio', 'Bereich löschen'],
	'spaceSettings.deleteFailed': [
		'Could not delete that space.',
		'Impossible de supprimer cet espace.',
		'No se pudo eliminar ese espacio.',
		'Der Bereich konnte nicht gelöscht werden.'
	],
	'spaceSettings.deleted': ['Space deleted.', 'Espace supprimé.', 'Espacio eliminado.', 'Bereich gelöscht.'],
	'spaceSettings.deleteTitle': [
		'Delete “{name}”?',
		'Supprimer « {name} » ?',
		'¿Eliminar «{name}»?',
		'„{name}“ löschen?'
	],
	'spaceSettings.deleteText': [
		'Every member loses access at once. The files inside stop being shared and fall back to their uploader. This cannot be undone.',
		'Tous les membres perdent l’accès d’un coup. Les fichiers à l’intérieur ne sont plus partagés et reviennent à la personne qui les a déposés. C’est irréversible.',
		'Todos los miembros pierden el acceso a la vez. Los archivos del interior dejan de compartirse y vuelven a quien los subió. No se puede deshacer.',
		'Alle Mitglieder verlieren sofort den Zugriff. Die Dateien darin werden nicht mehr geteilt und fallen an die Person zurück, die sie hochgeladen hat. Das lässt sich nicht rückgängig machen.'
	]
} satisfies Record<string, Row>;
