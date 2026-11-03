# lumemor_api.model.ModelListResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**models** | [**BuiltList&lt;ModelResponse&gt;**](ModelResponse.md) | Every registered model, grouped by slot then id.  | 
**download** | [**JobRunResponse**](JobRunResponse.md) | The most recent `model_download` run of whatever state, or null before the first download was ever started.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


