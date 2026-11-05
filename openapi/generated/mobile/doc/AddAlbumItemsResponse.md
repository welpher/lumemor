# lumemor_api.model.AddAlbumItemsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**added** | **int** | The membership rows the request inserted. Zero for an all-skipped request, which is still a 200 -- batch semantics, never an error.  | 
**skipped** | **int** | The distinct ids that did not land: unknown ids, another account's assets, trashed, hidden or component rows, and the ids already sitting in the album (the idempotent add).  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


