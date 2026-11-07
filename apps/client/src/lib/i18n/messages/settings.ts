import type { Row } from '../row';

/** The settings page: profile, appearance, API tokens, app data, Nook and the instance. */
export const settings = {
	'settings.title': ['Settings — Nuage', 'Paramètres — Nuage', 'Ajustes — Nuage', 'Einstellungen — Nuage'],
	'settings.heading': ['Settings', 'Paramètres', 'Ajustes', 'Einstellungen'],
	'settings.sections': ['Settings sections', 'Sections des paramètres', 'Secciones de ajustes', 'Einstellungsbereiche'],
	'settings.tab.profile': ['Profile', 'Profil', 'Perfil', 'Profil'],
	'settings.tab.preferences': ['Preferences', 'Préférences', 'Preferencias', 'Einstellungen'],
	'settings.tab.tokens': ['API tokens', 'Jetons d’API', 'Tokens de API', 'API-Tokens'],
	'settings.tab.subscription': ['Subscription', 'Abonnement', 'Suscripción', 'Abonnement'],
	'settings.subscription.plan.plan': ['Facile plan', 'Forfait Facile', 'Plan de Facile', 'Facile-Tarif'],
	'settings.subscription.plan.override': ['Custom limit', 'Limite personnalisée', 'Límite personalizada', 'Individuelles Limit'],
	'settings.subscription.plan.instance': [
		'Instance default',
		'Défaut de l’instance',
		'Predeterminado de la instancia',
		'Standard der Instanz'
	],
	'settings.subscription.storage': ['Storage (GB)', 'Stockage (Go)', 'Almacenamiento (GB)', 'Speicher (GB)'],
	'settings.subscription.storageLabel': ['Storage', 'Stockage', 'Almacenamiento', 'Speicher'],
	'settings.tab.apps': ['App data', 'Données des apps', 'Datos de apps', 'App-Daten'],
	'settings.tab.nook': ['Nook', 'Nook', 'Nook', 'Nook'],
	'settings.tab.advanced': ['Advanced', 'Avancé', 'Avanzado', 'Erweitert']
} satisfies Record<string, Row>;
