import { describe, expect, test } from 'bun:test';
import { isSelfOrDescendant } from './tree';

/*     1
 *     └── 2
 *         └── 3      4 (unrelated root)
 */
const TREE = [
	{ id: 1, parent_id: null },
	{ id: 2, parent_id: 1 },
	{ id: 3, parent_id: 2 },
	{ id: 4, parent_id: null }
];

describe('isSelfOrDescendant', () => {
	test('a folder is its own descendant, so it cannot be dropped into itself', () => {
		expect(isSelfOrDescendant(2, 2, TREE)).toBe(true);
	});

	test('catches a grandchild, not just a direct child', () => {
		expect(isSelfOrDescendant(3, 1, TREE)).toBe(true);
	});

	test('allows moving up the tree and across to a sibling branch', () => {
		expect(isSelfOrDescendant(1, 3, TREE)).toBe(false);
		expect(isSelfOrDescendant(4, 1, TREE)).toBe(false);
	});

	test('terminates on a cycle instead of hanging the tab', () => {
		const cyclic = [
			{ id: 1, parent_id: 2 },
			{ id: 2, parent_id: 1 }
		];
		expect(isSelfOrDescendant(1, 99, cyclic)).toBe(false);
	});
});
