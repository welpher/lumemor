# lumemor_api.model.TrashSweepResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affected** | **int** | The number of trashed rows the sweep applied: the rows the restore-all returned to the live views (the events the sweep published), or the rows the empty permanently deleted. The trash is owner-scoped, so the sweep never touches another account's rows.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


