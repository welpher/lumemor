# lumemor_api.model.ClusterDetail

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**clusterId** | **String** | Identifier of the face cluster. | 
**label** | [**PersonLabel**](PersonLabel.md) | The caller's label of the cluster, or null when the caller has not labeled it (the unnamed state). It is the calling user's label or no label at all: another account's label never shows here.  | 
**faces** | [**BuiltList&lt;FaceDetail&gt;**](FaceDetail.md) | The cluster's faces on assets visible to the caller, strongest score first with the face id as the tiebreak. Faces on other accounts' assets never appear.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


