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
**representativeFaceId** | **String** | The cluster's representative face (its strongest-scoring member); the face the cluster tile renders.  | 
**thumbnail** | **String** | URL of the representative face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the representative face's id.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


