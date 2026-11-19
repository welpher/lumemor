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

  });
}
