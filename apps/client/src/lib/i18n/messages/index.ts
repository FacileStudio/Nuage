import type { Catalogs, Message } from '@facile/muse';
import { activity } from './activity';
import { files } from './files';
import { members } from './members';
import { pages } from './pages';
import { settings } from './settings';
import { share } from './share';
import { shell } from './shell';
import { spaces } from './spaces';
import { tokens } from './tokens';
import { trash } from './trash';
import { viewer } from './viewer';
import type { Row } from '../row';

const rows = {
	...shell,
	...files,
	...share,
	...viewer,
	...pages,
	...trash,
	...activity,
	...spaces,
	...members,
	...settings,
	...tokens
};

export type AppKey = keyof typeof rows;
type AppCatalog = Record<AppKey, Message>;

function column(index: number): AppCatalog {
	const entries = Object.entries(rows as Record<string, Row>).map(([key, row]) => [key, row[index]]);
	return Object.fromEntries(entries) as AppCatalog;
}

/** The app's catalogs: English is the typed base, and every row carries all four locales. */
export const catalogs: Catalogs<AppCatalog> = {
	en: column(0),
	fr: column(1),
	es: column(2),
	de: column(3)
};
