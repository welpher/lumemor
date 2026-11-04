# lumemor_api.model.AssetFaceSummary

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | Identifier of the face row. | 
**score** | **double** | The detector's confidence for this face. | 
**thumbnail** | **String** | URL of the face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with this face's id, serving the bounding-box crop of the asset's view-size derivative.  | 
**clusterId** | **String** | Identifier of the face's cluster, or null before the clustering assigned one (a face the recognition run has not grouped yet).  | 
**label** | [**PersonLabel**](PersonLabel.md) | The caller's own label of the face's cluster, or null when the caller has not labeled it (the unnamed state) -- and null for a clusterless face, which has no cluster to label.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


