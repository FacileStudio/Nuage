import type { Row } from '../row';

/** The Ideas page: suggestions Bulle makes about the files, run in place. */
export const ideas = {
	'ideas.title': ['Ideas — Nuage', 'Idées — Nuage', 'Ideas — Nuage', 'Ideen — Nuage'],
	'ideas.description': [
		'Bulle looks at your files and suggests what to tidy, find or share. Run an idea and it is done here.',
		'Bulle regarde vos fichiers et propose quoi ranger, retrouver ou partager. Lancez une idée, elle se fait ici.',
		'Bulle revisa tus archivos y propone qué ordenar, encontrar o compartir. Lanza una idea y se hace aquí.',
		'Bulle sieht sich Ihre Dateien an und schlägt vor, was aufzuräumen, zu finden oder zu teilen ist. Starten Sie eine Idee, sie wird hier erledigt.'
	],
	'ideas.approval.summary': [
		'Bulle wants to run "{tool}". Check it before it goes ahead.',
		'Bulle veut lancer « {tool} ». Vérifiez avant de continuer.',
		'Bulle quiere ejecutar «{tool}». Revísalo antes de seguir.',
		'Bulle möchte „{tool}“ ausführen. Prüfen Sie es, bevor es weitergeht.'
	],
	'ideas.tool.nuage_save': ['Save a file', 'Enregistrer un fichier', 'Guardar un archivo', 'Datei speichern'],
	'ideas.tool.nuage_list': ['List files', 'Lister les fichiers', 'Listar archivos', 'Dateien auflisten'],
	'ideas.tool.nuage_read': ['Read a file', 'Lire un fichier', 'Leer un archivo', 'Datei lesen'],
	'ideas.tool.nuage_search': ['Search files', 'Chercher des fichiers', 'Buscar archivos', 'Dateien suchen'],
	'ideas.error.load': [
		'Could not load the ideas.',
		'Impossible de charger les idées.',
		'No se pudieron cargar las ideas.',
		'Die Ideen konnten nicht geladen werden.'
	],
	'ideas.error.unreachable': [
		'Bulle is unreachable for now. Try again in a moment.',
		'Bulle est injoignable pour le moment. Réessayez dans un instant.',
		'Bulle no responde por ahora. Inténtalo en un momento.',
		'Bulle ist gerade nicht erreichbar. Versuchen Sie es gleich noch einmal.'
	],
	'ideas.error.notLinked': [
		'Reconnect your Facile account to use Bulle, or pick a space that Registre manages.',
		'Reconnectez votre compte Facile pour utiliser Bulle, ou choisissez un espace géré par Registre.',
		'Vuelve a conectar tu cuenta Facile para usar Bulle, o elige un espacio gestionado por Registre.',
		'Verbinden Sie Ihr Facile-Konto erneut oder wählen Sie einen von Registre verwalteten Bereich.'
	],
	'ideas.error.capped': [
		'You asked for ideas three times today. More come tomorrow.',
		'Vous avez demandé des idées trois fois aujourd’hui. D’autres arrivent demain.',
		'Ya pediste ideas tres veces hoy. Mañana llegan más.',
		'Sie haben heute dreimal nach Ideen gefragt. Morgen kommen neue.'
	],
	'ideas.error.busy': [
		'Bulle is already thinking of ideas for this space.',
		'Bulle cherche déjà des idées pour cet espace.',
		'Bulle ya está buscando ideas para este espacio.',
		'Bulle sucht bereits Ideen für diesen Bereich.'
	],
	'ideas.error.gone': [
		'This idea is no longer available.',
		'Cette idée n’est plus disponible.',
		'Esta idea ya no está disponible.',
		'Diese Idee ist nicht mehr verfügbar.'
	],
	'ideas.error.run': [
		'The run stopped before the end.',
		'L’exécution s’est arrêtée avant la fin.',
		'La ejecución se detuvo antes del final.',
		'Der Lauf wurde vor dem Ende abgebrochen.'
	],
	'ideas.example.invoices.title': [
		'File the loose invoices',
		'Ranger les factures éparpillées',
		'Ordenar las facturas sueltas',
		'Lose Rechnungen ablegen'
	],
	'ideas.example.invoices.why': [
		'Eleven PDFs named scan-0412 and similar sit at the root. An Invoices 2026 folder would hold them.',
		'Onze PDF nommés scan-0412 et consorts traînent à la racine. Un dossier Factures 2026 les accueillerait.',
		'Once PDF llamados scan-0412 y similares están en la raíz. Una carpeta Facturas 2026 los acogería.',
		'Elf PDFs namens scan-0412 und ähnlich liegen im Stammordner. Ein Ordner Rechnungen 2026 würde sie aufnehmen.'
	],
	'ideas.example.duplicates.title': [
		'Clear the duplicate photos',
		'Supprimer les photos en double',
		'Eliminar las fotos duplicadas',
		'Doppelte Fotos entfernen'
	],
	'ideas.example.duplicates.why': [
		'Three copies of the holiday album take 1.2 GB of your plan.',
		'Trois copies de l’album de vacances occupent 1,2 Go de votre forfait.',
		'Tres copias del álbum de vacaciones ocupan 1,2 GB de tu plan.',
		'Drei Kopien des Urlaubsalbums belegen 1,2 GB Ihres Tarifs.'
	],
	'ideas.example.share.title': [
		'Share the signed contract',
		'Partager le contrat signé',
		'Compartir el contrato firmado',
		'Den unterschriebenen Vertrag teilen'
	],
	'ideas.example.share.why': [
		'Marc asked for it twice by mail. A read-only link is ready to send.',
		'Marc l’a demandé deux fois par mail. Un lien en lecture seule est prêt à envoyer.',
		'Marc lo pidió dos veces por correo. Hay un enlace de solo lectura listo para enviar.',
		'Marc hat ihn zweimal per Mail angefragt. Ein Link mit Lesezugriff liegt bereit.'
	],
	'ideas.dismissed': ['Idea dismissed', 'Idée écartée', 'Idea descartada', 'Idee verworfen']
} as const satisfies Record<string, Row>;
