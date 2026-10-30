# lumemor_api.model.BatchRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action** | **String** | The set operation to apply. `favorite` / `unfavorite` write or remove the caller's per-user favorites rows; `hide` / `unhide` flip the owner-level visibility switch (a hidden asset leaves every live view until unhidden); `trash` / `restore` run the trash lifecycle of the trash endpoints, the Live Photo pair moving as one asset.  | 
**ids** | **BuiltList&lt;String&gt;** | The asset ids to apply the action to: at least 1, at most 1000.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


