# lumemor_api.model.RemoveTagAssetsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**removed** | **int** | The assignment rows the request deleted. Zero for an all-skipped request, which is still a 200.  | 
**skipped** | **int** | The distinct ids that carried no assignment of this tag -- never tagged, or already untagged.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


