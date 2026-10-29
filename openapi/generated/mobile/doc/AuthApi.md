# lumemor_api.api.AuthApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**consumePairingToken**](AuthApi.md#consumepairingtoken) | **POST** /auth/pairing/consume | Exchange a pairing token for a mobile session.
[**createPairingToken**](AuthApi.md#createpairingtoken) | **POST** /auth/pairing/token | Create a one-time device pairing token.
[**listSessions**](AuthApi.md#listsessions) | **GET** /auth/sessions | List the caller&#39;s active sessions.
[**login**](AuthApi.md#login) | **POST** /auth/login | Exchange credentials for a web session.
[**logout**](AuthApi.md#logout) | **POST** /auth/logout | Revoke the current session.
[**revokeSession**](AuthApi.md#revokesession) | **DELETE** /auth/sessions/{id} | Revoke one of the caller&#39;s sessions.


# **consumePairingToken**
> BearerSession consumePairingToken(pairingConsumeRequest)

Exchange a pairing token for a mobile session.

Completes device pairing: the mobile app scans the QR code and posts the token together with its device metadata. The server burns the one-time token and returns the bearer credential of a new mobile session of the user who created the token.  Bearer sessions never expire; the app holds the token until the session is revoked -- from the device list, by logging the device out, or by a password change, which revokes every session of the user. The server stores only the hash of the token, so it appears exactly once, in this response; a device that loses it must pair again.  Failed attempts are rate limited to stop guessing tokens. 

### Example
```dart
import 'package:lumemor_api/api.dart';

final api = LumemorApi().getAuthApi();
final PairingConsumeRequest pairingConsumeRequest = ; // PairingConsumeRequest | 

try {
    final response = api.consumePairingToken(pairingConsumeRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AuthApi->consumePairingToken: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **pairingConsumeRequest** | [**PairingConsumeRequest**](PairingConsumeRequest.md)|  | 

### Return type

[**BearerSession**](BearerSession.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **createPairingToken**
> PairingTokenResponse createPairingToken()

Create a one-time device pairing token.

Issues the secret behind the add-a-family-device QR code: a signed-in user opens the pairing page, and the new device scans the rendered code and redeems it at `POST /auth/pairing/consume` to gain a session of that user.  The token encodes 128 random bits as a URL-safe string, can be consumed once within five minutes, and is stored only as a hash, so a leaked database cannot redeem it. Creation is rate limited per user to stop farming QR codes. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAuthApi();

try {
    final response = api.createPairingToken();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AuthApi->createPairingToken: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**PairingTokenResponse**](PairingTokenResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listSessions**
> BuiltList<SessionInfo> listSessions()

List the caller's active sessions.

Returns every active session of the authenticated user across devices -- web cookie sessions and mobile bearer sessions -- most recently used first. The device management view (signing out other devices) is this list plus `DELETE /auth/sessions/{id}`. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAuthApi();

try {
    final response = api.listSessions();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AuthApi->listSessions: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**BuiltList&lt;SessionInfo&gt;**](SessionInfo.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **login**
> SessionInfo login(loginRequest)

Exchange credentials for a web session.

The login endpoint of web clients: a valid email and password create a new web session. The session credential is the `lumemor_session` cookie set by this response (HttpOnly, SameSite=Strict, Secure over HTTPS); the body describes the session for the account view and the device list. Mobile devices never log in here -- they pair through `POST /auth/pairing/consume` and hold a bearer token instead.  Web sessions slide their expiry by 30 days on use. Failed attempts are rate limited per address and account: after a burst of failures the endpoint answers 429 and the lockout grows with each further attempt. 

### Example
```dart
import 'package:lumemor_api/api.dart';

final api = LumemorApi().getAuthApi();
final LoginRequest loginRequest = ; // LoginRequest | 

try {
    final response = api.login(loginRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AuthApi->login: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **loginRequest** | [**LoginRequest**](LoginRequest.md)|  | 

### Return type

[**SessionInfo**](SessionInfo.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **logout**
> logout()

Revoke the current session.

Signs the caller out by revoking the session that authenticates the request -- the web cookie session and the mobile bearer session are the same session records. On web clients the response also clears the session cookie. A request made with an already-revoked session fails authentication with 401 before reaching this handler. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAuthApi();

try {
    api.logout();
} on DioException catch (e) {
    print('Exception when calling AuthApi->logout: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **revokeSession**
> revokeSession(id)

Revoke one of the caller's sessions.

Signs out one device by revoking the session with the given id, for example from the device management list. Sessions are scoped to their owner and the delete follows set semantics: an id that is unknown, belongs to another user, or is already revoked still answers 204, without revealing whether such a session exists. Revoking the session of the current request behaves like logout. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAuthApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the session to revoke.

try {
    api.revokeSession(id);
} on DioException catch (e) {
    print('Exception when calling AuthApi->revokeSession: $e\n');
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

