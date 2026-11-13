# lumemor_api.model.UserSummary

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier. | 
**email** | **String** | Login address: the identity's unique key, the fallback a pick list renders when the display name is empty, and the disambiguator between two accounts that share a name.  | 
**name** | **String** | The account's display name, exactly as stored. The empty string when the account set none -- the client then renders `email` instead.  | 
**role** | [**UserRole**](UserRole.md) |  | 
**isDisabled** | **bool** | The disabled switch: a disabled account cannot sign in and its sessions carry no authority until re-enabled.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


