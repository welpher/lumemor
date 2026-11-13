# lumemor_api.model.UnnamedClusterSummary

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**clusterId** | **String** | Identifier of the face cluster. | 
**faceCount** | **int** | The cluster's faces on assets visible to the caller -- the honest per-caller count, not the global member count.  | 
**representativeFaceId** | **String** | The face the cluster tile renders: the person's chosen avatar face when it is a caller-visible live member of the cluster, otherwise the cluster's strongest-scoring caller-visible face (score descending, the face id ascending as the tiebreak) -- the best face the CALLER can see, never a face on an asset the caller cannot view.  | 
**thumbnail** | **String** | URL of the tile's face crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the caller-visible best face's id.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


