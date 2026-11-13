// The identity presentation rule: "who is this" renders
// the account's display name, and an account that set none falls back to
// the email -- a blank attribution never renders. The email is the
// identity's unique key, so it keeps its own places (sign-in, devices,
// the admin user table) and doubles as the disambiguator when the
// surface shows two accounts under one name.
//
// The helpers live here rather than beside one surface because the rule
// is shared: the album and memory walls' attribution chips, the memory
// detail's bar and the audience pickers all read it, and a second
// spelling of the fallback would drift.

/** The identity fields the rule reads: the stored name and the email. */
export type Identity = {
	/** The account's display name; '' (or absent) when it set none. */
	name?: string;
	/** The account's login address: the unique key and the fallback. */
	email: string;
};

/** One account of a surface's known set, as the ambiguity check reads it. */
export type NamedAccount = {
	id: string;
	name: string;
	email: string;
};

/**
 * The display label of an identity: the stored name, or the email when
 * the name is empty. A name of whitespace is no name (defensive: the
 * server serves the stored value, and the trim keeps a stray space from
 * rendering as an empty attribution).
 */
export function displayNameOf(identity: Identity): string {
	const name = identity.name?.trim() ?? '';
	return name === '' ? identity.email : name;
}

/**
 * The avatar initial of an identity: the display label's first rune --
 * a CJK name gives its first character, a Latin one its capital.
 */
export function initialOf(identity: Identity): string {
	const label = displayNameOf(identity);
	return [...label][0]?.toUpperCase() ?? '?';
}

/**
 * The labels a known set gives MORE THAN ONE account: the display names
 * two different accounts carry. `name` is not a unique key (the
 * contract pins that), so a surface that shows the whole set hangs the
 * email on exactly these labels as the hover title -- a unique name
 * needs no second spelling. Identities already labelled by their email
 * never collide (the address is the key), and one account's repeated
 * appearance (two cards from the same owner) is no ambiguity.
 */
export function ambiguousLabels(accounts: readonly NamedAccount[]): ReadonlySet<string> {
	const holders = new Map<string, Set<string>>();
	for (const account of accounts) {
		const label = displayNameOf(account);
		if (label === account.email) {
			continue;
		}
		const ids = holders.get(label) ?? new Set<string>();
		ids.add(account.id);
		holders.set(label, ids);
	}
	const ambiguous = new Set<string>();
	for (const [label, ids] of holders) {
		if (ids.size > 1) {
			ambiguous.add(label);
		}
	}
	return ambiguous;
}
