# lumemor_api.model.CameraFacetCount

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | The round-trip filter value: `make|model` with both halves trimmed, empty where the EXIF row carries no value.  | 
**make** | **String** | The EXIF camera make, null when absent. | [optional] 
**model** | **String** | The EXIF camera model, null when absent. | [optional] 
**assetCount** | **int** | The number of assets of the current filtered set (minus the camera filter) shot with this camera.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


