# lumemor_api.model.TagNameRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **String** | The tag's name, trimmed server-side. On create a name the caller already owns converges on the existing tag; on rename a name another tag carries answers 409 `TAG_NAME_TAKEN`, the row's own name the idempotent no-op.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


