# lumemor_api.model.FaceBoxPatchResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**faceId** | **String** | Identifier of the rewritten face. | 
**thumbnail** | **String** | URL of the face's crop, which now derives from the new box (the crop endpoint computes at query time; the change signal is the face row's updated_at).  | 
**bbox** | [**FaceBbox**](FaceBbox.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


