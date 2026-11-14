import type { Row } from '../row';

/** The landing demos: the hero album and the three demo cards. */
export const landing = {
	'landing.album.folder': ['Holidays 2026', 'Vacances 2026', 'Vacaciones 2026', 'Urlaub 2026'],
	'landing.album.progress': [
		'{done} of {total} uploaded',
		'{done} sur {total} envoyées',
		'{done} de {total} subidas',
		'{done} von {total} hochgeladen'
	],
	'landing.cardsTitle': ['One cloud for everyone', 'Un cloud pour tout le monde', 'Una nube para todos', 'Eine Cloud für alle'],
	'landing.card.spaces.title': ['Spaces for your team', 'Des espaces d’équipe', 'Espacios para tu equipo', 'Bereiche fürs Team'],
	'landing.card.spaces.body': [
		'Everyone in the space sees new files as they land.',
		'Chaque membre voit les nouveaux fichiers dès leur arrivée.',
		'Todo el equipo ve los archivos nuevos en cuanto llegan.',
		'Alle im Bereich sehen neue Dateien, sobald sie ankommen.'
	],
	'landing.card.spaces.name': ['Studio team', 'Équipe studio', 'Equipo del estudio', 'Studio-Team'],
	'landing.card.phone.title': ['Opens on any phone', 'S’ouvre sur tout téléphone', 'Se abre en cualquier móvil', 'Öffnet auf jedem Handy'],
	'landing.card.phone.body': [
		'Family opens the link in a browser. No account needed.',
		'La famille ouvre le lien dans un navigateur. Sans compte.',
		'La familia abre el enlace en el navegador. Sin cuenta.',
		'Die Familie öffnet den Link im Browser. Ohne Konto.'
	],
	'landing.card.sync.title': ['Sync from the terminal', 'Synchro depuis le terminal', 'Sincroniza desde la terminal', 'Sync im Terminal'],
	'landing.card.sync.body': [
		'The nuage client syncs a folder and shares files from the shell.',
		'Le client nuage synchronise un dossier et partage depuis le shell.',
		'El cliente nuage sincroniza una carpeta y comparte desde la shell.',
		'Der nuage-Client synchronisiert einen Ordner und teilt aus der Shell.'
	]
} satisfies Record<string, Row>;
