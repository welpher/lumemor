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
**photoCount** | **int** | The number of DISTINCT live assets the cluster has a face on that the caller can see -- photos, not faces, so two faces of one person in one photo count once. Honest per caller, never the global count.  | 
**representativeFaceId** | **String** | The face the person card renders: the person's chosen avatar face when it is a caller-visible live member of the cluster, otherwise the cluster's strongest-scoring caller-visible face (score descending, the face id ascending as the tiebreak) -- the best face the CALLER can see, never a face on an asset the caller cannot view. The field is the thumbnail's face id.  | 
**thumbnail** | **String** | URL of the card's face crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with `representativeFaceId`.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


