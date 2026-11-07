# lumemor_api.model.StorageSettingsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**template** | **String** | The template new uploads render their placement from. A fresh instance (or a corrupt row) answers the default `{{y}}/{{MM}}/{{filename}}`.  | 
**samplePath** | **String** | The template rendered over the fixed sample asset with the extension appended, as a stored relative path -- exactly the placement a real upload of the sample file would land at.  | 
**presets** | **BuiltList&lt;String&gt;** | The built-in preset templates, in the settings section's dropdown order: date nestings, album layouts, camera groupings, the type grouping, the asset-id fallback and the bare file name.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


