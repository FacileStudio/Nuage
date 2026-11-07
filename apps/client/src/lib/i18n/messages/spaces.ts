import type { Row } from '../row';

/** The spaces list, a space's page, its members and its settings. */
export const spaces = {
	'role.owner': ['Owner', 'Propriétaire', 'Propietario', 'Eigentümer'],
	'role.admin': ['Admin', 'Administrateur', 'Administrador', 'Administrator'],
	'role.member': ['Member', 'Membre', 'Miembro', 'Mitglied'],
	'spaces.title': ['Spaces — Nuage', 'Espaces — Nuage', 'Espacios — Nuage', 'Bereiche — Nuage'],
	'spaces.heading': ['Spaces', 'Espaces', 'Espacios', 'Bereiche'],
	'spaces.description': [
		'Shared storage for a team. Files in a space belong to the space, not to you.',
		'Stockage partagé pour une équipe. Les fichiers d’un espace appartiennent à l’espace, pas à vous.',
		'Almacenamiento compartido para un equipo. Los archivos de un espacio pertenecen al espacio, no a ti.',
		'Gemeinsamer Speicher für ein Team. Dateien in einem Bereich gehören dem Bereich, nicht dir.'
	],
	'spaces.manageInRegistre': ['Manage in Registre', 'Gérer dans Registre', 'Gestionar en Registre', 'In Registre verwalten'],
	'spaces.emptyTitle': ['No spaces yet', 'Aucun espace', 'Aún no hay espacios', 'Noch keine Bereiche'],
	'spaces.emptyText': [
		'Spaces are created in Registre and show up here with their members, in every Facile app.',
		'Les espaces se créent dans Registre et apparaissent ici avec leurs membres, dans toutes les apps Facile.',
		'Los espacios se crean en Registre y aparecen aquí con sus miembros, en todas las apps de Facile.',
		'Bereiche werden in Registre erstellt und erscheinen hier mit ihren Mitgliedern, in jeder Facile-App.'
	],
	'spaces.create': [
		'Create a space in Registre',
		'Créer un espace dans Registre',
		'Crear un espacio en Registre',
		'Bereich in Registre erstellen'
	],
	'spaces.loadFailed': [
		'Could not load your spaces.',
		'Impossible de charger vos espaces.',
		'No se pudieron cargar tus espacios.',
		'Deine Bereiche konnten nicht geladen werden.'
	],
	'space.pageTitle': ['{name} — Nuage', '{name} — Nuage', '{name} — Nuage', '{name} — Nuage'],
	'space.fallbackName': ['Space', 'Espace', 'Espacio', 'Bereich'],
	'space.notFound': ['Space not found', 'Espace introuvable', 'Espacio no encontrado', 'Bereich nicht gefunden'],
	'space.notFoundText': [
		'It may have been deleted, or you may no longer be a member.',
		'Il a peut-être été supprimé, ou vous n’en êtes plus membre.',
		'Puede que se haya eliminado o que ya no seas miembro.',
		'Er wurde möglicherweise gelöscht, oder du bist nicht mehr Mitglied.'
	],
	'space.backToSpaces': ['Back to spaces', 'Retour aux espaces', 'Volver a espacios', 'Zurück zu den Bereichen'],
	'space.switched': [
		'Now working in “{name}”.',
		'Vous travaillez maintenant dans « {name} ».',
		'Ahora trabajas en «{name}».',
		'Du arbeitest jetzt in „{name}“.'
	],
	'space.current': ['Current space', 'Espace actuel', 'Espacio actual', 'Aktueller Bereich'],
	'space.switch': ['Switch to this space', 'Passer à cet espace', 'Cambiar a este espacio', 'Zu diesem Bereich wechseln'],
	'space.settings': ['Settings', 'Paramètres', 'Ajustes', 'Einstellungen'],
	'space.leave': ['Leave space', 'Quitter l’espace', 'Salir del espacio', 'Bereich verlassen'],
	'space.leaveFailed': [
		'Could not leave that space.',
		'Impossible de quitter cet espace.',
		'No se pudo salir de ese espacio.',
		'Der Bereich konnte nicht verlassen werden.'
	],
	'space.left': [
		'You left “{name}”.',
		'Vous avez quitté « {name} ».',
		'Has salido de «{name}».',
		'Du hast „{name}“ verlassen.'
	],
	'space.leaveTitle': [
		'Leave “{name}”?',
		'Quitter « {name} » ?',
		'¿Salir de «{name}»?',
		'„{name}“ verlassen?'
	],
	'space.leaveFallback': ['this space', 'cet espace', 'este espacio', 'diesen Bereich'],
	'space.leaveText': [
		'You lose access to every file in the space, including the ones you uploaded — they stay here. An owner or admin can add you back.',
		'Vous perdez l’accès à tous les fichiers de l’espace, y compris ceux que vous avez déposés : ils restent ici. Un propriétaire ou un administrateur peut vous réinviter.',
		'Pierdes el acceso a todos los archivos del espacio, incluidos los que subiste: se quedan aquí. Un propietario o administrador puede volver a añadirte.',
		'Du verlierst den Zugriff auf alle Dateien im Bereich, auch auf die, die du hochgeladen hast: Sie bleiben hier. Ein Eigentümer oder Administrator kann dich wieder hinzufügen.'
	],
	'space.membersHeading': ['Members', 'Membres', 'Miembros', 'Mitglieder'],
	'space.membersCount': [
		{
			one: '{count} person has access to everything in this space.',
			other: '{count} people have access to everything in this space.'
		},
		{
			one: '{count} personne a accès à tout ce qui se trouve dans cet espace.',
			other: '{count} personnes ont accès à tout ce qui se trouve dans cet espace.'
		},
		{
			one: '{count} persona tiene acceso a todo lo que hay en este espacio.',
			other: '{count} personas tienen acceso a todo lo que hay en este espacio.'
		},
		{
			one: '{count} Person hat Zugriff auf alles in diesem Bereich.',
			other: '{count} Personen haben Zugriff auf alles in diesem Bereich.'
		}
	],
	'space.manage': ['Manage', 'Gérer', 'Gestionar', 'Verwalten'],
	'space.about': ['About', 'À propos', 'Acerca de', 'Info'],
	'space.created': ['Created {date}', 'Créé le {date}', 'Creado el {date}', 'Erstellt am {date}']
} satisfies Record<string, Row>;
