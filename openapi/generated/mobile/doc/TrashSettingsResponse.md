# lumemor_api.model.TrashSettingsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**retentionDays** | **int** | The retention window of a soft-deleted asset in whole days, 0 to 365. A fresh instance (or a corrupt row) answers the default 30; 0 means the next expiry run purges everything trashed.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


