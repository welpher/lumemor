# lumemor_api.api.AdminApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createUser**](AdminApi.md#createuser) | **POST** /admin/users | Create one account.
[**downloadModels**](AdminApi.md#downloadmodels) | **POST** /admin/models/download | Start the default model group&#39;s download.
[**listUsers**](AdminApi.md#listusers) | **GET** /admin/users | List the instance&#39;s accounts.
[**updateUser**](AdminApi.md#updateuser) | **PATCH** /admin/users/{id} | Toggle one account&#39;s disabled switch.


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

