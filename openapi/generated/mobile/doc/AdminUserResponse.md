# lumemor_api.model.AdminUserResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier. | 
**name** | **String** | Display name. | 
**email** | **String** | Login address. | 
**role** | [**UserRole**](UserRole.md) |  | 
**isDisabled** | **bool** | The disabled switch. A disabled account loses the authority of its sessions and its sign-in until re-enabled; its data stays untouched.  | 
**createdAt** | [**DateTime**](DateTime.md) | Creation time (RFC 3339 UTC). | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


