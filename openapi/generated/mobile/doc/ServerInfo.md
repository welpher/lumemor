# lumemor_api.model.ServerInfo

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**version** | **String** | Server version (semantic versioning). | 
**initialized** | **bool** | True once the instance has an administrator account. Clients use it to decide between the first-run wizard and the login screen.  | 
**features** | **BuiltMap&lt;String, bool&gt;** | Instance feature switches (model slots, search modes, pairing). Unknown keys must be ignored by clients.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


