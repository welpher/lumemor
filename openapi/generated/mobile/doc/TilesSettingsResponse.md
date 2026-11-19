# lumemor_api.model.TilesSettingsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**tilejsonUrl** | **String** | The light tile source URL: a style document (vector) or a TileJSON descriptor (raster), loaded by the browser in the light theme and in the dark theme when no dark style is configured. The empty string is the shipped state: no source, and the map draws its pins over the stated absence of a base layer.  | 
**darkTilejsonUrl** | **String** | The dark tile source URL, same shapes as the light one. The empty string means the dark theme reuses `tilejsonUrl` -- one source, one look in both themes. A configured value is loaded by the dark theme only.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


