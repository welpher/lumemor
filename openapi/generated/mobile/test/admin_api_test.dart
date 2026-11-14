import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for AdminApi
void main() {
  final instance = LumemorApi().getAdminApi();

  group(AdminApi, () {
    // Activate the model (the model switch).
    //
    // The activation switch of the admin surface: the model becomes the active one of its slot. The request body is an empty JSON object -- the path's model identifier is the whole request.  A same-slot switch runs in one transaction that only flips the activation flags: the previous model's products stay in the database untouched (multi-generation embeddings and faces coexist by their model columns), queries that filter on the active model stop seeing them, and switching back revives them with zero recompute -- but the newly activated model does nothing until its products are rerun.  Cross-dimension switches of the clip-embed slot are the one exception, because a vector column cannot hold two dimensions: activating a clip model whose dimension differs from the `clip_embeddings` column's current one (read from the database catalog, never assumed) clears the table and re-types the column to `vector(N)` inside the SAME transaction as the flag flip -- the wipe, the new dimension and the activation land together or not at all, and a failure rolls everything back. The response then reports the switch's facts (`dimensionChanged` with the from and to dimensions, `wipedEmbeddings` with the number of deleted rows) so the admin UI can demand the explicit confirmation a destructive switch deserves. The rerun that fills the new model's embeddings stays manual: nothing is recomputed here.  The model's files are verified against their pinned sha256 digests before anything flips: a model whose files are missing or drifted answers 400 MODEL_INVALID and activates nothing. 
    //
    //Future<ModelActivateResponse> activateModel(String id, BuiltMap<String, JsonObject?> requestBody) async
    test('test activateModel', () async {
      // TODO
    });

    // Create one account.
    //
    // The account creation of the admin surface, the management counterpart of the first-run wizard: an administrator provisions another account -- a family member's login, or a second administrator. The account rules mirror the wizard's exactly: the email is parsed and lower-cased as the login identity, the password is stored only as an Argon2id hash and never appears in any response, the id is a fresh UUIDv7, and the account's sync album -- the core backup stream every upload lands in and the timeline aggregates -- is created in the SAME transaction, so the new account can take uploads the moment it exists (a second creation point of the same wiring the wizard uses). The wizard's one-time guarantee is NOT part of this endpoint; it exists exactly because an administrator already does.  The role defaults to `user` when omitted; creating another administrator is the same request with `role: \"admin\"`. A duplicate email answers 409 USER_EMAIL_TAKEN -- the address is the account identity, unique instance-wide, and the login lookup depends on that uniqueness. 
    //
    //Future<AdminUserResponse> createUser(AdminCreateUserRequest adminCreateUserRequest) async
    test('test createUser', () async {
      // TODO
    });

    // Start the default model group's download.
    //
    // The model download trigger of the admin surface: the default group's models -- pinned by the manifest with upstream URLs and sha256 digests -- are downloaded, verified and registered, and each model activates into its slot when the slot is still empty. The request body is deliberately empty: the default group is the only group v1 downloads.  The endpoint is idempotent. A run that is still in flight (state `queued` or `running`) answers 200 with its identifier and nothing new is started; so does the most recent finished run -- a repeated click neither re-downloads a landed group nor piles up jobs, and the recovery of a failed run belongs to the job's own retries. Only a first call -- no run of this kind at all -- creates one and answers 202.  The run's progress is observable two ways: the `job_runs` row of the returned identifier (kind `model_download`, `total` and `done` carrying the group's pinned byte counts) and the `job.started` / `job.progress` / `job.finished` events on the activity stream, the progress events throttled to at most one per second. 
    //
    //Future<ModelDownloadResponse> downloadModels() async
    test('test downloadModels', () async {
      // TODO
    });

    // Read the model download settings.
    //
    // The model store's one policy switch: whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own.  WHAT AUTO-FILLS: only a slot that already has an active model whose files are gone from the store -- the case of a store wiped below a configured instance. A slot that was never configured stays empty; the first-run download and the settings face's download button remain the administrator's active path, and an unconfigured slot keeps the honest no-AI degraded mode.  A corrupt settings row answers the default (on) with the corruption logged server-side. 
    //
    //Future<ModelSettingsResponse> getModelSettings() async
    test('test getModelSettings', () async {
      // TODO
    });

    // Read the storage template setting.
    //
    // The storage settings read of the admin surface: the instance's upload path template, the sample path it renders, and the built-in preset list the settings section's dropdown fills from.  The template is the path grammar every NEW upload renders its file placement from: `originals/<ownerUserId>/<rendered>.<ext>`, with the per-user root pinning one directory tree per account. The grammar is variable substitution plus `{{#if album}}...{{else}}...{{/if}}` conditionals over the file (`{{filename}}`, `{{ext}}`, `{{filetype}}`, `{{filetypefull}}`), date (`{{y}} {{yy}} {{M}} {{MM}} {{MMM}} {{MMMM}} {{d}} {{dd}} {{W}} {{WW}} {{H}} {{HH}} {{h}} {{hh}} {{m}} {{mm}} {{s}} {{ss}} {{SSS}}`), album (`{{album}}`, `{{album-startDate-*}}`, `{{album-endDate-*}}`), camera (`{{make}} {{model}} {{lensModel}}`) and id (`{{assetId}}`, `{{assetIdShort}}`) families. Compilation is strict -- an unknown variable or a malformed conditional is an error, never a silent blank.  THE RENDERING CONTRACT (what changing the template does and never does): the path is rendered exactly once, at ingest, from the taken-at fallback chain (the file's EXIF header time, else the client-reported `deviceCreatedAt`, else the upload time) and the values known at that moment. Later metadata refinement NEVER re-paths an asset -- a refined taken-at, an album membership joined after upload, none of it moves a file. Existing assets keep the paths they were ingested under; there is deliberately NO migration of stored files when the template changes (moving files after ingest is the architecture photo libraries have lost data to; lumemor never moves them). New uploads follow the new template from the moment it is saved. Album variables render empty for direct uploads: the only album an upload joins at ingest is the system sync stream, which is not a naming source.  The sample path is the current template rendered over a fixed sample asset (an image named `IMG_1234.jpg`, taken `2023:07:14 10:15:30.123 +05:30`, camera `Acme Alpha 5` with the `24-70mm f/2.8` lens, no album, under the sample owner `0192f8c1-0000-7000-8000-000000000001`) with the extension appended -- exactly the shape a real upload of that file would land at under this template. The response never fails over a corrupt setting row: the default template answers instead, and the server log carries the corruption. 
    //
    //Future<StorageSettingsResponse> getStorageSettings() async
    test('test getStorageSettings', () async {
      // TODO
    });

    // Read the map tile source setting.
    //
    // The map's tile source, as the administrator configured it: one URL, or the empty string when the instance has none.  THE SHAPE: the URL names a document the browser's map library can build a base layer from -- a style document (the vector case: sources, layers, sprites and glyphs) or a TileJSON descriptor (the raster case: the tile templates and their attribution). The map detects which of the two it received and builds the base layer accordingly, so one field covers both the raster and the vector-and-style forms of the self-hosting ecosystem.  THE EMPTY VALUE IS THE SHIPPED DEFAULT and a real state: no source ships with the product (a public tile service's usage policy forbids carrying the traffic of every install), so a fresh instance draws its map with the tile layer absent and says so. Loading the document needs the host to send the usual cross-origin headers; a host that does not lands on the same stated absence.  A corrupt settings row answers the empty string with the corruption logged server-side. 
    //
    //Future<TilesSettingsResponse> getTilesSettings() async
    test('test getTilesSettings', () async {
      // TODO
    });

    // Read the video transcoding settings.
    //
    // The transcoding settings read of the admin surface: the instance's video transcoding profile -- the preset the transcode job re-encodes new video uploads at, the concurrency bound of simultaneous re-encodes, the video-thumbnail frame capture switch, and the keep-original-on-failure policy. A fresh instance (or a corrupt row) answers the documented defaults. 
    //
    //Future<TranscodingSettingsResponse> getTranscodingSettings() async
    test('test getTranscodingSettings', () async {
      // TODO
    });

    // Read the trash retention setting.
    //
    // The trash settings read of the admin surface: the instance's trash retention in whole days -- the window a soft-deleted asset stays recoverable before the hourly expiry job purges it (rows, thumbnails, derivatives and orphaned files together). A fresh instance (or a corrupt row) answers the default 30; the read never fails over a row the save endpoint's validation should have kept honest. 
    //
    //Future<TrashSettingsResponse> getTrashSettings() async
    test('test getTrashSettings', () async {
      // TODO
    });

    // Read the instance's release check state.
    //
    // The instance's release check, read from the cached answer -- this request never touches the network. The server runs the probe itself on the configured cadence (a release feed URL, the interval in hours) and this face reports what the last probe concluded.  THE PROBE: one plain HTTPS request for a public document -- no photo, no EXIF, no embedding, no instance identifier; the feed sees the instance's address and nothing else. The check can be switched off (a settings switch); the endpoint address must be a public one: an address that resolves to a loopback, private, link-local or otherwise internal range is refused, and redirects are never followed. A refused or failed probe is not an error state: `status` turns `unknown` and nothing else changes -- the next interval retries.  THE COMPARISON takes tags only: an optional `v` prefix, then the numeric version fields. `up_to_date` and `update_available` are the two claims a successful probe can make; `latestVersion` carries the feed's tag verbatim. The server never downloads or installs an update -- this surface only reports that a newer release exists.  `disabled` means the check does not run: the administrator switched it off, or the deployment disabled it at boot. The last answer is kept while disabled, so switching the check back on resumes from the same state. 
    //
    //Future<UpdateCheckResponse> getUpdateCheck() async
    test('test getUpdateCheck', () async {
      // TODO
    });

    // Read the release check settings.
    //
    // The release check's three knobs, as the settings face renders them: the switch, the release feed URL and the cadence in hours.  `endpoint` is the URL the server fetches on every check. The document behind it must be a JSON object whose newest tag rides `tag` (a release feed that names it `tag_name`, as release APIs commonly do, is read the same way). The value is handed back verbatim; the default is the product's own release feed.  `intervalHours` is the cadence: the server probes at most once per interval, and the answer is cached until the next one -- a failed probe waits the same interval instead of hammering a feed that is not there.  A corrupt settings row answers the documented defaults (enabled, the default feed, 24 hours) with the corruption logged server-side, the same discipline the other instance settings keep. 
    //
    //Future<UpdateCheckSettingsResponse> getUpdateCheckSettings() async
    test('test getUpdateCheckSettings', () async {
      // TODO
    });

    // List the job runs of the task panel.
    //
    // The task panel listing of the admin surface: every job run the instance ever recorded, newest first -- model downloads and ML reruns alike. Each row carries the run's progress as its own work's units: the model download's `total` and `done` count pinned bytes and `failed` counts failed models; a rerun's count the assets of its scope and the assets whose rerun failed. `startedAt` is null while a run sits `queued`, `finishedAt` is null until it closes.  `limit` bounds the page size: absent means the default of 50, a value over 200 is answered with the maximum of 200, and a value below 1 answers 400 `VALIDATION_FAILED`. 
    //
    //Future<JobRunListResponse> listJobs({ int limit }) async
    test('test listJobs', () async {
      // TODO
    });

    // List the registered models and the download run's state.
    //
    // The model management listing of the admin surface: every registered model of the instance with its facts, its activation state and the presence of its files, plus the most recent model download run as the summary the model page's progress bar renders.  Every model row of the `models` table appears, grouped by slot then id -- including a model whose weights went missing, which lists with `filesPresent: false` so the administrator can see and clean it up. `filesPresent` is a presence probe only (the on-disk manifest is readable and every file it pins exists); the sha256 verification itself is the activation's and the session loader's business, never the listing's. `license`, `notice` and `inputSize` come from the model's on-disk manifest, `dim` and `params` from the registered row (which the download and the scan keep in step with that manifest); `dim` and `inputSize` are null when the model declares none, `license` is empty and `notice` null when the manifest is unreadable; `isActive` comes from the table.  `download` is the most recent `model_download` run of whatever state -- the run the download endpoint's idempotence would return -- or null before the first download was ever started. 
    //
    //Future<ModelListResponse> listModels() async
    test('test listModels', () async {
      // TODO
    });

    // List the instance's accounts.
    //
    // The user management listing of the admin surface: every account of the instance with its role and its disabled switch, oldest first. The listing selects only the safe columns -- a password hash never appears in any admin response.  The list is small by construction (a family instance holds a handful of accounts) and grows only through this surface's own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. Account rows never disappear (deleting an account together with its data is a later disposal flow), so a refetch after any write reconciles the list completely.  The endpoint is admin-only: the session middleware authenticates the caller like every authed route, and a caller whose role is not `admin` answers 403 AUTH_FORBIDDEN -- a code distinct from the 401 family, because the caller is signed in, just not entitled to this surface. 
    //
    //Future<AdminUserListResponse> listUsers() async
    test('test listUsers', () async {
      // TODO
    });

    // Start a full place re-resolution run.
    //
    // The places-data-update path of the admin surface: one run that re-resolves the place of EVERY asset whose exif row carries GPS coordinates, against the baked place data the server currently runs on. The run that refills resolutions after the baked GeoNames data changed -- the startup import of a rebuilt image replaces the place tables and, through the schema's ON DELETE SET NULL, releases every stale resolution; this run is the deliberate step that refills them. (The metadata job resolves every NEW upload on its own; this run only matters for re-resolving what already exists.)  The scope is every GPS-bearing asset row whatever its lifecycle state -- place resolution is metadata, not visibility. `total` and `done` of the run's `job_runs` row count those assets and `failed` the ones whose re-resolution write failed.  The endpoint is idempotent the way the model rerun is: a place re-resolution run still in flight (`queued` or `running`) answers 200 with its identifier and starts nothing; a finished (or failed) run never blocks a new one.  The run is API-only for now: no admin page button is wired to it yet (the job panel's listing shows its progress like any other run). 
    //
    //Future<PlacesRerunResponse> reresolvePlaces() async
    test('test reresolvePlaces', () async {
      // TODO
    });

    // Start a manual rerun of the active model's slot.
    //
    // The rerun trigger of the admin surface: one run that redoes the slot's ML work over the whole library -- the run that fills the newly activated model's products, the manual step every model switch deliberately leaves to the administrator (a same-dimension switch keeps the old products usable immediately; a cross-dimension switch leaves the table empty until this run refills it).  Only the ACTIVE model of a slot can be rerun: products attach to the active model, so a rerun of a dormant model would fill rows no active-model query could see. The run's scope is every image asset that finished its pipeline and is not in the trash (ML work belongs after asset readiness, and video assets stay out of ML so far); `total` and `done` of the run's row count those assets and `failed` the ones whose rerun failed -- progress in asset counts, not bytes (the model download's row counts pinned bytes and failed models; each run's units are its own work's).  The endpoint is idempotent the way the download endpoint is: a rerun of the same slot still in flight (`queued` or `running`) answers 200 with its identifier and starts nothing. Unlike the download, a FINISHED run never blocks a new one -- the rerun is manual, so a second click after a completed (or failed) run starts a fresh run.  The per-asset ML workers of the slots arrive later; until a slot's dispatch is wired, a rerun of that slot still creates the run but the run fails (state `failed`, zero progress) instead of pretending to have done work it cannot do. 
    //
    //Future<ModelRerunResponse> rerunModel(String id) async
    test('test rerunModel', () async {
      // TODO
    });

    // Run the release check now.
    //
    // The manual check of the settings face: the server probes the configured release feed now (the same bounded, refused- internal-address, redirect-refusing request the scheduled check makes) and answers the fresh state -- the same document `GET` serves.  A failed probe answers 200 with `status: unknown`, exactly as the scheduled one would: a check that cannot reach its feed is the ordinary case of an offline instance, not a request error, and it carries no message. A check that is switched off (or disabled by the deployment) answers 200 with `disabled` and probes nothing. The probe is bounded server-side and well under the request timeout; a slow feed lands on `unknown` like any other failure. 
    //
    //Future<UpdateCheckResponse> runUpdateCheck() async
    test('test runUpdateCheck', () async {
      // TODO
    });

    // Register the custom models found in the models dir.
    //
    // The custom model registration of the admin surface: every directory under the instance's models dir that carries a `manifest.json` is verified against the digests the manifest pins and registered in the models table with source `custom`. Directories without a manifest are skipped -- the models dir also holds download staging, not every entry is a model.  Scanning registers, it never activates: an administrator reviews the registered models and switches them on through the activation endpoint. Re-scanning converges: the registration is an upsert on the descriptive fields and leaves every row's activation state alone. A directory whose files drifted from their pinned digests fails the whole scan with 400 MODEL_INVALID and registers nothing. 
    //
    //Future<ModelScanResponse> scanModels() async
    test('test scanModels', () async {
      // TODO
    });

    // Save the model download settings.
    //
    // The settings save. `autoDownload` is required (an omitted boolean would decode as off, silently changing the policy). The value takes effect on the next AI job's readiness check; no run is started or cancelled by the save itself. 
    //
    //Future<ModelSettingsResponse> updateModelSettings(ModelSettingsUpdateRequest modelSettingsUpdateRequest) async
    test('test updateModelSettings', () async {
      // TODO
    });

    // Validate and save the storage template.
    //
    // The storage settings save of the admin surface. The template passes a three-stage validation before anything is written:  1. **Compile** -- strict grammar: every `{{variable}}` must be a known member of the variable table, every conditional balanced. An unknown variable or a malformed conditional is rejected here. 2. **Sample render** -- the compiled template renders over a fixed set of sample assets (an album-less image, an album-grouped image, a video), exactly as an upload would: a template that renders an empty path (an album-only template over the album-less sample) is rejected here. 3. **Structure** -- the rendered paths must be safe: no `..` traversal, no dot segments, bounded depth (10) and bounded length, per-segment and total. The engine sanitizes the user-influenced values (file name, album, camera) and the structural pass guards the literal template text.  A template that passes is saved and answers 200 with the saved value, its sample render and the presets. A template that fails any stage answers 400 `VALIDATION_FAILED` naming `template` in the details, with the specific reason in the error message; nothing is written. The save takes effect for uploads from the next request on: paths already recorded are never re-derived and never moved (see GET for the full rendering contract). 
    //
    //Future<StorageSettingsResponse> updateStorageSettings(StorageSettingsUpdateRequest storageSettingsUpdateRequest) async
    test('test updateStorageSettings', () async {
      // TODO
    });

    // Save the map tile source setting.
    //
    // The settings save. `tilejsonUrl` is required: the empty string clears the source (the map falls back to its stated absence), any other value must be an absolute `http` or `https` URL.  The save validates the SHAPE only. The browser is the party that loads the source, so a LAN address is legitimate here (a self-hosted tile server inside the home network) and reachability is not judged at save time. The value takes effect on the map's next load -- there is no restart, and a map already open keeps the source it loaded. 
    //
    //Future<TilesSettingsResponse> updateTilesSettings(TilesSettingsUpdateRequest tilesSettingsUpdateRequest) async
    test('test updateTilesSettings', () async {
      // TODO
    });

    // Save the video transcoding settings.
    //
    // The transcoding settings save of the admin surface.  **The scope pin, documented honestly: the profile affects NEW ingests only.** The transcode job of a video upload reads the profile when it runs, so a saved change governs every video uploaded after it; videos already transcoded keep their existing playable copy and are never re-encoded, and a video whose job is still queued picks up the profile at its run time (the job reads the setting at run time, not enqueue time).  The fields: `preset` bounds the re-encode's vertical resolution (`auto` keeps the source resolution, the numeric presets scale down to the bound, `off` skips the re-encode entirely -- the original file stays in place and downloadable, and the video's pipeline still completes); `concurrency` bounds how many re-encodes run at once; `thumbnailFrame` turns the video poster-frame capture (the video's thumbnail) on or off; `keepOriginal` decides the failure policy -- a failed re-encode either completes the job quietly (the original stays in place and downloadable through the original endpoint) or fails it for retry.  The save validates every field against its enum and answers the saved profile; an out-of-enum value answers 400 VALIDATION_FAILED naming the field, and nothing is written. 
    //
    //Future<TranscodingSettingsResponse> updateTranscodingSettings(TranscodingSettingsUpdateRequest transcodingSettingsUpdateRequest) async
    test('test updateTranscodingSettings', () async {
      // TODO
    });

    // Save the trash retention setting.
    //
    // The trash settings save of the admin surface. The retention is a whole number of days, 0 to 365:  - **0 means immediate permanent delete** -- an asset enters the trash and the NEXT expiry run purges it (the hourly periodic job and the boot run are the runs; nothing is deleted by the save request itself). - The expiry job reads the setting at the START of each run (the boot run and every hourly periodic run alike), so a change takes effect for the next run's window, not retroactively for rows a past run already considered. - An already-trashed asset's fate is decided by the run's own deadline computation: with a SHORTENED retention, rows past the new window leave on the next run; with a LENGTHENED one, every row gains the added days from its own deletion mark.  The save validates the bound and answers the saved value; a value outside 0-365 answers 400 VALIDATION_FAILED naming `retentionDays`, and nothing is written. 
    //
    //Future<TrashSettingsResponse> updateTrashSettings(TrashSettingsUpdateRequest trashSettingsUpdateRequest) async
    test('test updateTrashSettings', () async {
      // TODO
    });

    // Save the release check settings.
    //
    // The settings save: all three fields are required (an omitted `enabled` would decode as the disabled switch, silently turning the check off), and each is validated before anything is written.  `endpoint` must be an `http` or `https` URL with a host -- the address the server will probe. The probe itself refuses addresses that resolve to loopback, private, link-local or otherwise internal ranges, and never follows a redirect: the setting can be saved pointing at an internal host, and the check simply stays `unknown` (the settings save does not probe, so it cannot judge reachability).  A failed validation answers 400 `VALIDATION_FAILED` naming the offending field, with nothing written. A saved value takes effect from the next scheduled check on; the manual check runs against the saved value immediately. 
    //
    //Future<UpdateCheckSettingsResponse> updateUpdateCheckSettings(UpdateCheckSettingsUpdateRequest updateCheckSettingsUpdateRequest) async
    test('test updateUpdateCheckSettings', () async {
      // TODO
    });

    // Toggle one account's disabled switch.
    //
    // The enable/disable switch of the admin surface: an administrator locks an account out of the instance (or lets it back in) without deleting the account or any of its data. The switch is enforced by the session middleware on EVERY request, which is what pins its observable semantics:  - Disabling takes effect with the account's next request: an   existing session of the account answers 401 from then on   (byte-identical with any other authentication failure) and   sign-in is refused the same way. The session rows   themselves are not deleted -- disabling revokes their   authority, not their existence. - Re-enabling restores the account's unexpired sessions   without a new sign-in; a session that lapsed its own   expiry while the account was disabled is gone for good.  The write is idempotent -- setting the current value again answers 200 with the unchanged account -- and the response is the account after the write, so the client adopts it as the fresh state without a refetch.  An administrator cannot disable their own account: the last admin locking themselves out would strand the admin surface, so the request answers 400 VALIDATION_FAILED naming the id. Nothing stops an admin from disabling ANOTHER administrator; the last enabled admin is the instance's line of defense.  The id must name an account. Account ids are enumerable through this surface's own listing, so there is no anti-enumeration pairing here: an unknown id (a stale list on the client) answers 404 USER_NOT_FOUND. 
    //
    //Future<AdminUserResponse> updateUser(String id, AdminUserPatch adminUserPatch) async
    test('test updateUser', () async {
      // TODO
    });

  });
}
