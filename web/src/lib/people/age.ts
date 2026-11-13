// The capture-time age of one person in one photo: the birth date's
// whole purpose beside the birthday memories (web-ia §7.17 -- the
// person profile row's date exists so a photo's chip can say how old
// the person was when it was taken).
//
// THE COMPUTATION is calendar arithmetic on the LOCAL date parts, not
// a duration: an age is whole months between two calendar dates (the
// day-of-month comparison borrows a month the way a person counts
// birthdays), and a photo taken before the person was born has NO age
// to state -- null, never a negative number.
//
// THE DISPLAY RULES (the E10 pin): months under a year read as
// months, whole years read as years, anything else as years + months.
// The component calls formatAge with its own `$_` so the copy stays
// locale-tracked; this module carries no strings.

/** The person's age in whole calendar months at a moment, or null. */
export function ageInMonths(birthDate: string, takenAt: string): number | null {
	const birth = splitDate(birthDate);
	const taken = new Date(takenAt);
	if (birth === null || Number.isNaN(taken.getTime())) {
		return null;
	}
	let months = (taken.getFullYear() - birth.year) * 12 + (taken.getMonth() - (birth.month - 1));
	if (taken.getDate() < birth.day) {
		months -= 1;
	}
	return months < 0 ? null : months;
}

function splitDate(date: string): { year: number; month: number; day: number } | null {
	const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
	if (match === null) {
		return null;
	}
	return { year: Number(match[1]), month: Number(match[2]), day: Number(match[3]) };
}

/**
 * The display form of one capture-time age: null when there is no
 * birth date (nothing renders -- never a zero), otherwise the years /
 * months / years-and-months message the caller's `t` renders.
 */
export function formatAge(
	birthDate: string | null,
	takenAt: string,
	t: (id: string, options?: { values?: Record<string, string | number> }) => string
): string | null {
	if (birthDate === null) {
		return null;
	}
	const months = ageInMonths(birthDate, takenAt);
	if (months === null) {
		return null;
	}
	if (months < 12) {
		return t('person_age_months', { values: { months } });
	}
	const years = Math.floor(months / 12);
	const rest = months % 12;
	if (rest === 0) {
		return t('person_age_years', { values: { years } });
	}
	return t('person_age_year_months', { values: { years, months: rest } });
}