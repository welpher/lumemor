# lumemor_api.model.TrashBatchResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affected** | **int** | How many distinct request ids were in the caller's trash at the time of the request and were processed. The trash batch's guard is trash-scoped, so an id already live or already purged lands in `missing` instead of counting here (the deliberate difference from the assets batch, whose state-agnostic guard counts an already-in-state row as affected): the trash view's rows are the caller's whole world, and a row that left it between render and click is a missing row, not a no-op.  | 
**missing** | **BuiltList&lt;String&gt;** | The distinct request ids that were not in the caller's trash at the time of the request -- unknown ids, other accounts' assets (indistinguishable by design), rows already restored to the live views and rows the expiry job already purged -- in the order of their first appearance in the request. Empty when every id was still in the trash.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


