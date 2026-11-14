// The places page test's reactive route state: the real `$app/state`
// page is a rune-backed object, so a same-route address change (the
// list <-> map switch) re-runs the page's derived face. A plain object
// mock would freeze the face at mount, so the test state lives here --
// a runes module the mock factory and the test share.

export const placesPageState = $state({
	url: new URL('http://localhost/places')
});
