# lumemor_api.model.PasswordChangeRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**current** | **String** | The account's current password; it must verify against the stored Argon2id hash before anything is written.  | 
**next** | **String** | The new password; 8 to 128 runes, stored as a fresh Argon2id hash. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


