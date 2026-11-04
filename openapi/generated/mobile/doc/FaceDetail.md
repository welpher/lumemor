# lumemor_api.model.FaceDetail

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | Identifier of the face row. | 
**assetId** | **String** | Identifier of the asset the face sits on. | 
**score** | **double** | The detector's confidence for this face. | 
**thumbnail** | **String** | URL of the face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with this face's id, serving the bounding-box crop of the asset's view-size derivative.  | 
**asset** | [**FaceDetailAsset**](FaceDetailAsset.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


