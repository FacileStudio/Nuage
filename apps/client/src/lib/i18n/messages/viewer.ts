import type { Row } from '../row';

/** The file viewers, file-type names and the public link page. */
export const viewer = {
	'viewer.preview': ['File preview', 'Aperçu du fichier', 'Vista previa del archivo', 'Dateivorschau'],
	'viewer.close': ['Close preview', 'Fermer l’aperçu', 'Cerrar la vista previa', 'Vorschau schließen'],
	'viewer.share': ['Share', 'Partager', 'Compartir', 'Teilen'],
	'viewer.download': ['Download', 'Télécharger', 'Descargar', 'Herunterladen'],
	'viewer.previous': ['Previous file', 'Fichier précédent', 'Archivo anterior', 'Vorherige Datei'],
	'viewer.next': ['Next file', 'Fichier suivant', 'Archivo siguiente', 'Nächste Datei'],
	'viewer.siblings': [
		'Files in this folder',
		'Fichiers de ce dossier',
		'Archivos de esta carpeta',
		'Dateien in diesem Ordner'
	],
	'viewer.noPreview': ['No preview available', 'Aucun aperçu disponible', 'Vista previa no disponible', 'Keine Vorschau verfügbar'],
	'viewer.imageFailed': [
		'This image could not be displayed.',
		'Cette image n’a pas pu être affichée.',
		'No se pudo mostrar esta imagen.',
		'Dieses Bild konnte nicht angezeigt werden.'
	],
	'viewer.zoomOut': ['Zoom out', 'Zoom arrière', 'Alejar', 'Verkleinern'],
	'viewer.zoomIn': ['Zoom in', 'Zoom avant', 'Acercar', 'Vergrößern'],
	'viewer.resetZoom': ['Reset zoom', 'Réinitialiser le zoom', 'Restablecer zoom', 'Zoom zurücksetzen'],
	'viewer.rotate': ['Rotate', 'Pivoter', 'Girar', 'Drehen'],
	'viewer.fitWidth': ['Fit to width', 'Ajuster à la largeur', 'Ajustar al ancho', 'An Breite anpassen'],
	'viewer.prevPage': ['Previous page', 'Page précédente', 'Página anterior', 'Vorherige Seite'],
	'viewer.nextPage': ['Next page', 'Page suivante', 'Página siguiente', 'Nächste Seite'],
	'viewer.pageNumber': ['Page number', 'Numéro de page', 'Número de página', 'Seitenzahl'],
	'viewer.find': ['Find in document', 'Rechercher dans le document', 'Buscar en el documento', 'Im Dokument suchen'],
	'viewer.prevMatch': ['Previous match', 'Résultat précédent', 'Coincidencia anterior', 'Vorheriger Treffer'],
	'viewer.nextMatch': ['Next match', 'Résultat suivant', 'Coincidencia siguiente', 'Nächster Treffer'],
	'viewer.pdfFailed': [
		'This PDF could not be opened. Download it to view it in another app.',
		'Ce PDF n’a pas pu être ouvert. Téléchargez-le pour l’afficher dans une autre app.',
		'No se pudo abrir este PDF. Descárgalo para verlo en otra app.',
		'Dieses PDF konnte nicht geöffnet werden. Lade es herunter, um es in einer anderen App anzusehen.'
	],
	'viewer.seek': ['Seek', 'Position', 'Posición', 'Suchen'],
	'viewer.back10': ['Back 10 seconds', 'Reculer de 10 secondes', 'Retroceder 10 segundos', '10 Sekunden zurück'],
	'viewer.forward10': ['Forward 10 seconds', 'Avancer de 10 secondes', 'Avanzar 10 segundos', '10 Sekunden vor'],
	'viewer.play': ['Play', 'Lecture', 'Reproducir', 'Wiedergabe'],
	'viewer.pause': ['Pause', 'Pause', 'Pausa', 'Pause'],
	'viewer.speed': ['Playback speed', 'Vitesse de lecture', 'Velocidad de reproducción', 'Wiedergabegeschwindigkeit'],
	'viewer.mute': ['Mute', 'Couper le son', 'Silenciar', 'Stummschalten'],
	'viewer.unmute': ['Unmute', 'Rétablir le son', 'Activar sonido', 'Ton einschalten'],
	'viewer.volume': ['Volume', 'Volume', 'Volumen', 'Lautstärke'],
	'viewer.fullscreen': ['Full screen', 'Plein écran', 'Pantalla completa', 'Vollbild'],
	'viewer.exitFullscreen': ['Exit full screen', 'Quitter le plein écran', 'Salir de pantalla completa', 'Vollbild beenden'],
	'viewer.cannotPlay': [
		'This file cannot be played in the browser. Download it to open it.',
		'Ce fichier ne peut pas être lu dans le navigateur. Téléchargez-le pour l’ouvrir.',
		'Este archivo no se puede reproducir en el navegador. Descárgalo para abrirlo.',
		'Diese Datei lässt sich im Browser nicht abspielen. Lade sie herunter, um sie zu öffnen.'
	],
	'viewer.videoPlayer': ['Video player: {name}', 'Lecteur vidéo : {name}', 'Reproductor de vídeo: {name}', 'Videoplayer: {name}'],
	'viewer.audioPlayer': ['Audio player: {name}', 'Lecteur audio : {name}', 'Reproductor de audio: {name}', 'Audioplayer: {name}'],
	'viewer.paused': ['Paused', 'En pause', 'En pausa', 'Pausiert'],
	'viewer.playing': ['Playing', 'En lecture', 'Reproduciendo', 'Wiedergabe läuft'],
	'type.image': ['Image', 'Image', 'Imagen', 'Bild'],
	'type.video': ['Video', 'Vidéo', 'Vídeo', 'Video'],
	'type.audio': ['Audio', 'Audio', 'Audio', 'Audio'],
	'type.pdf': ['PDF', 'PDF', 'PDF', 'PDF'],
	'type.archive': ['Archive', 'Archive', 'Archivo comprimido', 'Archiv'],
	'type.sheet': ['Spreadsheet', 'Tableur', 'Hoja de cálculo', 'Tabelle'],
	'type.slides': ['Presentation', 'Présentation', 'Presentación', 'Präsentation'],
	'type.document': ['Document', 'Document', 'Documento', 'Dokument'],
	'type.code': ['Code', 'Code', 'Código', 'Code'],
	'type.file': ['File', 'Fichier', 'Archivo', 'Datei'],
	'public.shared': ['Shared', 'Partagé', 'Compartido', 'Geteilt'],
	'public.title': ['{name} — Nuage', '{name} — Nuage', '{name} — Nuage', '{name} — Nuage'],
	'public.expired': [
		'This link has expired or does not exist',
		'Ce lien a expiré ou n’existe pas',
		'Este enlace ha caducado o no existe',
		'Dieser Link ist abgelaufen oder existiert nicht'
	],
	'public.expiredHint': [
		'Check the address, or ask whoever sent it for a fresh link.',
		'Vérifiez l’adresse, ou demandez un nouveau lien à la personne qui vous l’a envoyé.',
		'Comprueba la dirección o pide un enlace nuevo a quien te lo envió.',
		'Prüfe die Adresse oder bitte die absendende Person um einen neuen Link.'
	],
	'public.downloadFile': ['Download {name}', 'Télécharger {name}', 'Descargar {name}', '{name} herunterladen'],
	'public.preview': ['Preview {name}', 'Aperçu de {name}', 'Vista previa de {name}', 'Vorschau von {name}'],
	'public.emptyTitle': ['This folder is empty', 'Ce dossier est vide', 'Esta carpeta está vacía', 'Dieser Ordner ist leer'],
	'public.emptyText': [
		'There is nothing to show here.',
		'Il n’y a rien à afficher ici.',
		'No hay nada que mostrar aquí.',
		'Hier gibt es nichts anzuzeigen.'
	]
} satisfies Record<string, Row>;
