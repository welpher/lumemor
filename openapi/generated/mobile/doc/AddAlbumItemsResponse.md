# lumemor_api.model.AddAlbumItemsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**added** | **int** | The membership rows the request landed -- the photos the creator currently sees and that were not members yet. Zero for an all-skipped request, which is still a 200.  | 
**skipped** | **int** | The distinct ids that stayed out of the set: not currently visible to the caller (the re-share ground failed), not a live row, or already a member (the idempotent no-op).  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


