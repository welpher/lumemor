# lumemor_api.model.Error

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **String** | Stable error code. The i18n translation key is the lowercase form of this value.  | 
**message** | **String** | English technical description for debugging. | [optional] 
**details** | [**JsonObject**](.md) | Optional structured context, for example the offending field names of a `VALIDATION_FAILED` request. Its shape is code-specific.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


