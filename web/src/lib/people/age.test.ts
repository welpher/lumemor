import { describe, expect, test } from 'vitest';
import { ageInMonths } from './age';

// The capture-time age helper: calendar months between the birth date
// and the photo's moment, the day-of-month borrow, and the "before
// the person was born" arm (no age to state -- null, never negative).
// The display arms (the three i18n keys) are the component's, tested
// through the page suites.

describe('ageInMonths', () => {
	test('whole years and the month remainder', () => {
		expect(ageInMonths('1992-03-15', '2026-03-15T12:00:00Z')).toBe(408);
		expect(ageInMonths('1992-03-15', '2026-04-15T12:00:00Z')).toBe(409);
	});

	test('the day-of-month borrow: a day short of the anniversary is a month short', () => {
		expect(ageInMonths('1992-03-15', '2026-03-14T12:00:00Z')).toBe(407);
		expect(ageInMonths('1992-03-15', '2026-04-14T12:00:00Z')).toBe(408);
	});

	test('under a year reads in months', () => {
		expect(ageInMonths('2026-01-10', '2026-04-09T12:00:00Z')).toBe(2);
		expect(ageInMonths('2026-01-10', '2026-04-10T12:00:00Z')).toBe(3);
	});

	test('a photo taken before the birth date has no age', () => {
		expect(ageInMonths('1992-03-15', '1992-03-14T12:00:00Z')).toBeNull();
	});

	test('a malformed birth date or taken-at has no age', () => {
		expect(ageInMonths('15/03/1992', '2026-03-15T12:00:00Z')).toBeNull();
		expect(ageInMonths('1992-03-15', 'not a date')).toBeNull();
	});
});
