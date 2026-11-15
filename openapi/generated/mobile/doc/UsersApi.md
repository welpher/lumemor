# lumemor_api.api.UsersApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**changeUserPassword**](UsersApi.md#changeuserpassword) | **PUT** /users/me/password | Change the caller&#39;s own password.
[**getUserSelf**](UsersApi.md#getuserself) | **GET** /users/me | Read the caller&#39;s own account.
[**listInstanceUsers**](UsersApi.md#listinstanceusers) | **GET** /users | The instance&#39;s member list.
[**listUserDevices**](UsersApi.md#listuserdevices) | **GET** /users/me/sessions | List the caller&#39;s signed-in devices.
[**revokeUserDevice**](UsersApi.md#revokeuserdevice) | **DELETE** /users/me/sessions/{id} | Revoke one of the caller&#39;s other devices.
[**updateUserSelf**](UsersApi.md#updateuserself) | **PATCH** /users/me | Patch the caller&#39;s own account.


# **changeUserPassword**
> changeUserPassword(passwordChangeRequest)

Change the caller's own password.

The self-scoped password change behind the settings page's change-password section. The current password must verify against the account's stored Argon2id hash; the next one is stored as a fresh Argon2id hash (the same profile every creation path pins -- a rehash with a new salt, never a mutation of the old hash).  **The revocation contract, pinned:** a successful change revokes EVERY OTHER session of the account, across all devices -- web cookie sessions and mobile bearer sessions alike, their credentials failing authentication with 401 from the next request on -- while the session that performed the change STAYS signed in (the caller is not signed out of the device they just proved they own). The rationale is the credential-severance rule: a changed password means every other held credential of the account is stale, and a device that wants back in signs in with the new password.  A wrong current password answers 400 VALIDATION_FAILED naming `current` -- NOT 401, because the request''s session is valid and healthy; only the request''s own field failed to verify. (Clients must not treat it as a session expiry.) The next password follows the account rules everywhere: 8 to 128 runes. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getUsersApi();
final PasswordChangeRequest passwordChangeRequest = {"current":"correct horse battery staple","next":"another fine passphrase here"}; // PasswordChangeRequest | 

try {
    api.changeUserPassword(passwordChangeRequest);
} on DioException catch (e) {
    print('Exception when calling UsersApi->changeUserPassword: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **passwordChangeRequest** | [**PasswordChangeRequest**](PasswordChangeRequest.md)|  | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getUserSelf**
> UserSelfResponse getUserSelf()

Read the caller's own account.

The self-scoped profile read behind the user settings page: the calling session's own account row -- the identity fields of the profile section, the appearance pair (theme and locale) the account persisted, and the download preferences the batch and whole-album download flows will consume. The endpoint is `me`-scoped by construction: the session that authenticates the request IS the identity, there is no userId parameter to get wrong, and one account can never read another's row through it.  The theme and the locale are the values the account chose on the settings page; a client that applies them (the web app's `.dark` class and i18n store) restores the account's appearance at sign-in. The download preferences are declarative account state: they persist here and every future download flow reads them as the account's default packaging (see `UserSelfPatch` for the honest consumption state). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getUsersApi();

try {
    final response = api.getUserSelf();
    print(response);
} on DioException catch (e) {
    print('Exception when calling UsersApi->getUserSelf: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**UserSelfResponse**](UserSelfResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listInstanceUsers**
> UserListResponse listInstanceUsers()

The instance's member list.

The minimal member list of the instance: one entry per account, `email` ascending. Any authenticated account may enumerate it -- the family-instance posture: the accounts of a shared photo home are not secret from one another, and the album share dialog's pick list (the owner choosing whom to grant) is the list's first consumer.  The entries carry exactly the identity a pick list renders: `id`, `email`, `name`, `role` and `isDisabled` -- no timestamps, and a password hash never appears in any response. The display name leads there (the pick list says \"Mum\", not her login address); `email` stays the fallback an empty name leaves and the disambiguator between two accounts that share a name. The disabled switch rides the entry so callers can present accounts that cannot sign in for what they are (granting to one still answers 400 through the share endpoint's own target rule).  The list is small by construction (a family instance holds a handful of accounts) and grows only through the admin surface's own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. This is NOT the admin management face (`GET /admin/users`): it is the minimal, non-managing pick list every authenticated account may read. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getUsersApi();

try {
    final response = api.listInstanceUsers();
    print(response);
} on DioException catch (e) {
    print('Exception when calling UsersApi->listInstanceUsers: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**UserListResponse**](UserListResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listUserDevices**
> BuiltList<DeviceSession> listUserDevices()

List the caller's signed-in devices.

The device list behind the settings page's devices section: every active session of the calling account across devices, most recently used first, each entry carrying the device rendering's own fields -- and, unlike the auth surface's own `GET /auth/sessions`, the `isCurrent` marker derived by comparing the session id against the credential of THIS request, plus the session's creation-time IP address.  The list is the account's own sessions only (the `me` scoping again: the session IS the identity); a web session and a paired phone both appear. `ipAddress` is the address the session was created from (the login's or the pairing exchange's socket peer), omitted when the row predates the column or the address was not captured. It is an audit hint, not a geolocation claim. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getUsersApi();

try {
    final response = api.listUserDevices();
    print(response);
} on DioException catch (e) {
    print('Exception when calling UsersApi->listUserDevices: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**BuiltList&lt;DeviceSession&gt;**](DeviceSession.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **revokeUserDevice**
> revokeUserDevice(id)

Revoke one of the caller's other devices.

The device list's revoke button: signs out one device by deleting its session row. The endpoint is owner-scoped like the auth surface's own `DELETE /auth/sessions/{id}` -- an id that is unknown, belongs to another account, or is already revoked still answers the same 204, without revealing whether such a session exists.  THE ONE DIFFERENCE FROM THE AUTH SURFACE: the CURRENT session -- the one authenticating this very request, the row the device list marks `isCurrent` -- answers 400 VALIDATION_FAILED instead. The settings page's list is a remote-device manager; the current device's own exit is the sign-out flow (`POST /auth/logout`), and the 400 keeps a stray click on the wrong row from signing the caller out of the very session they are managing from. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getUsersApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the session to revoke.

try {
    api.revokeUserDevice(id);
} on DioException catch (e) {
    print('Exception when calling UsersApi->revokeUserDevice: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Identifier of the session to revoke. | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateUserSelf**
> UserSelfResponse updateUserSelf(userSelfPatch)

Patch the caller's own account.

The self-scoped write behind the settings page's profile, appearance and download-preferences sections. Every field is optional: an absent field keeps its stored value, a present one replaces it, and the response is the complete saved account state (the caller adopts it as the fresh state, exactly like the storage settings save).  THE ME-SCOPING: the endpoint addresses the session's own account row only -- no userId parameter exists, and the session middleware's identity decides the row. The email is the one field whose uniqueness spans accounts: an address another account already owns answers 409 USER_EMAIL_TAKEN and nothing is written.  The theme and the locale persist for the NEXT sign-in and every client that reads them; the client performing the patch is expected to apply them immediately itself (the web app flips its `.dark` class and reloads its i18n store on the response, not on a re-login).  THE DOWNLOAD PREFERENCES split honestly in two. The embedded-motion switch is consumed today: the single-asset download reads it when it assembles a Live Photo's archive. The format, videos and sidecars switches stay declarative -- the batch and whole-album download endpoints do not exist yet (they arrive with a later surface), and those flows will read the account's default packaging when they arrive: a stored preference is never silently dropped, it is simply not consumed yet. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getUsersApi();
final UserSelfPatch userSelfPatch = {"name":"Ada Lovelace","email":"ada@example.com"}; // UserSelfPatch | 

try {
    final response = api.updateUserSelf(userSelfPatch);
    print(response);
} on DioException catch (e) {
    print('Exception when calling UsersApi->updateUserSelf: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **userSelfPatch** | [**UserSelfPatch**](UserSelfPatch.md)|  | 

### Return type

[**UserSelfResponse**](UserSelfResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

