import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TilesSettingsUpdateRequest
void main() {
  final instance = TilesSettingsUpdateRequestBuilder();
  // TODO add properties to the builder and call build()

  group(TilesSettingsUpdateRequest, () {
    // The light tile source URL to save, or the empty string to clear it. A non-empty value must be an absolute `http` or `https` URL; a LAN address is legitimate (the browser loads the source, not the server). 
    // String tilejsonUrl
    test('to test the property `tilejsonUrl`', () async {
      // TODO
    });

    // The dark tile source URL to save. Omitting it or sending the empty string clears it (the dark theme then reuses the light source). A non-empty value follows the light field's rule: an absolute `http` or `https` URL, a LAN address included. 
    // String darkTilejsonUrl
    test('to test the property `darkTilejsonUrl`', () async {
      // TODO
    });

    // The light source's coordinate system: `true` for a GCJ-02 basemap (Amap, Tencent), `false` or omitted for the standard WGS-84. Omitting it saves the false default, so a client unaware of the flag keeps behaving exactly as before. 
    // bool gcj02 (default value: false)
    test('to test the property `gcj02`', () async {
      // TODO
    });

    // The dark source's coordinate system, on the same rule as `gcj02`: `true` for a GCJ-02 basemap, `false` or omitted for WGS-84. Ignored while the dark source is empty (the light source then governs both themes). 
    // bool darkGcj02 (default value: false)
    test('to test the property `darkGcj02`', () async {
      // TODO
    });

  });
}
