# lumemor_api.model.MergePeopleRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**sourceClusterIds** | **BuiltList&lt;String&gt;** | The clusters to fold into the target: at least 1 and at most 50 after duplicates collapse, every one visible to the caller. Each dissolves with its label rows. A single id is the pairwise fold of the cluster detail's \"merge into\" action.  | 
**targetClusterId** | **String** | The surviving cluster: every source face re-homes here and its representative recomputes to the strongest member of the merged set. It must be visible to the caller and must not sit among the sources.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


