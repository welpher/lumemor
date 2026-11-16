# lumemor_api.model.TrashBatchRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action** | **String** | The action to apply to every id in the caller's trash. `restore` clears the trash mark (the Live Photo pair moving as one asset, the external row's file moving back to its recorded original location); `delete` removes the rows and their files permanently (the shared-blob rule applies, the component video rows go with their photos).  | 
**ids** | **BuiltList&lt;String&gt;** | The asset ids to apply the action to: at least 1, at most 1000.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


