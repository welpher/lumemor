# lumemor_api.api.ServerApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createAdmin**](ServerApi.md#createadmin) | **POST** /server/setup/admin | Create the first administrator account.
[**getServerInfo**](ServerApi.md#getserverinfo) | **GET** /server/info | Report instance version and feature switches.


# **createAdmin**
> User createAdmin(createAdminRequest)

Create the first administrator account.

One-time first-run wizard endpoint, available only while the instance has no administrator. Once an admin exists the endpoint answers 409 `SETUP_ALREADY_COMPLETED` and the client should route to login.  The password is stored as an Argon2id hash; it never appears in any response. 

### Example
```dart
import 'package:lumemor_api/api.dart';

final api = LumemorApi().getServerApi();
final CreateAdminRequest createAdminRequest = ; // CreateAdminRequest | 

try {
    final response = api.createAdmin(createAdminRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling ServerApi->createAdmin: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createAdminRequest** | [**CreateAdminRequest**](CreateAdminRequest.md)|  | 

### Return type

[**User**](User.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getServerInfo**
> ServerInfo getServerInfo()

Report instance version and feature switches.

Unauthenticated capability discovery. Clients call it before login to locate the API, render the first-run wizard when the instance is not yet initialized, and adapt to the features the instance enables.  `initialized` is derived from the existence of an administrator account. 

### Example
```dart
import 'package:lumemor_api/api.dart';

final api = LumemorApi().getServerApi();

try {
    final response = api.getServerInfo();
    print(response);
} on DioException catch (e) {
    print('Exception when calling ServerApi->getServerInfo: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**ServerInfo**](ServerInfo.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

