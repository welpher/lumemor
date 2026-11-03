# lumemor_api.model.AdminCreateUserRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **String** | Optional display name; defaults to the email address. | [optional] 
**email** | **String** | Login address of the new account; unique instance-wide. | 
**password** | **String** | Account password; stored only as an Argon2id hash. | 
**role** | [**UserRole**](UserRole.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


