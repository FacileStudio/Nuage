import type { Row } from '../row';

/** The settings page: profile, appearance, API tokens, app data, Nook and the instance. */
export const settings = {
	'settings.title': ['Settings — Nuage', 'Paramètres — Nuage', 'Ajustes — Nuage', 'Einstellungen — Nuage'],
	'settings.heading': ['Settings', 'Paramètres', 'Ajustes', 'Einstellungen'],
	'settings.sections': ['Settings sections', 'Sections des paramètres', 'Secciones de ajustes', 'Einstellungsbereiche'],
	'settings.tab.profile': ['Profile', 'Profil', 'Perfil', 'Profil'],
	'settings.tab.appearance': ['Appearance', 'Apparence', 'Apariencia', 'Darstellung'],
	'settings.tab.api': ['API', 'API', 'API', 'API'],
	'settings.tab.apps': ['App data', 'Données des apps', 'Datos de apps', 'App-Daten'],
	'settings.tab.nook': ['Nook', 'Nook', 'Nook', 'Nook'],
	'settings.tab.advanced': ['Advanced', 'Avancé', 'Avanzado', 'Erweitert'],
	'settings.picture.title': ['Picture', 'Photo', 'Imagen', 'Bild'],
	'settings.picture.description': [
		'Shown next to your name across the app.',
		'Affichée à côté de votre nom dans toute l’application.',
		'Se muestra junto a tu nombre en toda la aplicación.',
		'Wird in der ganzen App neben deinem Namen angezeigt.'
	],
	'settings.picture.source': [
		'Your picture comes from your Facile account.',
		'Votre photo vient de votre compte Facile.',
		'Tu imagen proviene de tu cuenta de Facile.',
		'Dein Bild stammt aus deinem Facile-Konto.'
	],
	'settings.picture.hint': [
		'Change it there and it shows here on your next page load.',
		'Modifiez-la là-bas, elle apparaîtra ici au prochain chargement de la page.',
		'Cámbiala allí y aparecerá aquí al recargar la página.',
		'Ändere es dort, und es erscheint hier beim nächsten Laden der Seite.'
	],
	'settings.picture.change': ['Change picture', 'Changer la photo', 'Cambiar la imagen', 'Bild ändern'],
	'settings.identity.title': ['Identity', 'Identité', 'Identidad', 'Identität'],
	'settings.identity.description': [
		'How you appear to everyone sharing a space with you.',
		'La façon dont vous apparaissez à toutes les personnes qui partagent un espace avec vous.',
		'Cómo apareces ante quienes comparten un espacio contigo.',
		'Wie du für alle erscheinst, die einen Bereich mit dir teilen.'
	],
	'settings.displayName': ['Display name', 'Nom affiché', 'Nombre visible', 'Anzeigename'],
	'settings.namePlaceholder': ['Your name', 'Votre nom', 'Tu nombre', 'Dein Name'],
	'settings.emailManaged': [
		'Managed by your Facile account. Change it there; it updates here at your next sign-in.',
		'Géré par votre compte Facile. Modifiez-le là-bas : il se met à jour ici à votre prochaine connexion.',
		'Gestionado por tu cuenta de Facile. Cámbialo allí; se actualiza aquí en tu próximo inicio de sesión.',
		'Wird von deinem Facile-Konto verwaltet. Ändere sie dort; hier aktualisiert sie sich bei der nächsten Anmeldung.'
	],
	'settings.saveProfile': ['Save profile', 'Enregistrer le profil', 'Guardar perfil', 'Profil speichern'],
	'settings.profileUpdated': ['Profile updated.', 'Profil mis à jour.', 'Perfil actualizado.', 'Profil aktualisiert.'],
	'settings.profileFailed': [
		'Could not update your profile.',
		'Impossible de mettre à jour votre profil.',
		'No se pudo actualizar tu perfil.',
		'Dein Profil konnte nicht aktualisiert werden.'
	],
	'settings.session.title': ['Session', 'Session', 'Sesión', 'Sitzung'],
	'settings.session.description': [
		'This browser only — your files stay where they are.',
		'Ce navigateur uniquement : vos fichiers restent où ils sont.',
		'Solo este navegador: tus archivos se quedan donde están.',
		'Nur dieser Browser: Deine Dateien bleiben, wo sie sind.'
	],
	'settings.logout': ['Log out', 'Se déconnecter', 'Cerrar sesión', 'Abmelden'],
	'settings.logout.description': [
		'You will need to sign in again to reach your files.',
		'Vous devrez vous reconnecter pour accéder à vos fichiers.',
		'Tendrás que volver a iniciar sesión para acceder a tus archivos.',
		'Du musst dich erneut anmelden, um auf deine Dateien zuzugreifen.'
	],
	'settings.logout.confirmTitle': [
		'Log out of Nuage?',
		'Se déconnecter de Nuage ?',
		'¿Cerrar sesión en Nuage?',
		'Von Nuage abmelden?'
	],
	'settings.logout.confirmText': [
		'Your files stay where they are. You will need to sign in again on this browser.',
		'Vos fichiers restent où ils sont. Vous devrez vous reconnecter sur ce navigateur.',
		'Tus archivos se quedan donde están. Tendrás que volver a iniciar sesión en este navegador.',
		'Deine Dateien bleiben, wo sie sind. Du musst dich in diesem Browser erneut anmelden.'
	],
	'settings.theme.title': ['Theme', 'Thème', 'Tema', 'Design'],
	'settings.theme.description': [
		'Applied to this browser.',
		'Appliqué à ce navigateur.',
		'Se aplica a este navegador.',
		'Gilt für diesen Browser.'
	],
	'settings.theme.scheme': ['Colour scheme', 'Schéma de couleurs', 'Esquema de color', 'Farbschema'],
	'settings.theme.schemeHint': [
		'Follow the system setting, or pin one.',
		'Suivre le réglage du système, ou en imposer un.',
		'Seguir el ajuste del sistema o fijar uno.',
		'Der Systemeinstellung folgen oder eines festlegen.'
	],
	'settings.theme.system': ['System', 'Système', 'Sistema', 'System'],
	'settings.theme.light': ['Light', 'Clair', 'Claro', 'Hell'],
	'settings.theme.dark': ['Dark', 'Sombre', 'Oscuro', 'Dunkel']
} satisfies Record<string, Row>;
