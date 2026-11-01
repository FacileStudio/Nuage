import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url';

export type PdfEvents = {
	onPage: (page: number) => void;
	onScale: (scale: number) => void;
	onMatches: (matches: { current: number; total: number }) => void;
};

/**
 * One open document inside pdf.js's own PDFViewer: continuous scrolling, lazy page rendering,
 * a text layer for selection and the find controller. The component keeps the toolbar; this
 * keeps every pdf.js object, and `destroy` releases the worker and the transferred memory.
 */
export class PdfSession {
	pages = 0;
	#viewer: any;
	#bus: any;
	#doc: any;

	constructor(viewer: any, bus: any, doc: any) {
		this.#viewer = viewer;
		this.#bus = bus;
		this.#doc = doc;
		this.pages = doc.numPages;
	}

	goTo(page: number): number {
		if (!Number.isFinite(page)) return this.#viewer.currentPageNumber;
		this.#viewer.currentPageNumber = Math.min(this.pages, Math.max(1, Math.round(page)));
		return this.#viewer.currentPageNumber;
	}

	zoom(direction: 1 | -1): void {
		if (direction > 0) this.#viewer.increaseScale();
		else this.#viewer.decreaseScale();
	}

	zoomBy(factor: number, origin?: [number, number]): void {
		this.#viewer.updateScale({ scaleFactor: factor, drawingDelay: 120, origin });
	}

	fitWidth(): void {
		this.#viewer.currentScaleValue = 'page-width';
	}

	find(query: string, again = false, previous = false): void {
		this.#bus.dispatch('find', {
			source: null,
			type: again ? 'again' : '',
			query,
			caseSensitive: false,
			entireWord: false,
			highlightAll: true,
			findPrevious: previous,
			matchDiacritics: false
		});
	}

	destroy(): void {
		this.#viewer.setDocument(null);
		void this.#doc.destroy();
	}
}

/**
 * Loads `url` into `viewerEl`. Range requests come in 1 MB chunks rather than pdf.js's 64 KB
 * default, so a large document costs dozens of requests instead of hundreds.
 */
export async function openPdf(
	url: string,
	container: HTMLDivElement,
	viewerEl: HTMLDivElement,
	events: PdfEvents
): Promise<PdfSession> {
	const pdfjs = await import('pdfjs-dist');
	pdfjs.GlobalWorkerOptions.workerSrc = workerUrl;
	(globalThis as any).pdfjsLib = pdfjs;
	const web = await import('pdfjs-dist/web/pdf_viewer.mjs');
	const doc = await pdfjs.getDocument({ url, rangeChunkSize: 1 << 20 }).promise;

	const bus = new web.EventBus();
	const linkService = new web.PDFLinkService({ eventBus: bus });
	const findController = new web.PDFFindController({ eventBus: bus, linkService });
	const viewer = new web.PDFViewer({
		container,
		viewer: viewerEl,
		eventBus: bus,
		linkService,
		findController,
		removePageBorders: true,
		annotationMode: 0
	});
	linkService.setViewer(viewer);
	bus.on('pagesinit', () => (viewer.currentScaleValue = 'page-width'));
	bus.on('pagechanging', (e: { pageNumber: number }) => events.onPage(e.pageNumber));
	bus.on('scalechanging', (e: { scale: number }) => events.onScale(e.scale));
	bus.on('updatefindmatchescount', (e: any) => events.onMatches(e.matchesCount));
	bus.on('updatefindcontrolstate', (e: any) => e.matchesCount && events.onMatches(e.matchesCount));
	viewer.setDocument(doc);
	linkService.setDocument(doc);
	return new PdfSession(viewer, bus, doc);
}
