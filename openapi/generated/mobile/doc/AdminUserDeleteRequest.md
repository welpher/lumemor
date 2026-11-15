# lumemor_api.model.AdminUserDeleteRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**transferTo** | **String** | Identifier of the account that receives every photo, album and memory the departing account owned. Must name another existing account: the departing account itself answers 409 USER_TRANSFER_TO_SELF, an unknown id 400 VALIDATION_FAILED.  | [optional] 
**dispose** | **String** | The alternative to `transferTo`: the departing account's photos enter the calling administrator's trash, where the instance's retention window governs their permanent removal.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


