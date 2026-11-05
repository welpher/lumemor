# lumemor_api.model.UserSummary

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier. | 
**email** | **String** | Login address. | 
**role** | [**UserRole**](UserRole.md) |  | 
**isDisabled** | **bool** | The disabled switch: a disabled account cannot sign in and its sessions carry no authority until re-enabled.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


