# lumemor_api.model.MapConfigResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**tilejsonUrl** | **String** | The light tile source URL the browser loads directly from the tile host (a style document or a TileJSON descriptor); null means no source is configured and the map renders its stated absence. The dark theme falls back to this source when `darkTilejsonUrl` is null. The pin data never crosses the tile line: it comes from `GET /places`.  | 
**darkTilejsonUrl** | **String** | The dark tile source URL, loaded by the dark theme instead of `tilejsonUrl`; null (or, equivalently, the instance not configuring one) means the dark theme reuses the light source. The light theme never loads it.  | 
**gcj02** | **bool** | Whether the light source draws in GCJ-02 (the offset datum Amap and Tencent use inside China). False is the standard WGS-84 every other source draws in, and the shipped default. The browser converts the pins into the active source's datum for display and converts a picked point back to WGS-84 before it reaches the library -- the stored coordinates never leave WGS-84.  | 
**darkGcj02** | **bool** | Whether the dark source draws in GCJ-02, on the same rule as `gcj02`. When no dark source is configured the dark theme reuses the light source and this flag is not consulted.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


