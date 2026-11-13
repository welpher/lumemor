# lumemor_api.model.UnnamedClustersResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**BuiltList&lt;UnnamedClusterSummary&gt;**](UnnamedClusterSummary.md) | The page's clusters, ordered by `faceCount` descending with the cluster id as the tiebreak.  | 
**nextCursor** | **String** | The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page.  | [optional] 
**totalCount** | **int** | The caller's whole unnamed region: every cluster this listing would walk, independent of the page size and the cursor. The people list's unnamed entry badge and the unnamed page's own count both read it.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


