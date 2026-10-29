# lumemor_api.model.BearerSession

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**token** | **String** | Bearer credential of the new mobile session; send it as `Authorization: Bearer <token>`. It appears only in this response, and the server stores only its hash: a device that loses it must pair again.  | 
**session** | [**SessionInfo**](SessionInfo.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


