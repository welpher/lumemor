# lumemor_api.api.AdminApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**activateModel**](AdminApi.md#activatemodel) | **PATCH** /admin/models/{id} | Activate the model (the model switch).
[**createUser**](AdminApi.md#createuser) | **POST** /admin/users | Create one account.
[**deleteModel**](AdminApi.md#deletemodel) | **DELETE** /admin/models/{id} | Delete one registered model.
[**downloadModels**](AdminApi.md#downloadmodels) | **POST** /admin/models/download | Start the default model group&#39;s download.
[**getDerivativesSummary**](AdminApi.md#getderivativessummary) | **GET** /admin/derivatives/summary | Read the derivative library&#39;s size and rebuild estimate.
[**getImageSettings**](AdminApi.md#getimagesettings) | **GET** /admin/settings/image | Read the derivative image settings.
[**getModelProbe**](AdminApi.md#getmodelprobe) | **GET** /admin/models/probes/{probeId} | Read one model source probe&#39;s result.
[**getModelSettings**](AdminApi.md#getmodelsettings) | **GET** /admin/settings/models | Read the model download settings.
[**getRunBudgetSettings**](AdminApi.md#getrunbudgetsettings) | **GET** /admin/settings/run-budget | Read the background work&#39;s budget.
[**getStorageSettings**](AdminApi.md#getstoragesettings) | **GET** /admin/settings/storage | Read the storage template setting.
[**getTilesSettings**](AdminApi.md#gettilessettings) | **GET** /admin/settings/tiles | Read the map tile source setting.
[**getTranscodingSettings**](AdminApi.md#gettranscodingsettings) | **GET** /admin/settings/transcoding | Read the video transcoding settings.
[**getTrashSettings**](AdminApi.md#gettrashsettings) | **GET** /admin/settings/trash | Read the trash retention setting.
[**getUpdateCheck**](AdminApi.md#getupdatecheck) | **GET** /server/update-check | Read the instance&#39;s release check state.
[**getUpdateCheckSettings**](AdminApi.md#getupdatechecksettings) | **GET** /admin/settings/update-check | Read the release check settings.
[**listJobs**](AdminApi.md#listjobs) | **GET** /admin/jobs | List the job runs of the task panel.
[**listModels**](AdminApi.md#listmodels) | **GET** /admin/models | List the registered models and the download run&#39;s state.
[**listUsers**](AdminApi.md#listusers) | **GET** /admin/users | List the instance&#39;s accounts.
[**probeModelSource**](AdminApi.md#probemodelsource) | **POST** /admin/models/{slot}/probe | Probe a model source for one slot.
[**rebuildDerivatives**](AdminApi.md#rebuildderivatives) | **POST** /admin/derivatives/rebuild | Start a derivative rebuild run.
[**reresolvePlaces**](AdminApi.md#reresolveplaces) | **POST** /admin/places/reresolve | Start a full place re-resolution run.
[**rerunModel**](AdminApi.md#rerunmodel) | **POST** /admin/models/{id}/rerun | Start a manual rerun of the active model&#39;s slot.
[**runUpdateCheck**](AdminApi.md#runupdatecheck) | **POST** /server/update-check | Run the release check now.
[**scanModels**](AdminApi.md#scanmodels) | **POST** /admin/models/scan | Register the custom models found in the models dir.
[**stopDerivativesRebuild**](AdminApi.md#stopderivativesrebuild) | **POST** /admin/derivatives/rebuild/stop | Stop the in-flight derivative rebuild run.
[**switchModelSource**](AdminApi.md#switchmodelsource) | **POST** /admin/models/{slot}/source | Switch a slot onto a probed source.
[**updateImageSettings**](AdminApi.md#updateimagesettings) | **PUT** /admin/settings/image | Save the derivative image settings.
[**updateModelSettings**](AdminApi.md#updatemodelsettings) | **PUT** /admin/settings/models | Save the model download settings and download sources.
[**updateRunBudgetSettings**](AdminApi.md#updaterunbudgetsettings) | **PUT** /admin/settings/run-budget | Save the background work&#39;s budget.
[**updateStorageSettings**](AdminApi.md#updatestoragesettings) | **PUT** /admin/settings/storage | Validate and save the storage template.
[**updateTilesSettings**](AdminApi.md#updatetilessettings) | **PUT** /admin/settings/tiles | Save the map tile source setting.
[**updateTranscodingSettings**](AdminApi.md#updatetranscodingsettings) | **PUT** /admin/settings/transcoding | Save the video transcoding settings.
[**updateTrashSettings**](AdminApi.md#updatetrashsettings) | **PUT** /admin/settings/trash | Save the trash retention setting.
[**updateUpdateCheckSettings**](AdminApi.md#updateupdatechecksettings) | **PUT** /admin/settings/update-check | Save the release check settings.
[**updateUser**](AdminApi.md#updateuser) | **PATCH** /admin/users/{id} | Toggle one account&#39;s disabled switch.


# **activateModel**
> ModelActivateResponse activateModel(id, requestBody)

Activate the model (the model switch).

The activation switch of the admin surface: the model becomes the active one of its slot. The request body is an empty JSON object -- the path's model identifier is the whole request.  A same-slot switch runs in one transaction that only flips the activation flags: the previous model's products stay in the database untouched (multi-generation embeddings and faces coexist by their model columns), queries that filter on the active model stop seeing them, and switching back revives them with zero recompute -- but the newly activated model does nothing until its products are rerun.  Cross-dimension switches of the clip-embed slot are the one exception, because a vector column cannot hold two dimensions: activating a clip model whose dimension differs from the `clip_embeddings` column's current one (read from the database catalog, never assumed) clears the table and re-types the column to `vector(N)` inside the SAME transaction as the flag flip -- the wipe, the new dimension and the activation land together or not at all, and a failure rolls everything back. The response then reports the switch's facts (`dimensionChanged` with the from and to dimensions, `wipedEmbeddings` with the number of deleted rows) so the admin UI can demand the explicit confirmation a destructive switch deserves. The rerun that fills the new model's embeddings stays manual: nothing is recomputed here.  The model's files are verified against their pinned sha256 digests before anything flips: a model whose files are missing or drifted answers 400 MODEL_INVALID and activates nothing. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final String id = cnclip-vit-b16-vision; // String | The identifier of the model to activate (the models table's slug id, as the listing returns it). 
final BuiltMap<String, JsonObject?> requestBody = Object; // BuiltMap<String, JsonObject?> | 

try {
    final response = api.activateModel(id, requestBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->activateModel: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| The identifier of the model to activate (the models table's slug id, as the listing returns it).  | 
 **requestBody** | [**BuiltMap&lt;String, JsonObject?&gt;**](JsonObject.md)|  | 

### Return type

[**ModelActivateResponse**](ModelActivateResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **createUser**
> AdminUserResponse createUser(adminCreateUserRequest)

Create one account.

The account creation of the admin surface, the management counterpart of the first-run wizard: an administrator provisions another account -- a family member's login, or a second administrator. The account rules mirror the wizard's exactly: the email is parsed and lower-cased as the login identity, the password is stored only as an Argon2id hash and never appears in any response, the id is a fresh UUIDv7, and the account's sync album -- the core backup stream every upload lands in and the timeline aggregates -- is created in the SAME transaction, so the new account can take uploads the moment it exists (a second creation point of the same wiring the wizard uses). The wizard's one-time guarantee is NOT part of this endpoint; it exists exactly because an administrator already does.  The role defaults to `user` when omitted; creating another administrator is the same request with `role: \"admin\"`. A duplicate email answers 409 USER_EMAIL_TAKEN -- the address is the account identity, unique instance-wide, and the login lookup depends on that uniqueness. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final AdminCreateUserRequest adminCreateUserRequest = {"name":"Grace Hopper","email":"grace@example.com","password":"correct horse battery staple"}; // AdminCreateUserRequest | 

try {
    final response = api.createUser(adminCreateUserRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->createUser: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **adminCreateUserRequest** | [**AdminCreateUserRequest**](AdminCreateUserRequest.md)|  | 

### Return type

[**AdminUserResponse**](AdminUserResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteModel**
> DeleteModel200Response deleteModel(id)

Delete one registered model.

The removal of a model registration: the row is deleted and its model directory (weights plus manifest copy) is removed from the models dir. Only a model that is NOT its slot's active one can be deleted -- the active model of a slot is the pipeline's running configuration, and the removal of it is a switch first (the dialog's removal flow switches the slot back onto the official entry, then removes).  A rerun or other job still holding the model's products is unaffected: the products reference the model by id and the registry's foreign keys keep them resolvable, so a delete of a model whose products still exist answers 200 and leaves the products to their own lifecycle.  The model's files are removed from disk after the row is deleted; a directory removal failure logs and still answers 200 (the registration is gone; the leftover directory is cleaned by the next scan's refusal). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final String id = my-clip-980020bf; // String | The identifier of the model to delete.

try {
    final response = api.deleteModel(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->deleteModel: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| The identifier of the model to delete. | 

### Return type

[**DeleteModel200Response**](DeleteModel200Response.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **downloadModels**
> ModelDownloadResponse downloadModels()

Start the default model group's download.

The model download trigger of the admin surface: the default group's models -- pinned by the manifest with upstream URLs and sha256 digests -- are downloaded, verified and registered, and each model activates into its slot when the slot is still empty. The request body is deliberately empty: the default group is the only group v1 downloads.  The endpoint is idempotent. A run that is still in flight (state `queued` or `running`) answers 200 with its identifier and nothing new is started; so does the most recent finished run -- a repeated click neither re-downloads a landed group nor piles up jobs, and the recovery of a failed run belongs to the job's own retries. Only a first call -- no run of this kind at all -- creates one and answers 202.  The run's progress is observable two ways: the `job_runs` row of the returned identifier (kind `model_download`, `total` and `done` carrying the group's pinned byte counts) and the `job.started` / `job.progress` / `job.finished` events on the activity stream, the progress events throttled to at most one per second. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.downloadModels();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->downloadModels: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**ModelDownloadResponse**](ModelDownloadResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getDerivativesSummary**
> DerivativesSummaryResponse getDerivativesSummary()

Read the derivative library's size and rebuild estimate.

The image card's cost line: how many image assets the library carries and how long a full derivative rebuild would take, as the instance itself estimates it.  THE ESTIMATE IS DECLARED, NOT MEASURED: the server multiplies the image count by one second per asset (one decode plus the tile and view encodes at the shipped settings), divides by the concurrency the run-budget dial currently maps, and rounds up to whole minutes, with one minute the floor. The answer is the plan-of-record number the dialog quotes back before the rebuild starts, never a promise: NAS-class storage, the encoding format and the dial all move it.  The count covers IMAGE assets only: the rebuild re-derives the three serving sizes of images; video assets carry a poster frame the rebuild does not touch. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getDerivativesSummary();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getDerivativesSummary: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**DerivativesSummaryResponse**](DerivativesSummaryResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getImageSettings**
> ImageSettingsResponse getImageSettings()

Read the derivative image settings.

The image card's document: the sizes and encoding every NEW derivation runs at. Five knobs, the honest set --  - `tileSize`: the timeline tile's bounding box, 250 (the   shipped default) or 320 (sharper on high-density screens). - `previewSize`: the viewer's inline preview long edge, 1080   (the shipped default), 1440 or 2048. - `format`: the serving copies' encoding, `avif` (smaller,   slower to encode) or `webp` (faster, a little larger). On a   weak machine AVIF makes first browsing slower; the card says   so beside the knob. - `fullSize`: whether the thumbnail job also derives the   full-resolution serving copy (shipped OFF: the viewer reads   the preview, which is enough to look at and cheaper to   store). - `rawEmbedded`: whether a RAW source derives from its   embedded preview (shipped ON).  THE COST STATEMENT IS THE CARD'S, NOT THIS DOCUMENT'S: saving these values recomputes nothing. Derivatives already on disk stay exactly as they are; only assets derived AFTER the save (new uploads, and the explicit rebuild run) use the new values. A rebuild mid-flight keeps the parameters it started with.  The color space (P3), progressive encoding and the per-size quality are deliberately NOT knobs: a family instance does not need twenty dials.  A corrupt settings row answers the defaults with the corruption logged server-side. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getImageSettings();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getImageSettings: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**ImageSettingsResponse**](ImageSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getModelProbe**
> ModelProbeResponse getModelProbe(probeId)

Read one model source probe's result.

The probe the switch dialog polls. A source probe -- the download-to-staging of a URL, or the accept of an uploaded file, then the sha256 check when one was given, the sandbox load of the ONNX graph and the slot contract's shape and dimension checks -- runs in the server's background after the probe POST answered 202; this read answers the probe's current state, whatever it is: running (with its download progress), ok (with the facts the dialog renders), error (with the reason that names the exact check that failed) or warn (the checks passed, the host is not on the download sources list, and the save may still proceed).  A probe is transient: its record lives for a few minutes after it settles, then it is gone -- an unknown or expired probeId answers 404 and the dialog simply probes again. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final String probeId = 0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10; // String | The probe identifier the POST answered.

try {
    final response = api.getModelProbe(probeId);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getModelProbe: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **probeId** | **String**| The probe identifier the POST answered. | 

### Return type

[**ModelProbeResponse**](ModelProbeResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getModelSettings**
> ModelSettingsResponse getModelSettings()

Read the model download settings.

The model store's one policy switch: whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own.  WHAT AUTO-FILLS: only a slot that already has an active model whose files are gone from the store -- the case of a store wiped below a configured instance. A slot that was never configured stays empty; the first-run download and the settings face's download button remain the administrator's active path, and an unconfigured slot keeps the honest no-AI degraded mode.  A corrupt settings row answers the default (on) with the corruption logged server-side. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getModelSettings();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getModelSettings: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**ModelSettingsResponse**](ModelSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getRunBudgetSettings**
> RunBudgetSettingsResponse getRunBudgetSettings()

Read the background work's budget.

The task panel's run-settings document: the three dials that decide WHEN the background work spends the machine's capacity --  - `concurrencyPreset`: the dial's three steps, `light`   (one job at a time), `standard` (half the cores) and `full`   (all but one core). The panel never asks for a NUMBER: a   NAS owner does not know what to type, so the dial speaks in   steps and the mapping is the honest explanation line. - `nightlyEnabled` with `nightlyStart`/`nightlyEnd`: the night   window (shipped 02:00-06:00, off). While it is on, the HEAVY   work -- derivative rebuilds, video transcodes, model   recomputation -- runs only inside it; a window that ends   stops the work at the asset it just finished and the next   window resumes from there. A window that spans midnight   (start after end) is a legal shape. Uploads and browsing   are NEVER held by the window -- they are foreground work,   not background jobs. - `pausedJobs`: the five per-kind pause switches   (thumbnails, metadata, transcode, machine learning, place   resolution). A paused kind's NEW jobs sit held; the one   already running finishes. Uploads of a paused kind enqueue   without being blocked -- the request never waits, the job   does.  Both the concurrency preset and the pauses take effect immediately: every job reads this document when it starts, so a save changes what the NEXT job sees, and a held job wakes within the hold's own granularity to re-read it.  A corrupt settings row answers the defaults with the corruption logged server-side. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getRunBudgetSettings();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getRunBudgetSettings: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**RunBudgetSettingsResponse**](RunBudgetSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getStorageSettings**
> StorageSettingsResponse getStorageSettings()

Read the storage template setting.

The storage settings read of the admin surface: the instance's upload path template, the sample path it renders, and the built-in preset list the settings section's dropdown fills from.  The template is the path grammar every NEW upload renders its file placement from: `originals/<ownerUserId>/<rendered>.<ext>`, with the per-user root pinning one directory tree per account. The grammar is variable substitution plus `{{#if album}}...{{else}}...{{/if}}` conditionals over the file (`{{filename}}`, `{{ext}}`, `{{filetype}}`, `{{filetypefull}}`), date (`{{y}} {{yy}} {{M}} {{MM}} {{MMM}} {{MMMM}} {{d}} {{dd}} {{W}} {{WW}} {{H}} {{HH}} {{h}} {{hh}} {{m}} {{mm}} {{s}} {{ss}} {{SSS}}`), album (`{{album}}`, `{{album-startDate-*}}`, `{{album-endDate-*}}`), camera (`{{make}} {{model}} {{lensModel}}`) and id (`{{assetId}}`, `{{assetIdShort}}`) families. Compilation is strict -- an unknown variable or a malformed conditional is an error, never a silent blank.  THE RENDERING CONTRACT (what changing the template does and never does): the path is rendered exactly once, at ingest, from the taken-at fallback chain (the file's EXIF header time, else the client-reported `deviceCreatedAt`, else the upload time) and the values known at that moment. Later metadata refinement NEVER re-paths an asset -- a refined taken-at, an album membership joined after upload, none of it moves a file. Existing assets keep the paths they were ingested under; there is deliberately NO migration of stored files when the template changes (moving files after ingest is the architecture photo libraries have lost data to; lumemor never moves them). New uploads follow the new template from the moment it is saved. Album variables render empty for direct uploads: the only album an upload joins at ingest is the system sync stream, which is not a naming source.  The sample path is the current template rendered over a fixed sample asset (an image named `IMG_1234.jpg`, taken `2023:07:14 10:15:30.123 +05:30`, camera `Acme Alpha 5` with the `24-70mm f/2.8` lens, no album, under the sample owner `0192f8c1-0000-7000-8000-000000000001`) with the extension appended -- exactly the shape a real upload of that file would land at under this template. The response never fails over a corrupt setting row: the default template answers instead, and the server log carries the corruption. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getStorageSettings();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getStorageSettings: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**StorageSettingsResponse**](StorageSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getTilesSettings**
> TilesSettingsResponse getTilesSettings()

Read the map tile source setting.

The map's tile source, as the administrator configured it: one URL, or the empty string when the instance has none.  THE SHAPE: the URL names a document the browser's map library can build a base layer from -- a style document (the vector case: sources, layers, sprites and glyphs) or a TileJSON descriptor (the raster case: the tile templates and their attribution). The map detects which of the two it received and builds the base layer accordingly, so one field covers both the raster and the vector-and-style forms of the self-hosting ecosystem.  THE EMPTY VALUE IS THE SHIPPED DEFAULT and a real state: no source ships with the product (a public tile service's usage policy forbids carrying the traffic of every install), so a fresh instance draws its map with the tile layer absent and says so. Loading the document needs the host to send the usual cross-origin headers; a host that does not lands on the same stated absence.  A corrupt settings row answers the empty string with the corruption logged server-side. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getTilesSettings();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getTilesSettings: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**TilesSettingsResponse**](TilesSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getTranscodingSettings**
> TranscodingSettingsResponse getTranscodingSettings()

Read the video transcoding settings.

The transcoding settings read of the admin surface: the instance's video transcoding profile -- the preset the transcode job re-encodes new video uploads at, the concurrency bound of simultaneous re-encodes, the video-thumbnail frame capture switch, and the keep-original-on-failure policy. A fresh instance (or a corrupt row) answers the documented defaults. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getTranscodingSettings();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getTranscodingSettings: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**TranscodingSettingsResponse**](TranscodingSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getTrashSettings**
> TrashSettingsResponse getTrashSettings()

Read the trash retention setting.

The trash settings read of the admin surface: the instance's trash retention in whole days -- the window a soft-deleted asset stays recoverable before the hourly expiry job purges it (rows, thumbnails, derivatives and orphaned files together). A fresh instance (or a corrupt row) answers the default 30; the read never fails over a row the save endpoint's validation should have kept honest. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getTrashSettings();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getTrashSettings: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**TrashSettingsResponse**](TrashSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getUpdateCheck**
> UpdateCheckResponse getUpdateCheck()

Read the instance's release check state.

The instance's release check, read from the cached answer -- this request never touches the network. The server runs the probe itself on the configured cadence (a release feed URL, the interval in hours) and this face reports what the last probe concluded.  THE PROBE: one plain HTTPS request for a public document -- no photo, no EXIF, no embedding, no instance identifier; the feed sees the instance's address and nothing else. The check can be switched off (a settings switch); the endpoint address must be a public one: an address that resolves to a loopback, private, link-local or otherwise internal range is refused, and redirects are never followed. A refused or failed probe is not an error state: `status` turns `unknown` and nothing else changes -- the next interval retries.  THE COMPARISON takes tags only: an optional `v` prefix, then the numeric version fields. `up_to_date` and `update_available` are the two claims a successful probe can make; `latestVersion` carries the feed's tag verbatim. The server never downloads or installs an update -- this surface only reports that a newer release exists.  `disabled` means the check does not run: the administrator switched it off, or the deployment disabled it at boot. The last answer is kept while disabled, so switching the check back on resumes from the same state. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getUpdateCheck();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getUpdateCheck: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**UpdateCheckResponse**](UpdateCheckResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getUpdateCheckSettings**
> UpdateCheckSettingsResponse getUpdateCheckSettings()

Read the release check settings.

The release check's three knobs, as the settings face renders them: the switch, the release feed URL and the cadence in hours.  `endpoint` is the URL the server fetches on every check. The document behind it must be a JSON object whose newest tag rides `tag` (a release feed that names it `tag_name`, as release APIs commonly do, is read the same way). The value is handed back verbatim; the default is the product's own release feed.  `intervalHours` is the cadence: the server probes at most once per interval, and the answer is cached until the next one -- a failed probe waits the same interval instead of hammering a feed that is not there.  A corrupt settings row answers the documented defaults (enabled, the default feed, 24 hours) with the corruption logged server-side, the same discipline the other instance settings keep. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.getUpdateCheckSettings();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->getUpdateCheckSettings: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**UpdateCheckSettingsResponse**](UpdateCheckSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listJobs**
> JobRunListResponse listJobs(limit)

List the job runs of the task panel.

The task panel listing of the admin surface: every job run the instance ever recorded, newest first -- model downloads and ML reruns alike. Each row carries the run's progress as its own work's units: the model download's `total` and `done` count pinned bytes and `failed` counts failed models; a rerun's count the assets of its scope and the assets whose rerun failed. `startedAt` is null while a run sits `queued`, `finishedAt` is null until it closes.  `limit` bounds the page size: absent means the default of 50, a value over 200 is answered with the maximum of 200, and a value below 1 answers 400 `VALIDATION_FAILED`. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final int limit = 50; // int | Page size: the number of job runs one response carries at most. Defaults to 50; values over 200 are answered with the maximum of 200, values below 1 answer 400. 

try {
    final response = api.listJobs(limit);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->listJobs: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **limit** | **int**| Page size: the number of job runs one response carries at most. Defaults to 50; values over 200 are answered with the maximum of 200, values below 1 answer 400.  | [optional] [default to 50]

### Return type

[**JobRunListResponse**](JobRunListResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listModels**
> ModelListResponse listModels()

List the registered models and the download run's state.

The model management listing of the admin surface: every registered model of the instance with its facts, its activation state and the presence of its files, plus the most recent model download run as the summary the model page's progress bar renders.  Every model row of the `models` table appears, grouped by slot then id -- including a model whose weights went missing, which lists with `filesPresent: false` so the administrator can see and clean it up. `filesPresent` is a presence probe only (the on-disk manifest is readable and every file it pins exists); the sha256 verification itself is the activation's and the session loader's business, never the listing's. `license`, `notice` and `inputSize` come from the model's on-disk manifest, `dim` and `params` from the registered row (which the download and the scan keep in step with that manifest); `dim` and `inputSize` are null when the model declares none, `license` is empty and `notice` null when the manifest is unreadable; `isActive` comes from the table.  `download` is the most recent `model_download` run of whatever state -- the run the download endpoint's idempotence would return -- or null before the first download was ever started.  `slots` is the per-slot state view the page's cards render: one row per pipeline slot, each carrying one of the four states the cards speak (`active`, `downloading`, `not_downloaded`, `failed`), the slot's active model, its source facts (official or custom, the host, the digest, the download time) and the download run's progress while a run is in flight. `official` is the manifest's catalog -- the validated list the switch dialog's official group renders, every entry with its download state (`downloaded` is the presence probe, the digest check stays the activation's). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.listModels();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->listModels: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**ModelListResponse**](ModelListResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listUsers**
> AdminUserListResponse listUsers()

List the instance's accounts.

The user management listing of the admin surface: every account of the instance with its role and its disabled switch, oldest first. The listing selects only the safe columns -- a password hash never appears in any admin response.  The list is small by construction (a family instance holds a handful of accounts) and grows only through this surface's own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. Account rows never disappear (deleting an account together with its data is a later disposal flow), so a refetch after any write reconciles the list completely.  The endpoint is admin-only: the session middleware authenticates the caller like every authed route, and a caller whose role is not `admin` answers 403 AUTH_FORBIDDEN -- a code distinct from the 401 family, because the caller is signed in, just not entitled to this surface. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.listUsers();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->listUsers: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**AdminUserListResponse**](AdminUserListResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **probeModelSource**
> ModelProbeAcceptedResponse probeModelSource(slot, kind, url, sha256, file)

Probe a model source for one slot.

The first half of a source import: download (or read) the candidate model, check it against the slot's contract, and stage it -- without touching the slot's active model. The answer is 202 with a probeId; the probe itself runs in the server's background (a URL download legitimately takes minutes) and is read through the probe GET until it settles.  The body is `multipart/form-data` with these parts:  - `kind` (text, required) -- `url` or `file`. - `url` (text, required for kind `url`) -- the model URL; the   download lands in the slot's staging area, its bytes are   never trusted before the digest check. - `sha256` (text, optional) -- a digest the bytes must match;   without it the probe still computes and reports the digest,   and the save marks the model an unverified source. - `file` (file, required for kind `file`) -- the ONNX bytes   (up to 2 GiB; the sniff reads the content, never the file   name).  The checks, in order, every failure naming its own check: the digest when one was given; the graph loads (a file that is not a readable ONNX model is refused); the input shape and the output dimension against the slot's contract -- a mismatch is refused naming both numbers. A host that is not on the download sources list does NOT refuse: the probe answers `warn` (the checks passed, the host is undeclared, the save is still the administrator's own act and its use is logged).  The probe does not register anything: the save endpoint takes the probeId, and a probe whose checks failed can never be saved. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final ModelSlot slot = clip_embed; // ModelSlot | The slot the candidate model would fill.
final String kind = kind_example; // String | Which source the candidate comes from.
final String url = url_example; // String | The candidate model's URL, for kind `url`.
final String sha256 = sha256_example; // String | The digest the bytes must match, when the caller pins one (64 lowercase hex chars). 
final MultipartFile file = BINARY_DATA_HERE; // MultipartFile | The candidate model's bytes, for kind `file`.

try {
    final response = api.probeModelSource(slot, kind, url, sha256, file);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->probeModelSource: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slot** | [**ModelSlot**](.md)| The slot the candidate model would fill. | 
 **kind** | **String**| Which source the candidate comes from. | 
 **url** | **String**| The candidate model's URL, for kind `url`. | [optional] 
 **sha256** | **String**| The digest the bytes must match, when the caller pins one (64 lowercase hex chars).  | [optional] 
 **file** | **MultipartFile**| The candidate model's bytes, for kind `file`. | [optional] 

### Return type

[**ModelProbeAcceptedResponse**](ModelProbeAcceptedResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: multipart/form-data
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **rebuildDerivatives**
> RebuildDerivativesResponse rebuildDerivatives(rebuildDerivativesRequest)

Start a derivative rebuild run.

The image card's explicit rebuild: one run that re-derives the serving copies of every image asset at the settings the instance currently carries.  THE COST IS NEVER SILENT: the image card states what the settings change (nothing is recomputed until this run is asked for), and the two scopes are the honest pair --  - `missing` derives only the assets whose current-format   serving files are absent: a library that has never been   rebuilt, or one that gained the full-size copy, pays only   for what it lacks (minutes on a family library). - `all` re-derives every image asset at the current settings   and removes the other encoding format's stale files: the CPU   belongs to the rebuild for the estimate's duration, and   browsing is slower while it runs.  THE RUN IS A JOB: it lists on the task panel with the four states, its progress counts the image assets it visited, and the pause and window rules that govern thumbnail-class work hold it the same way -- a night window that closes stops it at the asset it just finished, and the next window resumes from there, never from zero.  The endpoint is idempotent the way the place re-resolution is: a rebuild run still in flight (`queued` or `running`) answers 200 with its identifier and starts nothing; a finished run never blocks a new one.  SETTINGS CHANGES NEVER TRIGGER THIS: saving the image settings writes the document only -- the serving copies already on disk stay as they are until an explicit rebuild re-derives them. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final RebuildDerivativesRequest rebuildDerivativesRequest = {"scope":"missing"}; // RebuildDerivativesRequest | 

try {
    final response = api.rebuildDerivatives(rebuildDerivativesRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->rebuildDerivatives: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **rebuildDerivativesRequest** | [**RebuildDerivativesRequest**](RebuildDerivativesRequest.md)|  | 

### Return type

[**RebuildDerivativesResponse**](RebuildDerivativesResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **reresolvePlaces**
> PlacesRerunResponse reresolvePlaces()

Start a full place re-resolution run.

The places-data-update path of the admin surface: one run that re-resolves the place of EVERY asset whose exif row carries GPS coordinates, against the baked place data the server currently runs on. The run that refills resolutions after the baked GeoNames data changed -- the startup import of a rebuilt image replaces the place tables and, through the schema's ON DELETE SET NULL, releases every stale resolution; this run is the deliberate step that refills them. (The metadata job resolves every NEW upload on its own; this run only matters for re-resolving what already exists.)  The scope is every GPS-bearing asset row whatever its lifecycle state -- place resolution is metadata, not visibility. `total` and `done` of the run's `job_runs` row count those assets and `failed` the ones whose re-resolution write failed.  The endpoint is idempotent the way the model rerun is: a place re-resolution run still in flight (`queued` or `running`) answers 200 with its identifier and starts nothing; a finished (or failed) run never blocks a new one.  The run is API-only for now: no admin page button is wired to it yet (the job panel's listing shows its progress like any other run). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.reresolvePlaces();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->reresolvePlaces: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**PlacesRerunResponse**](PlacesRerunResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **rerunModel**
> ModelRerunResponse rerunModel(id)

Start a manual rerun of the active model's slot.

The rerun trigger of the admin surface: one run that redoes the slot's ML work over the whole library -- the run that fills the newly activated model's products, the manual step every model switch deliberately leaves to the administrator (a same-dimension switch keeps the old products usable immediately; a cross-dimension switch leaves the table empty until this run refills it).  Only the ACTIVE model of a slot can be rerun: products attach to the active model, so a rerun of a dormant model would fill rows no active-model query could see. The run's scope is every image asset that finished its pipeline and is not in the trash (ML work belongs after asset readiness, and video assets stay out of ML so far); `total` and `done` of the run's row count those assets and `failed` the ones whose rerun failed -- progress in asset counts, not bytes (the model download's row counts pinned bytes and failed models; each run's units are its own work's).  The endpoint is idempotent the way the download endpoint is: a rerun of the same slot still in flight (`queued` or `running`) answers 200 with its identifier and starts nothing. Unlike the download, a FINISHED run never blocks a new one -- the rerun is manual, so a second click after a completed (or failed) run starts a fresh run.  The per-asset ML workers of the slots arrive later; until a slot's dispatch is wired, a rerun of that slot still creates the run but the run fails (state `failed`, zero progress) instead of pretending to have done work it cannot do. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final String id = yunet-2023mar; // String | The identifier of the active model whose slot is rerun (the models table's slug id, as the listing returns it). 

try {
    final response = api.rerunModel(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->rerunModel: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| The identifier of the active model whose slot is rerun (the models table's slug id, as the listing returns it).  | 

### Return type

[**ModelRerunResponse**](ModelRerunResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **runUpdateCheck**
> UpdateCheckResponse runUpdateCheck()

Run the release check now.

The manual check of the settings face: the server probes the configured release feed now (the same bounded, refused- internal-address, redirect-refusing request the scheduled check makes) and answers the fresh state -- the same document `GET` serves.  A failed probe answers 200 with `status: unknown`, exactly as the scheduled one would: a check that cannot reach its feed is the ordinary case of an offline instance, not a request error, and it carries no message. A check that is switched off (or disabled by the deployment) answers 200 with `disabled` and probes nothing. The probe is bounded server-side and well under the request timeout; a slow feed lands on `unknown` like any other failure. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.runUpdateCheck();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->runUpdateCheck: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**UpdateCheckResponse**](UpdateCheckResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **scanModels**
> ModelScanResponse scanModels()

Register the custom models found in the models dir.

The custom model registration of the admin surface: every directory under the instance's models dir that carries a `manifest.json` is verified against the digests the manifest pins and registered in the models table with source `custom`. Directories without a manifest are skipped -- the models dir also holds download staging, not every entry is a model.  Scanning registers, it never activates: an administrator reviews the registered models and switches them on through the activation endpoint. Re-scanning converges: the registration is an upsert on the descriptive fields and leaves every row's activation state alone. A directory whose files drifted from their pinned digests fails the whole scan with 400 MODEL_INVALID and registers nothing. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.scanModels();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->scanModels: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**ModelScanResponse**](ModelScanResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **stopDerivativesRebuild**
> RebuildDerivativesResponse stopDerivativesRebuild()

Stop the in-flight derivative rebuild run.

The task panel's stop for a rebuild in flight: the run's row is marked as asked-to-stop, and the driver -- which checks the flag between assets -- closes the run at the asset it just finished, never mid-asset. The work already done stays on disk: a stopped rebuild has paid exactly for the assets it re-derived, and the next rebuild's `missing` scope skips them.  The run closes `failed` with `failed` counting nothing: the task panel renders the reason line as stopped, not as a broken run -- the admin asked for this ending.  A stop with no rebuild in flight (the run finished between the panel's listing and the click, or none ever started) answers 409 `REBUILD_NOT_RUNNING`: the flag has nothing to land on. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();

try {
    final response = api.stopDerivativesRebuild();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->stopDerivativesRebuild: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**RebuildDerivativesResponse**](RebuildDerivativesResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **switchModelSource**
> ModelSourceSwitchResponse switchModelSource(slot, modelSourceSwitchRequest)

Switch a slot onto a probed source.

The second half of a source import, and the official list's switch: the slot is switched onto the source the request names, under the slot's lock. The answer is 202 -- the switch itself is seconds, the recalculation its semantics demand is an async run the task panel owns.  The body depends on the kind:  - `builtin` (the official list): `modelId` names the manifest   entry; nothing was probed because the entry is the   validated list. A cross-dimension switch of the embed slot   requires `confirmWipe: true` -- the confirmation is the   dialog's contract, and the response reports the wipe's facts   exactly as the activation switch does. - `url` or `file` (a custom source): `probeId` names a probe   that finished OK -- the server never reuses an unverified   result, a missing or failed probe answers 400. The model is   registered from the staged bytes (same sha256 reuse: bytes   already on disk are never stored twice), activated, and the   slot's recalculation run is enqueued.  The slot lock: a switch stops being possible while one is in flight (409) -- the same-dimension switch is seconds, but the recalculation it started is the slot's business, and two overlapping switches would race its products. The switch refuses to lose the named persons: a face slot's recalculation inherits the cluster names by overlap, so the confirm copy states what is kept; nothing here deletes a user label.  A failed switch changes nothing: the slot keeps its previous model and its products. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final ModelSlot slot = clip_embed; // ModelSlot | The slot being switched.
final ModelSourceSwitchRequest modelSourceSwitchRequest = {"kind":"builtin","modelId":"cnclip-vit-b16-vision"}; // ModelSourceSwitchRequest | 

try {
    final response = api.switchModelSource(slot, modelSourceSwitchRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->switchModelSource: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slot** | [**ModelSlot**](.md)| The slot being switched. | 
 **modelSourceSwitchRequest** | [**ModelSourceSwitchRequest**](ModelSourceSwitchRequest.md)|  | 

### Return type

[**ModelSourceSwitchResponse**](ModelSourceSwitchResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateImageSettings**
> ImageSettingsResponse updateImageSettings(imageSettingsUpdateRequest)

Save the derivative image settings.

The settings save. Every field is contract-required (an omitted one would decode as its zero value and silently change the posture), and each value is checked against its enum: the tile sizes 250 and 320, the preview sizes 1080, 1440 and 2048, the encodings `avif` and `webp`.  The save writes the document and nothing else: no derivation starts, no existing serving file is touched, and a rebuild already running keeps its own starting parameters. The new values bind at the next derivation -- a new upload's thumbnail job, or the explicit rebuild run the image card starts.  The save takes effect without a restart: the thumbnail job reads the settings at run time, so an upload that arrives after the save derives at them. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final ImageSettingsUpdateRequest imageSettingsUpdateRequest = {"tileSize":250,"previewSize":1080,"format":"avif","fullSize":false,"rawEmbedded":true}; // ImageSettingsUpdateRequest | 

try {
    final response = api.updateImageSettings(imageSettingsUpdateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->updateImageSettings: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **imageSettingsUpdateRequest** | [**ImageSettingsUpdateRequest**](ImageSettingsUpdateRequest.md)|  | 

### Return type

[**ImageSettingsResponse**](ImageSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateModelSettings**
> ModelSettingsResponse updateModelSettings(modelSettingsUpdateRequest)

Save the model download settings and download sources.

The settings save. `autoDownload`, `allowedHosts` and `mirrorBaseUrl` are required (an omitted field would decode as its zero value, silently changing the policy): the caller saves the whole object it read.  `allowedHosts` replaces the stored list wholesale; every entry must be a hostname, optionally with a leading `*.` wildcard label and an optional port. `mirrorBaseUrl`, when non-empty, must be a host (or a full http/https URL whose host is kept); its host joins `allowedHosts` automatically, so saving a mirror never leaves it refusing its own first hop. The values take effect on the next download run; no run is started or cancelled by the save itself. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final ModelSettingsUpdateRequest modelSettingsUpdateRequest = {"autoDownload":false,"allowedHosts":["huggingface.co","*.hf.co","hf-mirror.com"],"mirrorBaseUrl":""}; // ModelSettingsUpdateRequest | 

try {
    final response = api.updateModelSettings(modelSettingsUpdateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->updateModelSettings: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **modelSettingsUpdateRequest** | [**ModelSettingsUpdateRequest**](ModelSettingsUpdateRequest.md)|  | 

### Return type

[**ModelSettingsResponse**](ModelSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateRunBudgetSettings**
> RunBudgetSettingsResponse updateRunBudgetSettings(runBudgetSettingsUpdateRequest)

Save the background work's budget.

The budget save. Every field is contract-required (an omitted one would decode as its zero value and silently change the posture), and each value is checked: the concurrency preset against its three-step enum, the two times against the `HH:MM` form, and the five pause switches against booleans.  The two times must differ: an equal pair would define a zero-length window, which reads as a window that never opens -- turning the window OFF is the honest way to say that, so the save refuses the ambiguous shape.  The save takes effect immediately, without a restart: the next job a worker picks up reads the fresh document, and a job already held wakes within its hold granularity and re-reads. The switches never touch a job already running -- the one in flight finishes, which is the card's own headline.  The window and the pauses NEVER gate an upload or a serving request: they are worker-side facts. The upload path enqueues without consulting them, and the browser's reads are answered whatever the budget says. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final RunBudgetSettingsUpdateRequest runBudgetSettingsUpdateRequest = {"concurrencyPreset":"standard","nightlyEnabled":false,"nightlyStart":"02:00","nightlyEnd":"06:00","pausedJobs":{"thumbnail":false,"metadata":false,"transcode":false,"ml":false,"places":false}}; // RunBudgetSettingsUpdateRequest | 

try {
    final response = api.updateRunBudgetSettings(runBudgetSettingsUpdateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->updateRunBudgetSettings: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **runBudgetSettingsUpdateRequest** | [**RunBudgetSettingsUpdateRequest**](RunBudgetSettingsUpdateRequest.md)|  | 

### Return type

[**RunBudgetSettingsResponse**](RunBudgetSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateStorageSettings**
> StorageSettingsResponse updateStorageSettings(storageSettingsUpdateRequest)

Validate and save the storage template.

The storage settings save of the admin surface. The template passes a three-stage validation before anything is written:  1. **Compile** -- strict grammar: every `{{variable}}` must be a known member of the variable table, every conditional balanced. An unknown variable or a malformed conditional is rejected here. 2. **Sample render** -- the compiled template renders over a fixed set of sample assets (an album-less image, an album-grouped image, a video), exactly as an upload would: a template that renders an empty path (an album-only template over the album-less sample) is rejected here. 3. **Structure** -- the rendered paths must be safe: no `..` traversal, no dot segments, bounded depth (10) and bounded length, per-segment and total. The engine sanitizes the user-influenced values (file name, album, camera) and the structural pass guards the literal template text.  A template that passes is saved and answers 200 with the saved value, its sample render and the presets. A template that fails any stage answers 400 `VALIDATION_FAILED` naming `template` in the details, with the specific reason in the error message; nothing is written. The save takes effect for uploads from the next request on: paths already recorded are never re-derived and never moved (see GET for the full rendering contract). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final StorageSettingsUpdateRequest storageSettingsUpdateRequest = {"template":"{{y}}/{{MM}}/{{filename}}"}; // StorageSettingsUpdateRequest | 

try {
    final response = api.updateStorageSettings(storageSettingsUpdateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->updateStorageSettings: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **storageSettingsUpdateRequest** | [**StorageSettingsUpdateRequest**](StorageSettingsUpdateRequest.md)|  | 

### Return type

[**StorageSettingsResponse**](StorageSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateTilesSettings**
> TilesSettingsResponse updateTilesSettings(tilesSettingsUpdateRequest)

Save the map tile source setting.

The settings save. `tilejsonUrl` is required: the empty string clears the source (the map falls back to its stated absence), any other value must be an absolute `http` or `https` URL.  The save validates the SHAPE only. The browser is the party that loads the source, so a LAN address is legitimate here (a self-hosted tile server inside the home network) and reachability is not judged at save time. The value takes effect on the map's next load -- there is no restart, and a map already open keeps the source it loaded. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final TilesSettingsUpdateRequest tilesSettingsUpdateRequest = {"tilejsonUrl":"http://192.168.1.20:8080/tilejson.json"}; // TilesSettingsUpdateRequest | 

try {
    final response = api.updateTilesSettings(tilesSettingsUpdateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->updateTilesSettings: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tilesSettingsUpdateRequest** | [**TilesSettingsUpdateRequest**](TilesSettingsUpdateRequest.md)|  | 

### Return type

[**TilesSettingsResponse**](TilesSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateTranscodingSettings**
> TranscodingSettingsResponse updateTranscodingSettings(transcodingSettingsUpdateRequest)

Save the video transcoding settings.

The transcoding settings save of the admin surface.  **The scope pin, documented honestly: the profile affects NEW ingests only.** The transcode job of a video upload reads the profile when it runs, so a saved change governs every video uploaded after it; videos already transcoded keep their existing playable copy and are never re-encoded, and a video whose job is still queued picks up the profile at its run time (the job reads the setting at run time, not enqueue time).  The fields: `preset` bounds the re-encode's vertical resolution (`auto` keeps the source resolution, the numeric presets scale down to the bound, `off` skips the re-encode entirely -- the original file stays in place and downloadable, and the video's pipeline still completes); `concurrency` bounds how many re-encodes run at once; `thumbnailFrame` turns the video poster-frame capture (the video's thumbnail) on or off; `keepOriginal` decides the failure policy -- a failed re-encode either completes the job quietly (the original stays in place and downloadable through the original endpoint) or fails it for retry.  The save validates every field against its enum and answers the saved profile; an out-of-enum value answers 400 VALIDATION_FAILED naming the field, and nothing is written. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final TranscodingSettingsUpdateRequest transcodingSettingsUpdateRequest = {"preset":"720p","concurrency":1,"thumbnailFrame":true,"keepOriginal":false}; // TranscodingSettingsUpdateRequest | 

try {
    final response = api.updateTranscodingSettings(transcodingSettingsUpdateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->updateTranscodingSettings: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **transcodingSettingsUpdateRequest** | [**TranscodingSettingsUpdateRequest**](TranscodingSettingsUpdateRequest.md)|  | 

### Return type

[**TranscodingSettingsResponse**](TranscodingSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateTrashSettings**
> TrashSettingsResponse updateTrashSettings(trashSettingsUpdateRequest)

Save the trash retention setting.

The trash settings save of the admin surface. The retention is a whole number of days, 0 to 365:  - **0 means immediate permanent delete** -- an asset enters the trash and the NEXT expiry run purges it (the hourly periodic job and the boot run are the runs; nothing is deleted by the save request itself). - The expiry job reads the setting at the START of each run (the boot run and every hourly periodic run alike), so a change takes effect for the next run's window, not retroactively for rows a past run already considered. - An already-trashed asset's fate is decided by the run's own deadline computation: with a SHORTENED retention, rows past the new window leave on the next run; with a LENGTHENED one, every row gains the added days from its own deletion mark.  The save validates the bound and answers the saved value; a value outside 0-365 answers 400 VALIDATION_FAILED naming `retentionDays`, and nothing is written. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final TrashSettingsUpdateRequest trashSettingsUpdateRequest = {"retentionDays":0}; // TrashSettingsUpdateRequest | 

try {
    final response = api.updateTrashSettings(trashSettingsUpdateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->updateTrashSettings: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **trashSettingsUpdateRequest** | [**TrashSettingsUpdateRequest**](TrashSettingsUpdateRequest.md)|  | 

### Return type

[**TrashSettingsResponse**](TrashSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateUpdateCheckSettings**
> UpdateCheckSettingsResponse updateUpdateCheckSettings(updateCheckSettingsUpdateRequest)

Save the release check settings.

The settings save: all three fields are required (an omitted `enabled` would decode as the disabled switch, silently turning the check off), and each is validated before anything is written.  `endpoint` must be an `http` or `https` URL with a host -- the address the server will probe. The probe itself refuses addresses that resolve to loopback, private, link-local or otherwise internal ranges, and never follows a redirect: the setting can be saved pointing at an internal host, and the check simply stays `unknown` (the settings save does not probe, so it cannot judge reachability).  A failed validation answers 400 `VALIDATION_FAILED` naming the offending field, with nothing written. A saved value takes effect from the next scheduled check on; the manual check runs against the saved value immediately. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final UpdateCheckSettingsUpdateRequest updateCheckSettingsUpdateRequest = {"enabled":true,"endpoint":"https://api.github.com/repos/welpher/lumemor/releases/latest","intervalHours":24}; // UpdateCheckSettingsUpdateRequest | 

try {
    final response = api.updateUpdateCheckSettings(updateCheckSettingsUpdateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->updateUpdateCheckSettings: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **updateCheckSettingsUpdateRequest** | [**UpdateCheckSettingsUpdateRequest**](UpdateCheckSettingsUpdateRequest.md)|  | 

### Return type

[**UpdateCheckSettingsResponse**](UpdateCheckSettingsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateUser**
> AdminUserResponse updateUser(id, adminUserPatch)

Toggle one account's disabled switch.

The enable/disable switch of the admin surface: an administrator locks an account out of the instance (or lets it back in) without deleting the account or any of its data. The switch is enforced by the session middleware on EVERY request, which is what pins its observable semantics:  - Disabling takes effect with the account's next request: an   existing session of the account answers 401 from then on   (byte-identical with any other authentication failure) and   sign-in is refused the same way. The session rows   themselves are not deleted -- disabling revokes their   authority, not their existence. - Re-enabling restores the account's unexpired sessions   without a new sign-in; a session that lapsed its own   expiry while the account was disabled is gone for good.  The write is idempotent -- setting the current value again answers 200 with the unchanged account -- and the response is the account after the write, so the client adopts it as the fresh state without a refetch.  An administrator cannot disable their own account: the last admin locking themselves out would strand the admin surface, so the request answers 400 VALIDATION_FAILED naming the id. Nothing stops an admin from disabling ANOTHER administrator; the last enabled admin is the instance's line of defense.  The id must name an account. Account ids are enumerable through this surface's own listing, so there is no anti-enumeration pairing here: an unknown id (a stale list on the client) answers 404 USER_NOT_FOUND. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAdminApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the account.
final AdminUserPatch adminUserPatch = {"isDisabled":true}; // AdminUserPatch | 

try {
    final response = api.updateUser(id, adminUserPatch);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AdminApi->updateUser: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Identifier of the account. | 
 **adminUserPatch** | [**AdminUserPatch**](AdminUserPatch.md)|  | 

### Return type

[**AdminUserResponse**](AdminUserResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

