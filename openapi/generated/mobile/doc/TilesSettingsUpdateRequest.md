# lumemor_api.model.TilesSettingsUpdateRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**tilejsonUrl** | **String** | The light tile source URL to save, or the empty string to clear it. A non-empty value must be an absolute `http` or `https` URL; a LAN address is legitimate (the browser loads the source, not the server).  | 
**darkTilejsonUrl** | **String** | The dark tile source URL to save. Omitting it or sending the empty string clears it (the dark theme then reuses the light source). A non-empty value follows the light field's rule: an absolute `http` or `https` URL, a LAN address included.  | [optional] 
**gcj02** | **bool** | The light source's coordinate system: `true` for a GCJ-02 basemap (Amap, Tencent), `false` or omitted for the standard WGS-84. Omitting it saves the false default, so a client unaware of the flag keeps behaving exactly as before.  | [optional] [default to false]
**darkGcj02** | **bool** | The dark source's coordinate system, on the same rule as `gcj02`: `true` for a GCJ-02 basemap, `false` or omitted for WGS-84. Ignored while the dark source is empty (the light source then governs both themes).  | [optional] [default to false]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


