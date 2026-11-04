# lumemor_api.model.PersonSummary

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**clusterId** | **String** | Identifier of the face cluster. | 
**name** | **String** | The caller's own label of the cluster. | 
**faceCount** | **int** | The cluster's faces on assets visible to the caller -- the honest per-caller count, not the global member count.  | 
**representativeFaceId** | **String** | The cluster's representative face (its strongest-scoring member); the face the person card renders.  | 
**thumbnail** | **String** | URL of the representative face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the representative face's id.  | 
**isHidden** | **bool** | The caller's hidden switch for this person. Hidden people stay listed carrying this flag (badge semantics); they are not filtered out.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


