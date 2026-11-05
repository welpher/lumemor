# lumemor_api.api.UsersApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**listInstanceUsers**](UsersApi.md#listinstanceusers) | **GET** /users | The instance&#39;s member list.


# **listInstanceUsers**
> UserListResponse listInstanceUsers()

The instance's member list.

The minimal member list of the instance: one entry per account, `email` ascending. Any authenticated account may enumerate it -- the family-instance posture: the accounts of a shared photo home are not secret from one another, and the album share dialog's pick list (the owner choosing whom to grant) is the list's first consumer.  The entries carry exactly the identity a pick list renders: `id`, `email`, `role` and `isDisabled` -- no display name, no timestamps, and a password hash never appears in any response. The disabled switch rides the entry so callers can present accounts that cannot sign in for what they are (granting to one still answers 400 through the share endpoint's own target rule).  The list is small by construction (a family instance holds a handful of accounts) and grows only through the admin surface's own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. This is NOT the admin management face (`GET /admin/users`): it is the minimal, non-managing pick list every authenticated account may read. 

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

