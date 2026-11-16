# lumemor_api.model.TrashResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**BuiltList&lt;TrashItem&gt;**](TrashItem.md) | The trashed assets of the caller, ordered by `deletedAt` descending.  | 
**retentionDays** | **int** | The instance's trash retention window in whole days, the same setting the listing's `remainingDays` countdowns were computed against (`trash_retention_days`, default 30). The trash page's note renders it; adjusting the setting stays the administrator's surface, but the number itself is visible to every signed-in member. A corrupt setting row answers the default 30 -- the same degradation the countdowns follow.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


