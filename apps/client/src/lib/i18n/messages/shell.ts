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
		'Your files everywhere, shared with a link.',
		'Vos fichiers partout, partagés en un lien.',
		'Tus archivos en todas partes, compartidos con un enlace.',
		'Ihre Dateien überall, geteilt mit einem Link.'
	],
	'login.failed': [
		'Sign-in failed. Try again.',
		'La connexion a échoué. Réessayez.',
		'No se pudo iniciar sesión. Inténtalo de nuevo.',
		'Die Anmeldung ist fehlgeschlagen. Versuche es erneut.'
	],
	'landing.heroTitle': [
		'Your files, everywhere.|Shared in one link.',
		'Vos fichiers, partout.|Partagés en un lien.',
		'Tus archivos, en todas partes.|Compartidos con un enlace.',
		'Ihre Dateien, überall.|Geteilt mit einem Link.'
	],
	'landing.heroText': [
		'Keep your photos and documents, share them by link and work together in team folders.',
		'Rangez vos photos et documents, partagez-les par lien et travaillez à plusieurs dans des dossiers d’équipe.',
		'Guarda tus fotos y documentos, compártelos con un enlace y trabaja en equipo en carpetas compartidas.',
		'Bewahren Sie Fotos und Dokumente auf, teilen Sie sie per Link und arbeiten Sie gemeinsam in Teamordnern.'
	],
	'landing.navFeatures': ['Features', 'Fonctionnalités', 'Funciones', 'Funktionen'],
	'landing.navDownload': ['Download', 'Télécharger', 'Descargar', 'Download'],
	'landing.demoLabel': [
		'Holiday photos dropped into a folder upload to Nuage, then the album gets shared with a link.',
		'Des photos de vacances glissées dans un dossier arrivent dans Nuage, puis l’album part en lien de partage.',
		'Unas fotos de vacaciones arrastradas a una carpeta se suben a Nuage y el álbum se comparte con un enlace.',
		'Urlaubsfotos landen per Drag-and-drop in Nuage, dann wird das Album per Link geteilt.'
	],
	'landing.featuresTitle': ['What Nuage does', 'Ce que fait Nuage', 'Qué hace Nuage', 'Was Nuage kann'],
	'landing.featuresLine': [
		'Everything you need to keep and share files, in your browser.',
		'Tout pour ranger et partager vos fichiers, dans votre navigateur.',
		'Todo para guardar y compartir tus archivos, en tu navegador.',
		'Alles, um Dateien aufzubewahren und zu teilen, im Browser.'
	],
	'landing.feature.view.title': [
		'See it before you download',
		'Voyez avant de télécharger',
		'Míralo antes de descargar',
		'Ansehen vor dem Download'
	],
	'landing.feature.view.text': [
		'Photos, videos and PDFs open right in the browser.',
		'Photos, vidéos et PDF s’ouvrent directement dans le navigateur.',
		'Fotos, vídeos y PDF se abren directamente en el navegador.',
		'Fotos, Videos und PDFs öffnen sich direkt im Browser.'
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
		'Shared folders with members and roles.',
		'Des dossiers partagés, avec membres et rôles.',
		'Carpetas compartidas con miembros y roles.',
		'Geteilte Ordner mit Mitgliedern und Rollen.'
	],
	'landing.feature.trash.title': [
		'Nothing lost by mistake',
		'Rien de perdu par erreur',
		'Nada se pierde por error',
		'Nichts geht aus Versehen verloren'
	],
	'landing.feature.trash.text': [
		'Deleted files wait in the trash, ready to restore.',
		'Les fichiers supprimés attendent dans la corbeille, prêts à revenir.',
		'Los archivos eliminados esperan en la papelera, listos para restaurar.',
		'Gelöschte Dateien warten im Papierkorb auf ihre Wiederherstellung.'
	],
	'landing.downloadText': [
		'One binary for macOS and Linux. Run nuage login, then nuage start.',
		'Un seul binaire pour macOS et Linux. Lancez nuage login, puis nuage start.',
		'Un solo binario para macOS y Linux. Ejecuta nuage login y luego nuage start.',
		'Ein Binary für macOS und Linux. Führe nuage login aus, dann nuage start.'
	],
	'downloads.title': ['Download · Nuage', 'Télécharger · Nuage', 'Descargar · Nuage', 'Download · Nuage'],
	'downloads.description': [
		'Install the nuage command-line sync client on macOS or Linux.',
		'Installez le client de synchro nuage en ligne de commande sur macOS ou Linux.',
		'Instala el cliente de sincronización nuage en línea de comandos en macOS o Linux.',
		'Installieren Sie den nuage-Sync-Client für die Kommandozeile unter macOS oder Linux.'
	],
	'downloads.heading': ['Download Nuage', 'Télécharger Nuage', 'Descargar Nuage', 'Nuage herunterladen'],
	'landing.pricing': ['Pricing', 'Tarifs', 'Precios', 'Preise'],
	'landing.ctaTitle': ['Ready to start?', 'Prêt à commencer ?', '¿Listo para empezar?', 'Bereit loszulegen?'],
	'landing.ctaText': [
		'Create a free account and get 5 GB to start.',
		'Créez un compte gratuit et commencez avec 5 Go.',
		'Crea una cuenta gratuita y empieza con 5 GB.',
		'Erstellen Sie ein kostenloses Konto und starten Sie mit 5 GB.'
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
