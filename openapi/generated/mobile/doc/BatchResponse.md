# lumemor_api.model.BatchResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affected** | **int** | How many distinct request ids resolved to a caller-owned asset and were processed -- INCLUDING ids that were already in the target state (the idempotent no-op of the set semantics: a repeat of a successful batch answers the same count, because the endpoint is a convergence operation, not a state-change counter). NOT a count of rows written: trashing a Live Photo pair moves two rows but one request id, and counts once.  | 
**missing** | **BuiltList&lt;String&gt;** | The distinct request ids that did not resolve to a caller-owned asset -- unknown ids and other accounts' assets, indistinguishable by design -- in the order of their first appearance in the request. Empty when every id resolved. This deliberate reporting (instead of the byte-identical 404 of the single-asset endpoints) is the trade of the list surface: the caller handed the ids over themselves, so reporting them back hides nothing.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


