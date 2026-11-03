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
[**downloadModels**](AdminApi.md#downloadmodels) | **POST** /admin/models/download | Start the default model group&#39;s download.
[**listJobs**](AdminApi.md#listjobs) | **GET** /admin/jobs | List the job runs of the task panel.
[**listModels**](AdminApi.md#listmodels) | **GET** /admin/models | List the registered models and the download run&#39;s state.
[**listUsers**](AdminApi.md#listusers) | **GET** /admin/users | List the instance&#39;s accounts.
[**rerunModel**](AdminApi.md#rerunmodel) | **POST** /admin/models/{id}/rerun | Start a manual rerun of the active model&#39;s slot.
[**scanModels**](AdminApi.md#scanmodels) | **POST** /admin/models/scan | Register the custom models found in the models dir.
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

The model management listing of the admin surface: every registered model of the instance with its facts, its activation state and the presence of its files, plus the most recent model download run as the summary the model page's progress bar renders.  Every model row of the `models` table appears, grouped by slot then id -- including a model whose weights went missing, which lists with `filesPresent: false` so the administrator can see and clean it up. `filesPresent` is a presence probe only (the on-disk manifest is readable and every file it pins exists); the sha256 verification itself is the activation's and the session loader's business, never the listing's. `license` and `inputSize` come from the model's on-disk manifest, `dim` and `params` from the registered row (which the download and the scan keep in step with that manifest); `dim` and `inputSize` are null when the model declares none, `license` is empty when the manifest is unreadable; `isActive` comes from the table.  `download` is the most recent `model_download` run of whatever state -- the run the download endpoint's idempotence would return -- or null before the first download was ever started. 

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

The user management listing of the admin surface: every account of the instance with its role and its disabled switch, oldest first. The listing selects only the safe columns -- a password hash never appears in any admin response.  The list is small by construction (a family instance holds a handful of accounts) and grows only through this surface's own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. Account rows never disappear (deleting an account together with its data is a later milestone's disposal flow), so a refetch after any write reconciles the list completely.  The endpoint is admin-only: the session middleware authenticates the caller like every authed route, and a caller whose role is not `admin` answers 403 AUTH_FORBIDDEN -- a code distinct from the 401 family, because the caller is signed in, just not entitled to this surface. 

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

# **rerunModel**
> ModelRerunResponse rerunModel(id)

Start a manual rerun of the active model's slot.

The rerun trigger of the admin surface: one run that redoes the slot's ML work over the whole library -- the run that fills the newly activated model's products, the manual step every model switch deliberately leaves to the administrator (a same-dimension switch keeps the old products usable immediately; a cross-dimension switch leaves the table empty until this run refills it).  Only the ACTIVE model of a slot can be rerun: products attach to the active model, so a rerun of a dormant model would fill rows no active-model query could see. The run's scope is every image asset that finished its pipeline and is not in the trash (ML work belongs after asset readiness, and video assets stay out of ML in this milestone); `total` and `done` of the run's row count those assets and `failed` the ones whose rerun failed -- progress in asset counts, not bytes (the model download's row counts pinned bytes and failed models; each run's units are its own work's).  The endpoint is idempotent the way the download endpoint is: a rerun of the same slot still in flight (`queued` or `running`) answers 200 with its identifier and starts nothing. Unlike the download, a FINISHED run never blocks a new one -- the rerun is manual, so a second click after a completed (or failed) run starts a fresh run.  The per-asset ML workers of the slots arrive with the later tasks of this milestone; until a slot's dispatch is wired, a rerun of that slot still creates the run but the run fails (state `failed`, zero progress) instead of pretending to have done work it cannot do. 

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

