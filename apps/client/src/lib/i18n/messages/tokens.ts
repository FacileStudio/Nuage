import type { Row } from '../row';

/** Settings: API tokens, WebDAV, Nook and the instance. */
export const tokens = {
	'settings.tokens.actions': ['Actions', 'Actions', 'Acciones', 'Aktionen'],
	'settings.tokens.revoked': [
		'Revoked “{name}”.',
		'« {name} » révoqué.',
		'«{name}» revocado.',
		'„{name}“ widerrufen.'
	],
	'settings.webdav.title': ['WebDAV', 'WebDAV', 'WebDAV', 'WebDAV'],
	'settings.webdav.description': [
		'Mount Nuage in Finder or any WebDAV client.',
		'Montez Nuage dans le Finder ou dans n’importe quel client WebDAV.',
		'Monta Nuage en el Finder o en cualquier cliente WebDAV.',
		'Binde Nuage im Finder oder in einem beliebigen WebDAV-Client ein.'
	],
	'settings.webdav.personal': ['Personal', 'Personnel', 'Personal', 'Persönlich'],
	'settings.webdav.personalText': [
		'Sign in with your email and an API token as the password.',
		'Connectez-vous avec votre email et un jeton d’API comme mot de passe.',
		'Inicia sesión con tu correo y un token de API como contraseña.',
		'Melde dich mit deiner E-Mail und einem API-Token als Passwort an.'
	],
	'settings.webdav.space': [
		'Mount this space on its own.',
		'Montez cet espace séparément.',
		'Monta este espacio por separado.',
		'Binde diesen Bereich einzeln ein.'
	],
	'settings.webdav.index': ['Space index', 'Index des espaces', 'Índice de espacios', 'Bereichsübersicht'],
	'settings.webdav.indexText': [
		'Read-only. Lists the spaces you can mount.',
		'Lecture seule. Liste les espaces que vous pouvez monter.',
		'Solo lectura. Lista los espacios que puedes montar.',
		'Nur lesen. Listet die Bereiche auf, die du einbinden kannst.'
	],
	'settings.nook.description': [
		'Nuage posts file events to a Nook webhook so alerts land where the rest of the suite reports.',
		'Nuage envoie les événements de fichiers à un webhook Nook pour que les alertes arrivent là où le reste de la suite les signale.',
		'Nuage envía los eventos de archivos a un webhook de Nook para que las alertas lleguen donde informa el resto de la suite.',
		'Nuage sendet Dateiereignisse an einen Nook-Webhook, damit Warnungen dort ankommen, wo der Rest der Suite meldet.'
	],
	'settings.nook.status': ['Status', 'État', 'Estado', 'Status'],
	'settings.nook.disabled': ['Disabled', 'Désactivé', 'Desactivado', 'Deaktiviert'],
	'settings.nook.testing': [
		'Testing the connection…',
		'Test de la connexion…',
		'Probando la conexión…',
		'Verbindung wird getestet…'
	],
	'settings.nook.noUrl': [
		'Enabled, but no URL is set',
		'Activé, mais aucune URL n’est définie',
		'Activado, pero no hay ninguna URL',
		'Aktiviert, aber keine URL festgelegt'
	],
	'settings.nook.reached': ['Reached the endpoint', 'Point d’accès joint', 'Se alcanzó el endpoint', 'Endpunkt erreicht'],
	'settings.nook.noAnswer': [
		'The endpoint did not answer',
		'Le point d’accès n’a pas répondu',
		'El endpoint no respondió',
		'Der Endpunkt hat nicht geantwortet'
	],
	'settings.nook.untested': [
		'Enabled — not tested since load',
		'Activé, non testé depuis le chargement',
		'Activado, sin probar desde la carga',
		'Aktiviert, seit dem Laden nicht getestet'
	],
	'settings.nook.enabled': ['Enabled', 'Activé', 'Activado', 'Aktiviert'],
	'settings.nook.enabledText': [
		'Stops every outbound event when off.',
		'Arrête tous les événements sortants lorsqu’il est désactivé.',
		'Detiene todos los eventos salientes cuando está desactivado.',
		'Stoppt alle ausgehenden Ereignisse, wenn deaktiviert.'
	],
	'settings.nook.switchLabel': [
		'Nook integration enabled',
		'Intégration Nook activée',
		'Integración con Nook activada',
		'Nook-Integration aktiviert'
	],
	'settings.nook.url': ['Webhook URL', 'URL du webhook', 'URL del webhook', 'Webhook-URL'],
	'settings.nook.secret': ['Shared secret', 'Secret partagé', 'Secreto compartido', 'Gemeinsames Geheimnis'],
	'settings.nook.secretText': [
		'Signs every payload so Nook can tell it came from here.',
		'Signe chaque envoi pour que Nook sache qu’il vient d’ici.',
		'Firma cada envío para que Nook sepa que viene de aquí.',
		'Signiert jede Nachricht, damit Nook erkennt, dass sie von hier stammt.'
	],
	'settings.nook.notSet': ['Not set', 'Non défini', 'Sin definir', 'Nicht festgelegt'],
	'settings.nook.test': ['Test connection', 'Tester la connexion', 'Probar la conexión', 'Verbindung testen'],
	'settings.nook.failed': ['Connection failed.', 'Échec de la connexion.', 'Falló la conexión.', 'Verbindung fehlgeschlagen.'],
	'settings.instance.title': ['Instance', 'Instance', 'Instancia', 'Instanz'],
	'settings.instance.description': [
		'Applies to everyone on this Nuage.',
		'S’applique à toutes les personnes de ce Nuage.',
		'Se aplica a todas las personas de este Nuage.',
		'Gilt für alle Personen auf diesem Nuage.'
	],
	'settings.instance.name': ['Instance name', 'Nom de l’instance', 'Nombre de la instancia', 'Name der Instanz'],
	'settings.retention': ['Trash retention', 'Durée de conservation de la corbeille', 'Retención de la papelera', 'Papierkorb-Aufbewahrung'],
	'settings.retention.text': [
		'Days a deleted item stays in the trash before it is removed for good and its storage is given back. 0 keeps items until someone empties the trash.',
		'Nombre de jours pendant lesquels un élément supprimé reste dans la corbeille avant d’être effacé définitivement et que son espace soit libéré. 0 le conserve jusqu’à ce que quelqu’un vide la corbeille.',
		'Días que un elemento eliminado permanece en la papelera antes de borrarse definitivamente y liberar su espacio. 0 lo conserva hasta que alguien vacíe la papelera.',
		'Tage, die ein gelöschtes Element im Papierkorb bleibt, bevor es endgültig entfernt und sein Speicher freigegeben wird. 0 behält Elemente, bis jemand den Papierkorb leert.'
	],
	'settings.retention.label': [
		'Trash retention in days',
		'Durée de conservation de la corbeille en jours',
		'Retención de la papelera en días',
		'Papierkorb-Aufbewahrung in Tagen'
	],
	'settings.saved': ['Settings saved.', 'Paramètres enregistrés.', 'Ajustes guardados.', 'Einstellungen gespeichert.'],
	'settings.saveFailed': [
		'Could not save settings.',
		'Impossible d’enregistrer les paramètres.',
		'No se pudieron guardar los ajustes.',
		'Die Einstellungen konnten nicht gespeichert werden.'
	],
	'settings.loadFailed': [
		'Could not load instance settings.',
		'Impossible de charger les paramètres de l’instance.',
		'No se pudieron cargar los ajustes de la instancia.',
		'Die Instanzeinstellungen konnten nicht geladen werden.'
	],
	'settings.about.title': ['About', 'À propos', 'Acerca de', 'Info'],
	'settings.about.description': [
		'Facts worth having on hand when something breaks.',
		'Les informations utiles quand quelque chose ne va pas.',
		'Datos útiles cuando algo falla.',
		'Angaben, die hilfreich sind, wenn etwas kaputtgeht.'
	],
	'settings.about.docs': ['API documentation', 'Documentation de l’API', 'Documentación de la API', 'API-Dokumentation'],
	'settings.about.docsText': [
		'Every endpoint this instance serves.',
		'Tous les points d’accès de cette instance.',
		'Todos los endpoints que sirve esta instancia.',
		'Alle Endpunkte, die diese Instanz bereitstellt.'
	],
	'settings.about.open': ['Open', 'Ouvrir', 'Abrir', 'Öffnen'],
	'settings.about.created': ['Account created', 'Compte créé le', 'Cuenta creada', 'Konto erstellt'],
	'settings.instancePlaceholder': ['My Nuage', 'Mon Nuage', 'Mi Nuage', 'Mein Nuage']
} satisfies Record<string, Row>;
