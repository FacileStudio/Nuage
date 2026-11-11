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
		'Nuage is a self-hosted file storage platform. Upload, organize into folders, share with a link — done.',
		'Nuage est une plateforme de stockage de fichiers auto-hébergée. Déposez, rangez dans des dossiers, partagez par lien : c’est tout.',
		'Nuage es una plataforma de almacenamiento de archivos autoalojada. Sube, organiza en carpetas y comparte con un enlace: así de simple.',
		'Nuage ist eine selbst gehostete Plattform für Dateispeicher. Hochladen, in Ordnern ordnen, per Link teilen: fertig.'
	],
	'landing.feature.upload.title': ['Upload anything', 'Déposez tout', 'Sube lo que quieras', 'Alles hochladen'],
	'landing.feature.upload.text': [
		'Drag and drop files or click to upload. Any file type, any size.',
		'Glissez-déposez vos fichiers ou cliquez pour les envoyer. Tous les types, toutes les tailles.',
		'Arrastra y suelta archivos o haz clic para subirlos. Cualquier tipo, cualquier tamaño.',
		'Dateien per Drag-and-drop oder Klick hochladen. Jeder Dateityp, jede Größe.'
	],
	'landing.feature.folders.title': [
		'Organize with folders',
		'Rangez avec des dossiers',
		'Organiza con carpetas',
		'Mit Ordnern ordnen'
	],
	'landing.feature.folders.text': [
		'Create nested folders, move files around, keep everything tidy.',
		'Créez des dossiers imbriqués, déplacez vos fichiers, gardez tout en ordre.',
		'Crea carpetas anidadas, mueve archivos y mantén todo en orden.',
		'Lege verschachtelte Ordner an, verschiebe Dateien und halte alles aufgeräumt.'
	],
	'landing.feature.hosted.title': ['Self-hosted', 'Auto-hébergé', 'Autoalojado', 'Selbst gehostet'],
	'landing.feature.hosted.text': [
		'Your files stay on your server. No third-party cloud, no data harvesting.',
		'Vos fichiers restent sur votre serveur. Pas de cloud tiers, pas de collecte de données.',
		'Tus archivos se quedan en tu servidor. Sin nube de terceros ni recopilación de datos.',
		'Deine Dateien bleiben auf deinem Server. Keine Fremd-Cloud, kein Datensammeln.'
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
