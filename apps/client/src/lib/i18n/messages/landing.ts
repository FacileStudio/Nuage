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
	'landing.card.apps.title': [
		'Your files in your other apps',
		'Vos fichiers dans vos autres apps',
		'Tus archivos en tus otras apps',
		'Ihre Dateien in Ihren anderen Apps'
	],
	'landing.card.apps.body': [
		'A contract signed in Plume is stored in your Nuage on its own.',
		'Un contrat signé dans Plume se range tout seul dans votre Nuage.',
		'Un contrato firmado en Plume se guarda solo en tu Nuage.',
		'Ein in Plume unterschriebener Vertrag landet von selbst in Ihrem Nuage.'
	],
	'landing.card.apps.signed': ['Signed', 'Signé', 'Firmado', 'Unterschrieben'],
	'landing.album.names': [
		'beach.jpg,sunset.jpg,mountain.jpg,lake.jpg,village.jpg,picnic.jpg',
		'plage.jpg,coucher.jpg,montagne.jpg,lac.jpg,village.jpg,pique-nique.jpg',
		'playa.jpg,atardecer.jpg,montana.jpg,lago.jpg,pueblo.jpg,picnic.jpg',
		'strand.jpg,abendrot.jpg,berge.jpg,see.jpg,dorf.jpg,picknick.jpg'
	],
	'landing.metaTitle': [
		'Nuage · Your files everywhere, shared with a link',
		'Nuage · Vos fichiers partout, partagés en un lien',
		'Nuage · Tus archivos en todas partes, compartidos con un enlace',
		'Nuage · Ihre Dateien überall, geteilt mit einem Link'
	],
	'landing.metaDescription': [
		'Keep your photos and documents, share them by link and work together in team folders. Free to start.',
		'Rangez vos photos et documents, partagez-les par lien et travaillez à plusieurs dans des dossiers d’équipe. Gratuit pour commencer.',
		'Guarda tus fotos y documentos, compártelos con un enlace y trabaja en equipo en carpetas compartidas. Gratis para empezar.',
		'Bewahren Sie Fotos und Dokumente auf, teilen Sie sie per Link und arbeiten Sie gemeinsam in Teamordnern. Kostenlos starten.'
	],
	'landing.with.plume': [
		'Your signed documents land here on their own.',
		'Vos documents signés se rangent ici tout seuls.',
		'Tus documentos firmados se guardan aquí solos.',
		'Ihre unterschriebenen Dokumente landen von selbst hier.'
	],
	'landing.with.ardoise': [
		'Your contracts are filed in your Ardoise folder.',
		'Vos contrats sont classés dans votre dossier Ardoise.',
		'Tus contratos se archivan en tu carpeta de Ardoise.',
		'Ihre Verträge liegen in Ihrem Ardoise-Ordner.'
	],
	'landing.with.toile': [
		'Your board images are kept here.',
		'Les images de vos tableaux sont gardées ici.',
		'Las imágenes de tus pizarras se guardan aquí.',
		'Die Bilder Ihrer Boards liegen hier.'
	],
	'landing.with.bulle': [
		'Files and finds your documents for you.',
		'Range et retrouve vos fichiers pour vous.',
		'Guarda y encuentra tus archivos por ti.',
		'Legt Ihre Dateien ab und findet sie für Sie.'
	],
	'landing.dev.text': [
		'A command-line sync client, WebDAV and an MCP server. Open source.',
		'Un client de synchro en ligne de commande, WebDAV et un serveur MCP. Code ouvert.',
		'Un cliente de sincronización en línea de comandos, WebDAV y un servidor MCP. Código abierto.',
		'Ein Sync-Client für die Kommandozeile, WebDAV und ein MCP-Server. Open Source.'
	],
	'landing.dev.cli': [
		'Keep a folder in sync and share files from the terminal, on macOS and Linux.',
		'Synchronisez un dossier et partagez depuis le terminal, sur macOS et Linux.',
		'Sincroniza una carpeta y comparte desde la terminal, en macOS y Linux.',
		'Einen Ordner synchronisieren und aus dem Terminal teilen, unter macOS und Linux.'
	],
	'landing.dev.webdav': [
		'Mount your files as a network drive at /webdav.',
		'Montez vos fichiers comme un lecteur réseau sur /webdav.',
		'Monta tus archivos como unidad de red en /webdav.',
		'Binden Sie Ihre Dateien als Netzlaufwerk unter /webdav ein.'
	],
	'landing.dev.api': [
		'A documented REST API with API tokens.',
		'Une API REST documentée, avec jetons d’API.',
		'Una API REST documentada, con tokens de API.',
		'Eine dokumentierte REST-API mit API-Tokens.'
	],
	'landing.dev.mcp': [
		'Reach your files from any MCP client at /mcp.',
		'Ouvrez vos fichiers depuis un client MCP sur /mcp.',
		'Abre tus archivos desde cualquier cliente MCP en /mcp.',
		'Öffnen Sie Ihre Dateien aus jedem MCP-Client unter /mcp.'
	],
	'landing.dev.selfHost': [
		'https://suite.facile.studio/en/auto-hebergement',
		'https://suite.facile.studio/auto-hebergement',
		'https://suite.facile.studio/es/auto-hebergement',
		'https://suite.facile.studio/de/auto-hebergement'
	],
	'landing.dev.source': [
		'The code is open, your data stays yours.',
		'Code ouvert, vos données restent les vôtres.',
		'Código abierto, tus datos siguen siendo tuyos.',
		'Offener Code, Ihre Daten bleiben Ihre.'
	]
} satisfies Record<string, Row>;
