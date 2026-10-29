# lumemor_api.model.User

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
**role** | **String** | Authorization role; `admin` is granted by the first-run wizard, later accounts are created by an admin. | 
**createdAt** | [**DateTime**](DateTime.md) | Creation time (RFC 3339 UTC). | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


