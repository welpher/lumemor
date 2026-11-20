// The jobs page test's reactive route state: the real `$app/state`
// page is a rune-backed object, so an address change (the tab
// switch) re-runs the page's derived face. A plain object mock would
// freeze the face at mount, so the test state lives here -- a runes
// module the mock factory and the tests share.

export const jobsPageState = $state({
	url: new URL('http://localhost/admin/jobs')
});
