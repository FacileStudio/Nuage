import type { Row } from '../row';

/** The page where the person confirms what an AI client asked Nuage to do. */
export const pending = {
	'pending.title': ['Pending actions — Nuage', 'Actions en attente — Nuage', 'Acciones pendientes — Nuage', 'Ausstehende Aktionen — Nuage'],
	'pending.heading': ['Pending actions', 'Actions en attente', 'Acciones pendientes', 'Ausstehende Aktionen'],
	'pending.intro': [
		'An AI client asked Nuage to do these. Nothing happens until you confirm.',
		'Un client IA a demandé ces actions à Nuage. Rien ne se passe tant que vous ne confirmez pas.',
		'Un cliente de IA pidió estas acciones a Nuage. No ocurre nada hasta que confirmes.',
		'Ein KI-Client hat Nuage um diese Aktionen gebeten. Nichts passiert, bevor Sie bestätigen.'
	],
	'pending.none': ['Nothing is waiting', 'Rien en attente', 'Nada pendiente', 'Nichts ausstehend'],
	'pending.noneBody': [
		'Actions an AI client asks for appear here until you confirm or dismiss them.',
		'Les actions demandées par un client IA apparaissent ici jusqu’à ce que vous les confirmiez ou les ignoriez.',
		'Las acciones que pide un cliente de IA aparecen aquí hasta que las confirmes o descartes.',
		'Aktionen, um die ein KI-Client bittet, erscheinen hier, bis Sie sie bestätigen oder verwerfen.'
	],
	'pending.kind.trash': ['Move to trash', 'Mettre à la corbeille', 'Mover a la papelera', 'In den Papierkorb'],
	'pending.kind.share': ['Public link', 'Lien public', 'Enlace público', 'Öffentlicher Link'],
	'pending.askedBy': ['Asked by {client}', 'Demandé par {client}', 'Pedido por {client}', 'Angefragt von {client}'],
	'pending.folderAll': [
		'The folder and everything in it',
		'Le dossier et tout son contenu',
		'La carpeta y todo su contenido',
		'Der Ordner mit seinem gesamten Inhalt'
	],
	'pending.shareWarn': [
		'Anyone with the link can open it.',
		'Toute personne ayant le lien pourra l’ouvrir.',
		'Cualquiera con el enlace podrá abrirlo.',
		'Jeder mit dem Link kann es öffnen.'
	],
	'pending.shareDays': [
		'The link works for {count} days.',
		'Le lien fonctionne pendant {count} jours.',
		'El enlace funciona durante {count} días.',
		'Der Link funktioniert {count} Tage lang.'
	],
	'pending.shareForever': [
		'The link does not expire.',
		'Le lien n’expire pas.',
		'El enlace no caduca.',
		'Der Link läuft nicht ab.'
	],
	'pending.trashNote': [
		'You can restore it from the trash.',
		'Vous pourrez le restaurer depuis la corbeille.',
		'Podrás restaurarlo desde la papelera.',
		'Sie können es aus dem Papierkorb wiederherstellen.'
	],
	'pending.confirmTrash': ['Move to trash', 'Mettre à la corbeille', 'Mover a la papelera', 'In den Papierkorb'],
	'pending.confirmShare': ['Create the link', 'Créer le lien', 'Crear el enlace', 'Link erstellen'],
	'pending.dismiss': ['Dismiss', 'Ignorer', 'Descartar', 'Verwerfen'],
	'pending.changed': [
		'This changed after the request. Ask the AI client again.',
		'Cet élément a changé après la demande. Redemandez au client IA.',
		'Esto cambió después de la solicitud. Vuelve a pedírselo al cliente de IA.',
		'Das wurde nach der Anfrage geändert. Fragen Sie den KI-Client erneut.'
	],
	'pending.missing': [
		'This file or folder no longer exists, or you can no longer reach it.',
		'Ce fichier ou dossier n’existe plus, ou vous n’y avez plus accès.',
		'Este archivo o carpeta ya no existe, o ya no tienes acceso.',
		'Diese Datei oder dieser Ordner existiert nicht mehr oder ist für Sie nicht mehr erreichbar.'
	],
	'pending.trashed': ['Moved to the trash', 'Mis à la corbeille', 'Movido a la papelera', 'In den Papierkorb verschoben'],
	'pending.shared': ['Link created', 'Lien créé', 'Enlace creado', 'Link erstellt'],
	'pending.newLink': ['Public link', 'Lien public', 'Enlace público', 'Öffentlicher Link'],
	'pending.dismissed': [
		'Dismissed. Nothing happened.',
		'Ignoré. Rien ne s’est passé.',
		'Descartado. No ha pasado nada.',
		'Verworfen. Nichts ist passiert.'
	],
	'pending.failed': ['That did not work', 'Cela n’a pas marché', 'No ha funcionado', 'Das hat nicht geklappt'],
	'pending.loadFailed': [
		'Could not load pending actions',
		'Impossible de charger les actions en attente',
		'No se pudieron cargar las acciones pendientes',
		'Ausstehende Aktionen konnten nicht geladen werden'
	],
	'pending.signInAgain': [
		'Sign in again to confirm',
		'Reconnectez-vous pour confirmer',
		'Vuelve a iniciar sesión para confirmar',
		'Melden Sie sich erneut an, um zu bestätigen'
	],
	'pending.signInAgainBody': [
		'Only a sign-in made in this browser can confirm what an AI client asked for. Sign in again here. The mobile and desktop apps cannot confirm these actions.',
		'Seule une connexion faite dans ce navigateur peut confirmer ce qu’un client IA a demandé. Reconnectez-vous ici. Les applications mobile et de bureau ne peuvent pas confirmer ces actions.',
		'Solo un inicio de sesión hecho en este navegador puede confirmar lo que pidió un cliente de IA. Vuelve a iniciar sesión aquí. Las aplicaciones móvil y de escritorio no pueden confirmar estas acciones.',
		'Nur eine Anmeldung in diesem Browser kann bestätigen, worum ein KI-Client gebeten hat. Melden Sie sich hier erneut an. Die Mobil- und Desktop-Apps können diese Aktionen nicht bestätigen.'
	],
	'pending.signIn': ['Sign in again', 'Se reconnecter', 'Iniciar sesión de nuevo', 'Erneut anmelden']
} satisfies Record<string, Row>;
