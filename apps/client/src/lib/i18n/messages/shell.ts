import type { Row } from '../row';

/** Strings shared by the whole app: buttons, the sign-in and landing pages, the shell. */
export const shell = {
	'common.cancel': ['Cancel', 'Annuler', 'Cancelar', 'Abbrechen'],
	'common.save': ['Save', 'Enregistrer', 'Guardar', 'Speichern'],
	'common.saving': ['Saving…', 'Enregistrement…', 'Guardando…', 'Wird gespeichert…'],
	'common.create': ['Create', 'Créer', 'Crear', 'Erstellen'],
	'common.creating': ['Creating…', 'Création…', 'Creando…', 'Wird erstellt…'],
	'common.delete': ['Delete', 'Supprimer', 'Eliminar', 'Löschen'],
	'common.deleting': ['Deleting…', 'Suppression…', 'Eliminando…', 'Wird gelöscht…'],
	'common.remove': ['Remove', 'Retirer', 'Quitar', 'Entfernen'],
	'common.close': ['Close', 'Fermer', 'Cerrar', 'Schließen'],
	'common.revoke': ['Revoke', 'Révoquer', 'Revocar', 'Widerrufen'],
	'common.dismiss': ['Dismiss', 'Ignorer', 'Descartar', 'Ausblenden'],
	'common.done': ['Done', 'Terminé', 'Hecho', 'Fertig'],
	'common.retry': ['Retry', 'Réessayer', 'Reintentar', 'Erneut versuchen'],
	'common.loading': ['Loading', 'Chargement', 'Cargando', 'Wird geladen'],
	'common.name': ['Name', 'Nom', 'Nombre', 'Name'],
	'common.email': ['Email', 'Email', 'Correo electrónico', 'E-Mail'],
	'common.description': ['Description', 'Description', 'Descripción', 'Beschreibung'],
	'common.color': ['Color', 'Couleur', 'Color', 'Farbe'],
	'common.members': ['Members', 'Membres', 'Miembros', 'Mitglieder'],
	'common.size': ['Size', 'Taille', 'Tamaño', 'Größe'],
	'common.modified': ['Modified', 'Modifié', 'Modificado', 'Geändert'],
	'common.you': ['You', 'Vous', 'Tú', 'Du'],
	'common.unknownUser': ['Unknown user', 'Utilisateur inconnu', 'Usuario desconocido', 'Unbekannter Nutzer'],
	'common.deletedUser': ['Deleted user', 'Utilisateur supprimé', 'Usuario eliminado', 'Gelöschter Nutzer'],
	'common.unreachable': [
		'Could not reach Nuage.',
		'Nuage est injoignable.',
		'No se pudo conectar con Nuage.',
		'Nuage ist nicht erreichbar.'
	],
	'common.somethingWrong': [
		'Something went wrong.',
		'Une erreur est survenue.',
		'Se ha producido un error.',
		'Es ist ein Fehler aufgetreten.'
	],
	'nav.files': ['Files', 'Fichiers', 'Archivos', 'Dateien'],
	'nav.shared': ['Shared links', 'Liens partagés', 'Enlaces compartidos', 'Geteilte Links'],
	'nav.sharedShort': ['Shared', 'Partagés', 'Compartidos', 'Geteilt'],
	'nav.trash': ['Trash', 'Corbeille', 'Papelera', 'Papierkorb'],
	'nav.activity': ['Activity', 'Activité', 'Actividad', 'Aktivität'],
	'nav.spaces': ['Spaces', 'Espaces', 'Espacios', 'Bereiche'],
	'login.title': ['Sign in — Nuage', 'Connexion — Nuage', 'Iniciar sesión — Nuage', 'Anmelden — Nuage'],
	'login.tagline': [
		'Your files. Your cloud.',
		'Vos fichiers. Votre cloud.',
		'Tus archivos. Tu nube.',
		'Deine Dateien. Deine Cloud.'
	],
	'login.failed': [
		'Sign-in failed. Try again.',
		'La connexion a échoué. Réessayez.',
		'No se pudo iniciar sesión. Inténtalo de nuevo.',
		'Die Anmeldung ist fehlgeschlagen. Versuche es erneut.'
	],
	'landing.heroTitle': [
		'Your files.|Your cloud.',
		'Vos fichiers.|Votre cloud.',
		'Tus archivos.|Tu nube.',
		'Deine Dateien.|Deine Cloud.'
	],
	'landing.heroText': [
		'Store your files, sync them with your computer, share them with a link.',
		'Stockez vos fichiers, synchronisez-les avec votre ordinateur, partagez-les par lien.',
		'Guarda tus archivos, sincronízalos con tu ordenador y compártelos con un enlace.',
		'Speichere deine Dateien, synchronisiere sie mit deinem Computer, teile sie per Link.'
	],
	'landing.navFeatures': ['Features', 'Fonctionnalités', 'Funciones', 'Funktionen'],
	'landing.navDownload': ['Download', 'Télécharger', 'Descargar', 'Download'],
	'landing.downloadCta': [
		'Get the sync client',
		'Installer le client de synchro',
		'Instala el cliente de sincronización',
		'Sync-Client installieren'
	],
	'landing.demoLabel': [
		'A file saved on the computer syncs to Nuage, then gets shared with a link.',
		'Un fichier enregistré sur l’ordinateur arrive dans Nuage, puis part en lien de partage.',
		'Un archivo guardado en el ordenador se sincroniza con Nuage y se comparte con un enlace.',
		'Eine auf dem Computer gespeicherte Datei landet in Nuage und wird per Link geteilt.'
	],
	'landing.featuresTitle': ['What Nuage does', 'Ce que fait Nuage', 'Qué hace Nuage', 'Was Nuage kann'],
	'landing.feature.sync.title': [
		'Sync a folder',
		'Synchronisez un dossier',
		'Sincroniza una carpeta',
		'Einen Ordner synchronisieren'
	],
	'landing.feature.sync.text': [
		'The nuage daemon keeps a folder on your computer in sync with each space.',
		'Le démon nuage garde un dossier de votre ordinateur synchronisé avec chaque espace.',
		'El demonio nuage mantiene una carpeta de tu ordenador sincronizada con cada espacio.',
		'Der nuage-Daemon hält einen Ordner auf deinem Computer mit jedem Bereich synchron.'
	],
	'landing.feature.share.title': ['Share with a link', 'Partagez par lien', 'Comparte con un enlace', 'Per Link teilen'],
	'landing.feature.share.text': [
		'Links for viewing or editing, with an optional expiry date.',
		'Des liens en lecture ou en écriture, avec une date d’expiration si besoin.',
		'Enlaces de lectura o edición, con fecha de caducidad opcional.',
		'Links zum Ansehen oder Bearbeiten, auf Wunsch mit Ablaufdatum.'
	],
	'landing.feature.spaces.title': [
		'Spaces for your team',
		'Des espaces d’équipe',
		'Espacios para tu equipo',
		'Bereiche fürs Team'
	],
	'landing.feature.spaces.text': [
		'Shared drives with members and roles.',
		'Des disques partagés, avec membres et rôles.',
		'Unidades compartidas con miembros y roles.',
		'Geteilte Laufwerke mit Mitgliedern und Rollen.'
	],
	'landing.feature.mount.title': ['Open it anywhere', 'Ouvrez-le partout', 'Ábrelo donde quieras', 'Überall öffnen'],
	'landing.feature.mount.text': [
		'Mount your drive over WebDAV, or reach it from any MCP client.',
		'Montez votre disque en WebDAV, ou ouvrez-le depuis un client MCP.',
		'Monta tu unidad por WebDAV o ábrela desde cualquier cliente MCP.',
		'Binde dein Laufwerk per WebDAV ein oder öffne es in einem MCP-Client.'
	],
	'landing.downloadTitle': [
		'Sync from your computer',
		'Synchronisez depuis votre ordinateur',
		'Sincroniza desde tu ordenador',
		'Vom Computer synchronisieren'
	],
	'landing.downloadText': [
		'One binary for macOS and Linux. Run nuage login, then nuage start.',
		'Un seul binaire pour macOS et Linux. Lancez nuage login, puis nuage start.',
		'Un solo binario para macOS y Linux. Ejecuta nuage login y luego nuage start.',
		'Ein Binary für macOS und Linux. Führe nuage login aus, dann nuage start.'
	],
	'landing.pricing': ['Pricing', 'Tarifs', 'Precios', 'Preise'],
	'landing.ctaTitle': ['Ready to start?', 'Prêt à commencer ?', '¿Listo para empezar?', 'Bereit loszulegen?'],
	'landing.ctaText': [
		'Sign in with your Facile account to open your cloud.',
		'Connectez-vous avec votre compte Facile pour ouvrir votre cloud.',
		'Inicia sesión con tu cuenta de Facile para abrir tu nube.',
		'Melde dich mit deinem Facile-Konto an, um deine Cloud zu öffnen.'
	],
	'quota.group': ['Storage usage', 'Utilisation du stockage', 'Uso del almacenamiento', 'Speichernutzung'],
	'quota.of': [
		'{used} of {limit}',
		'{used} sur {limit}',
		'{used} de {limit}',
		'{used} von {limit}'
	],
	'quota.ofPlan': [
		'{used} of {limit} on your plan',
		'{used} sur {limit} avec votre forfait',
		'{used} de {limit} en tu plan',
		'{used} von {limit} in deinem Tarif'
	],
	'quota.usedLabel': ['Storage used', 'Stockage utilisé', 'Almacenamiento usado', 'Genutzter Speicher'],
	'quota.used': ['{used} used', '{used} utilisés', '{used} usados', '{used} belegt'],
	'quota.upgrade': ['Upgrade plan', 'Changer de forfait', 'Mejorar el plan', 'Tarif upgraden'],
	'plan.full': ['Your plan is full', 'Votre forfait est plein', 'Tu plan está lleno', 'Dein Tarif ist voll'],
	'plan.noRoom': [
		'Your plan has no room for this.',
		'Votre forfait n’a plus de place pour cela.',
		'Tu plan no tiene espacio para esto.',
		'Dein Tarif bietet dafür keinen Platz mehr.'
	],
	'plan.stillWorks': [
		'Reading, downloading and deleting still work.',
		'La lecture, le téléchargement et la suppression restent possibles.',
		'Leer, descargar y eliminar siguen funcionando.',
		'Lesen, Herunterladen und Löschen funktionieren weiterhin.'
	]
} satisfies Record<string, Row>;
