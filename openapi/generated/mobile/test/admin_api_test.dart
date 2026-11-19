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

    // Cancel one in-flight job run.
    //
    // The task panel's cancel (E41): asks the named run to end at its next chunk boundary. The request itself flags the run only -- the run's driver checks the flag BETWEEN chunks (a walk page, a rescan batch, one model of a download group) and closes the run `cancelled` there, so:  - The committed portion stays: every chunk already finished keeps its work on disk and its count in the run's `done`. The response makes no promise about work that has not committed yet. - An in-flight asset-level job (one decode, one inference, one encode) runs to its own end -- the boundary is between chunks, never mid-asset. - The run's state flips to `cancelled` when the driver closes it, and the close rides the task panel's existing events (`job.finished`) -- the listing converges through the same stream it always converges through. A QUEUED run cancels at once (its driver checks the flag before the first chunk). - A run that already finished -- succeeded, failed or cancelled -- answers 409 `JOB_NOT_CANCELLABLE`: nothing is in flight to cancel, and the row's own state is the honest answer. An unknown run id answers 404 `JOB_RUN_NOT_FOUND`.  The cancel takes no request body and answers no body: the 204 is the flag's whole receipt. 
    //
    //Future cancelJobRun(String runId) async
    test('test cancelJobRun', () async {
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

    // Delete one registered model.
    //
    // The removal of a model registration: the row is deleted and its model directory (weights plus manifest copy) is removed from the models dir. Only a model that is NOT its slot's active one can be deleted -- the active model of a slot is the pipeline's running configuration, and the removal of it is a switch first (the dialog's removal flow switches the slot back onto the official entry, then removes).  A rerun or other job still holding the model's products is unaffected: the products reference the model by id and the registry's foreign keys keep them resolvable, so a delete of a model whose products still exist answers 200 and leaves the products to their own lifecycle.  The model's files are removed from disk after the row is deleted; a directory removal failure logs and still answers 200 (the registration is gone; the leftover directory is cleaned by the next scan's refusal). 
    //
    //Future<DeleteModel200Response> deleteModel(String id) async
    test('test deleteModel', () async {
      // TODO
    });

    // Delete one account, disposing of its content.
    //
    // The account disposal of the admin surface: an account leaves the instance, and everything it held -- photos, albums, memories and the labels it gave them -- is settled before the row goes. The account row goes immediately: there is no soft delete and no recovery window for the account itself. The whole settlement is ONE transaction that commits whole or rolls back whole, so a failure leaves the account and its content exactly as they were.  THE DISPOSITION is exactly one of two arms:  - `transferTo` names another account that receives everything   the departing account owned: its photos with their albums   and memories, their archived and trash marks intact, and the   tags attached to them. The photos join the receiver's own   stream, so the family timeline continues without a gap, and   the albums the departing account shared keep their grantees.    COLLIDING CONTENT. The storage's uniqueness rule is   owner plus content checksum, and it is TOTAL: one account   holds at most one row of a given content whether that row is   live or in the trash. Content the receiver already holds --   in ANY of the four state pairs -- therefore cannot simply   change hands, and the disposal folds it instead:    - the LIVE copy wins a mixed pair, so content that arrives     live stays visible under the receiver's name, and a copy     the receiver had trashed earlier does not bind it (trash it     again to be rid of it) -- symmetrically, the receiver's     live copy survives a trashed departing one;   - a tie keeps the RECEIVER's own row: both live keeps its     copy, both trashed keeps it under the LATER of the two     deletion marks (the content stays recoverable as long as     either copy had left);   - the losing copy's album memberships, tags, favorites and     any album cover it pinned move onto the survivor, then the     losing row goes. A losing Live Photo takes its component     video row with it (the pair is one asset), and every losing     photo announces `asset.deleted`, so a client holding the id     lets it go.  - `dispose: \"trash\"` walks the departing account's photos into   the CALLING administrator's trash, where the instance's   retention window governs them -- restorable until it runs   out, permanently removed after. The departing account's own   albums and memories go with the account (a container whose   owner is gone has no reader), so the photos survive as   individual trash items; a directory album's source file is   moved into the server's trash area on this arm exactly like   any other trash move, and the rest of the family's source   files are never touched. The collision rule above applies   here too, onto the calling administrator: content they   already hold in any state folds rather than collides, and   their own live copy is never trashed by someone else's   disposal.  What dies with the account in BOTH arms: its sessions, API keys and pairing tokens (it can never authenticate again), its favorites, its per-album timeline preferences, the private names it gave shared face clusters (a cluster belongs to the family; the name a person gave it is theirs), and every album share granted TO it. A share the account granted on its own album rides the content: the album changes hands and its grantees keep their access. Authorship another member's album carries -- a membership the departing account added -- passes to the surviving owner instead of deleting the membership, so no photo leaves an album it sits in.  THE REFUSALS. The instance's last administrator cannot be deleted (409 USER_LAST_ADMIN): an instance without an administrator strands this surface, and while one administrator remains that account is necessarily the caller's own row. An administrator cannot delete their own account at all (400 VALIDATION_FAILED naming the id) -- the disposal hands the content to a survivor, and the caller is the one performing the handover. `transferTo` naming the departing account itself answers 409 USER_TRANSFER_TO_SELF, and an unknown receiver is a payload error (400 VALIDATION_FAILED naming the field). Account ids are enumerable through this surface's own listing, so an unknown path id answers 404 USER_NOT_FOUND without an anti-enumeration pairing.  The 204 carries no body: the account is gone, and the caller's own disposal target (the receiver's stream or the caller's trash) is where its content answers next. 
    //
    //Future deleteUser(String id, AdminUserDeleteRequest adminUserDeleteRequest) async
    test('test deleteUser', () async {
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

    // Read the derivative library's size and rebuild estimate.
    //
    // The image card's cost line: how many image assets the library carries and how long a full derivative rebuild would take, as the instance itself estimates it.  THE ESTIMATE IS DECLARED, NOT MEASURED: the server multiplies the image count by one second per asset (one decode plus the tile and view encodes at the shipped settings), divides by the concurrency the run-budget dial currently maps, and rounds up to whole minutes, with one minute the floor. The answer is the plan-of-record number the dialog quotes back before the rebuild starts, never a promise: NAS-class storage, the encoding format and the dial all move it.  The count covers IMAGE assets only: the rebuild re-derives the three serving sizes of images; video assets carry a poster frame the rebuild does not touch. 
    //
    //Future<DerivativesSummaryResponse> getDerivativesSummary() async
    test('test getDerivativesSummary', () async {
      // TODO
    });

    // Read the derivative image settings.
    //
    // The image card's document: the sizes and encoding every NEW derivation runs at. Five knobs, the honest set --  - `tileSize`: the timeline tile's bounding box, 250 (the   shipped default) or 320 (sharper on high-density screens). - `previewSize`: the viewer's inline preview long edge, 1080   (the shipped default), 1440 or 2048. - `format`: the serving copies' encoding, `avif` (smaller,   slower to encode) or `webp` (faster, a little larger). On a   weak machine AVIF makes first browsing slower; the card says   so beside the knob. - `fullSize`: whether the thumbnail job also derives the   full-resolution serving copy (shipped OFF: the viewer reads   the preview, which is enough to look at and cheaper to   store). - `rawEmbedded`: whether a RAW source derives from its   embedded preview (shipped ON).  THE COST STATEMENT IS THE CARD'S, NOT THIS DOCUMENT'S: saving these values recomputes nothing. Derivatives already on disk stay exactly as they are; only assets derived AFTER the save (new uploads, and the explicit rebuild run) use the new values. A rebuild mid-flight keeps the parameters it started with.  The color space (P3), progressive encoding and the per-size quality are deliberately NOT knobs: a family instance does not need twenty dials.  A corrupt settings row answers the defaults with the corruption logged server-side. 
    //
    //Future<ImageSettingsResponse> getImageSettings() async
    test('test getImageSettings', () async {
      // TODO
    });

    // Read one model source probe's result.
    //
    // The probe the switch dialog polls. A source probe -- the download-to-staging of a URL, or the accept of an uploaded file, then the sha256 check when one was given, the sandbox load of the ONNX graph and the slot contract's shape and dimension checks -- runs in the server's background after the probe POST answered 202; this read answers the probe's current state, whatever it is: running (with its download progress), ok (with the facts the dialog renders), error (with the reason that names the exact check that failed) or warn (the checks passed, the host is not on the download sources list, and the save may still proceed).  A probe is transient: its record lives for a few minutes after it settles, then it is gone -- an unknown or expired probeId answers 404 and the dialog simply probes again. 
    //
    //Future<ModelProbeResponse> getModelProbe(String probeId) async
    test('test getModelProbe', () async {
      // TODO
    });

    // Read the model download settings.
    //
    // The model store's policy switch, download sources and pipeline params: whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own, where the bytes may travel, and the two steering params the ML jobs read at their own dispatch (see `ModelParams`).  WHAT AUTO-FILLS: only a slot that already has an active model whose files are gone from the store -- the case of a store wiped below a configured instance. A slot that was never configured stays empty; the first-run download and the settings face's download button remain the administrator's active path, and an unconfigured slot keeps the honest no-AI degraded mode.  A corrupt settings row answers the defaults (the policy on, the shipped hosts, the pipeline's own params) with the corruption logged server-side. 
    //
    //Future<ModelSettingsResponse> getModelSettings() async
    test('test getModelSettings', () async {
      // TODO
    });

    // Read the loaded place database and the re-resolution scopes' sizes.
    //
    // The map & places card's read: the place database the instance currently resolves against, plus the size of the two re-resolution scopes the rebuild dialog chooses between.  `version` is the data-version marker the startup import wrote. When the marker is a 64-character digest it is a content digest, not a release date: the card displays its first eight characters. An empty string means the instance carries no bake -- the place tables were never imported, so nothing resolves.  `languages` is the display-name locale set the bake carries (the same set the places read's `locale` parameter accepts).  The counts are the re-resolution walk's scope sizes: `gpsAssets` counts every GPS-bearing asset row the pipeline owns and `missingPlaces` the subset whose place is absent -- exactly what a full run and a missing-only run walk. A row the owner located by hand (`location_lat` set) is outside both: that manual decision owns the row, so no count promises work the walk will not do. Place resolution is metadata, not visibility, so the counts are library-wide, never per caller. 
    //
    //Future<PlacesDatabaseResponse> getPlacesDatabase() async
    test('test getPlacesDatabase', () async {
      // TODO
    });

    // Read the background work's budget.
    //
    // The task panel's run-settings document: the three dials that decide WHEN the background work spends the machine's capacity --  - `concurrencyPreset`: the dial's three steps, `light`   (one job at a time), `standard` (half the cores) and `full`   (all but one core). The panel never asks for a NUMBER: a   NAS owner does not know what to type, so the dial speaks in   steps and the mapping is the honest explanation line. - `nightlyEnabled` with `nightlyStart`/`nightlyEnd`: the night   window (shipped 02:00-06:00, off). While it is on, the HEAVY   work -- derivative rebuilds, video transcodes, model   recomputation -- runs only inside it; a window that ends   stops the work at the asset it just finished and the next   window resumes from there. A window that spans midnight   (start after end) is a legal shape. Uploads and browsing   are NEVER held by the window -- they are foreground work,   not background jobs. - `pausedJobs`: the five per-kind pause switches   (thumbnails, metadata, transcode, machine learning, place   resolution). A paused kind's NEW jobs sit held; the one   already running finishes. Uploads of a paused kind enqueue   without being blocked -- the request never waits, the job   does.  Both the concurrency preset and the pauses take effect immediately: every job reads this document when it starts, so a save changes what the NEXT job sees, and a held job wakes within the hold's own granularity to re-read it.  A corrupt settings row answers the defaults with the corruption logged server-side. 
    //
    //Future<RunBudgetSettingsResponse> getRunBudgetSettings() async
    test('test getRunBudgetSettings', () async {
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
    // The map's tile sources, as the administrator configured them: the light style every theme loads as its fallback, and the optional dark style the dark theme loads instead. Either is the empty string when the instance has none.  THE SHAPE: each URL names a document the browser's map library can build a base layer from -- a style document (the vector case: sources, layers, sprites and glyphs) or a TileJSON descriptor (the raster case: the tile templates and their attribution). The map detects which of the two it received and builds the base layer accordingly, so each field covers both the raster and the vector-and-style forms of the self-hosting ecosystem.  THE DARK STYLE IS OPTIONAL: an empty `darkTilejsonUrl` means the dark theme reuses the light source, so a single source keeps a single look in both themes. A configured one is loaded by the dark theme only; the light theme always loads `tilejsonUrl`. A row written before the field existed reads as the empty string.  THE EMPTY VALUE IS THE SHIPPED DEFAULT and a real state: no source ships with the product (a public tile service's usage policy forbids carrying the traffic of every install), so a fresh instance draws its map with the tile layer absent and says so. Loading the document needs the host to send the usual cross-origin headers; a host that does not lands on the same stated absence.  A corrupt settings row answers the empty strings with the corruption logged server-side. 
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
    // The model management listing of the admin surface: every registered model of the instance with its facts, its activation state and the presence of its files, plus the most recent model download run as the summary the model page's progress bar renders.  Every model row of the `models` table appears, grouped by slot then id -- including a model whose weights went missing, which lists with `filesPresent: false` so the administrator can see and clean it up. `filesPresent` is a presence probe only (the on-disk manifest is readable and every file it pins exists); the sha256 verification itself is the activation's and the session loader's business, never the listing's. `license`, `notice` and `inputSize` come from the model's on-disk manifest, `dim` and `params` from the registered row (which the download and the scan keep in step with that manifest); `dim` and `inputSize` are null when the model declares none, `license` is empty and `notice` null when the manifest is unreadable; `isActive` comes from the table.  `download` is the most recent `model_download` run of whatever state -- the run the download endpoint's idempotence would return -- or null before the first download was ever started.  `slots` is the per-slot state view the page's cards render: one row per pipeline slot, each carrying one of the four states the cards speak (`active`, `downloading`, `not_downloaded`, `failed`), the slot's active model, its source facts (official or custom, the host, the digest, the download time) and the download run's progress while a run is in flight. `official` is the manifest's catalog -- the validated list the switch dialog's official group renders, every entry with its download state (`downloaded` is the presence probe, the digest check stays the activation's). 
    //
    //Future<ModelListResponse> listModels() async
    test('test listModels', () async {
      // TODO
    });

    // List the instance's accounts.
    //
    // The user management listing of the admin surface: every account of the instance with its role and its disabled switch, oldest first. The listing selects only the safe columns -- a password hash never appears in any admin response.  The list is small by construction (a family instance holds a handful of accounts) and grows only through this surface's own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. Account rows disappear only through this surface's own delete endpoint, whose disposal moves the account's content to a surviving owner or into the trash before the row goes, so a refetch after any write reconciles the list completely.  The endpoint is admin-only: the session middleware authenticates the caller like every authed route, and a caller whose role is not `admin` answers 403 AUTH_FORBIDDEN -- a code distinct from the 401 family, because the caller is signed in, just not entitled to this surface. 
    //
    //Future<AdminUserListResponse> listUsers() async
    test('test listUsers', () async {
      // TODO
    });

    // Probe a model source for one slot.
    //
    // The first half of a source import: download (or read) the candidate model, check it against the slot's contract, and stage it -- without touching the slot's active model. The answer is 202 with a probeId; the probe itself runs in the server's background (a URL download legitimately takes minutes) and is read through the probe GET until it settles.  The body is `multipart/form-data` with these parts:  - `kind` (text, required) -- `url` or `file`. - `url` (text, required for kind `url`) -- the model URL; the   download lands in the slot's staging area, its bytes are   never trusted before the digest check. - `sha256` (text, optional) -- a digest the bytes must match;   without it the probe still computes and reports the digest,   and the save marks the model an unverified source. - `file` (file, required for kind `file`) -- the ONNX bytes   (up to 2 GiB; the sniff reads the content, never the file   name).  The checks, in order, every failure naming its own check: the digest when one was given; the graph loads (a file that is not a readable ONNX model is refused); the input shape and the output dimension against the slot's contract -- a mismatch is refused naming both numbers. A host that is not on the download sources list does NOT refuse: the probe answers `warn` (the checks passed, the host is undeclared, the save is still the administrator's own act and its use is logged).  The probe does not register anything: the save endpoint takes the probeId, and a probe whose checks failed can never be saved. 
    //
    //Future<ModelProbeAcceptedResponse> probeModelSource(ModelSlot slot, String kind, { String url, String sha256, MultipartFile file }) async
    test('test probeModelSource', () async {
      // TODO
    });

    // Start a derivative rebuild run.
    //
    // The image card's explicit rebuild: one run that re-derives the serving copies of every image asset at the settings the instance currently carries.  THE COST IS NEVER SILENT: the image card states what the settings change (nothing is recomputed until this run is asked for), and the two scopes are the honest pair --  - `missing` derives only the assets whose current-format   serving files are absent: a library that has never been   rebuilt, or one that gained the full-size copy, pays only   for what it lacks (minutes on a family library). - `all` re-derives every image asset at the current settings   and removes the other encoding format's stale files: the CPU   belongs to the rebuild for the estimate's duration, and   browsing is slower while it runs.  THE RUN IS A JOB: it lists on the task panel with the four states, its progress counts the image assets it visited, and the pause and window rules that govern thumbnail-class work hold it the same way -- a night window that closes stops it at the asset it just finished, and the next window resumes from there, never from zero.  The endpoint is idempotent the way the place re-resolution is: a rebuild run still in flight (`queued` or `running`) answers 200 with its identifier and starts nothing; a finished run never blocks a new one.  SETTINGS CHANGES NEVER TRIGGER THIS: saving the image settings writes the document only -- the serving copies already on disk stay as they are until an explicit rebuild re-derives them. 
    //
    //Future<RebuildDerivativesResponse> rebuildDerivatives(RebuildDerivativesRequest rebuildDerivativesRequest) async
    test('test rebuildDerivatives', () async {
      // TODO
    });

    // Start a place re-resolution run over the chosen mode.
    //
    // The places-data-update path of the admin surface: one run that re-resolves the place of the assets its mode covers, against the baked place data the server currently runs on. The run that refills resolutions after the baked GeoNames data changed -- the startup import of a rebuilt image replaces the place tables and, through the schema's ON DELETE SET NULL, releases every stale resolution; this run is the deliberate step that refills them. (The metadata job resolves every NEW upload on its own; this run only matters for re-resolving what already exists.)  The mode is the dialog's honest pair: `missing` walks only the GPS-bearing rows whose place is absent (the fill-in path, which rewrites nothing that already resolved), and `all` walks every GPS-bearing row (the full re-resolve, which rewrites history -- boundaries that moved change the city). An absent body means `all`, the endpoint's original behavior; any other value is 400 VALIDATION_FAILED.  The scope's rows are counted whatever their lifecycle state -- place resolution is metadata, not visibility. A row the owner located by hand is outside both modes (the manual decision owns it, no walk rewrites it), and the enqueue counts the same rows the walk visits, so `total` and `done` converge. `failed` counts the rows whose re-resolution write failed.  The endpoint is idempotent the way the model rerun is: a place re-resolution run still in flight (`queued` or `running`) answers 200 with its identifier and starts nothing; a finished (or failed) run never blocks a new one. The in-flight join is per kind, not per mode: a `missing` run in flight answers a request for `all` with 200 as well (v1 does not queue modes; the card points at the jobs page). 
    //
    //Future<PlacesRerunResponse> reresolvePlaces({ PlacesRerunRequest placesRerunRequest }) async
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

    // Stop the in-flight derivative rebuild run.
    //
    // The task panel's stop for a rebuild in flight: the run's row is marked as asked-to-stop, and the driver -- which checks the flag between assets -- closes the run at the asset it just finished, never mid-asset. The work already done stays on disk: a stopped rebuild has paid exactly for the assets it re-derived, and the next rebuild's `missing` scope skips them.  The run closes `cancelled` -- the task panel renders the cancelled state, not a broken run: the admin asked for this ending (E41 shared the flag-closed ending with the task panel's per-run cancel, and the two speak one state).  A stop with no rebuild in flight (the run finished between the panel's listing and the click, or none ever started) answers 409 `REBUILD_NOT_RUNNING`: the flag has nothing to land on. 
    //
    //Future<RebuildDerivativesResponse> stopDerivativesRebuild() async
    test('test stopDerivativesRebuild', () async {
      // TODO
    });

    // Switch a slot onto a probed source.
    //
    // The second half of a source import, and the official list's switch: the slot is switched onto the source the request names, under the slot's lock. The answer is 202 -- the switch itself is seconds, the recalculation its semantics demand is an async run the task panel owns.  The body depends on the kind:  - `builtin` (the official list): `modelId` names the manifest   entry; nothing was probed because the entry is the   validated list. A cross-dimension switch of the embed slot   requires `confirmWipe: true` -- the confirmation is the   dialog's contract, and the response reports the wipe's facts   exactly as the activation switch does. - `url` or `file` (a custom source): `probeId` names a probe   that finished OK -- the server never reuses an unverified   result, a missing or failed probe answers 400. The model is   registered from the staged bytes (same sha256 reuse: bytes   already on disk are never stored twice), activated, and the   slot's recalculation run is enqueued.  The slot lock: a switch stops being possible while one is in flight (409) -- the same-dimension switch is seconds, but the recalculation it started is the slot's business, and two overlapping switches would race its products. The switch refuses to lose the named persons: a face slot's recalculation inherits the cluster names by overlap, so the confirm copy states what is kept; nothing here deletes a user label.  A failed switch changes nothing: the slot keeps its previous model and its products. 
    //
    //Future<ModelSourceSwitchResponse> switchModelSource(ModelSlot slot, ModelSourceSwitchRequest modelSourceSwitchRequest) async
    test('test switchModelSource', () async {
      // TODO
    });

    // Save the derivative image settings.
    //
    // The settings save. Every field is contract-required (an omitted one would decode as its zero value and silently change the posture), and each value is checked against its enum: the tile sizes 250 and 320, the preview sizes 1080, 1440 and 2048, the encodings `avif` and `webp`.  The save writes the document and nothing else: no derivation starts, no existing serving file is touched, and a rebuild already running keeps its own starting parameters. The new values bind at the next derivation -- a new upload's thumbnail job, or the explicit rebuild run the image card starts.  The save takes effect without a restart: the thumbnail job reads the settings at run time, so an upload that arrives after the save derives at them. 
    //
    //Future<ImageSettingsResponse> updateImageSettings(ImageSettingsUpdateRequest imageSettingsUpdateRequest) async
    test('test updateImageSettings', () async {
      // TODO
    });

    // Save the model download settings and download sources.
    //
    // The settings save. `autoDownload`, `allowedHosts`, `mirrorBaseUrl` and `params` are required (an omitted field would decode as its zero value, silently changing the policy): the caller saves the whole object it read.  `allowedHosts` replaces the stored list wholesale; every entry must be a hostname, optionally with a leading `*.` wildcard label and an optional port. `mirrorBaseUrl`, when non-empty, must be a host (or a full http/https URL whose host is kept); its host joins `allowedHosts` automatically, so saving a mirror never leaves it refusing its own first hop. `params` carries the two pipeline steering values, each validated against the closed 0-1 domain -- a value outside it answers 400 `VALIDATION_FAILED` naming the field, and nothing is written.  The values take effect on the next dispatch (the download's next run; the ML jobs' next per-asset enqueue) -- no run is started or cancelled by the save itself, and work already in flight runs to completion on the values it read. 
    //
    //Future<ModelSettingsResponse> updateModelSettings(ModelSettingsUpdateRequest modelSettingsUpdateRequest) async
    test('test updateModelSettings', () async {
      // TODO
    });

    // Save the background work's budget.
    //
    // The budget save. Every field is contract-required (an omitted one would decode as its zero value and silently change the posture), and each value is checked: the concurrency preset against its three-step enum, the two times against the `HH:MM` form, and the five pause switches against booleans.  The two times must differ: an equal pair would define a zero-length window, which reads as a window that never opens -- turning the window OFF is the honest way to say that, so the save refuses the ambiguous shape.  The save takes effect immediately, without a restart: the next job a worker picks up reads the fresh document, and a job already held wakes within its hold granularity and re-reads. The switches never touch a job already running -- the one in flight finishes, which is the card's own headline.  The window and the pauses NEVER gate an upload or a serving request: they are worker-side facts. The upload path enqueues without consulting them, and the browser's reads are answered whatever the budget says. 
    //
    //Future<RunBudgetSettingsResponse> updateRunBudgetSettings(RunBudgetSettingsUpdateRequest runBudgetSettingsUpdateRequest) async
    test('test updateRunBudgetSettings', () async {
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
    // The settings save. `tilejsonUrl` is required: the empty string clears the light source (the map falls back to its stated absence), any other value must be an absolute `http` or `https` URL. `darkTilejsonUrl` is the optional dark style: omitting it or sending the empty string clears it, which makes the dark theme reuse the light source, and any other value must be an absolute `http` or `https` URL as well.  The save replaces both fields: it is one document, not two partial writes. The validation is symmetric -- a bad dark URL is refused with the field named, and nothing is written.  The save validates the SHAPE only. The browser is the party that loads the source, so a LAN address is legitimate here (a self-hosted tile server inside the home network) and reachability is not judged at save time. The value takes effect on the map's next load -- there is no restart, and a map already open keeps the source it loaded. 
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
