import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TilesSettingsResponse
void main() {
  final instance = TilesSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(TilesSettingsResponse, () {
    // The light tile source URL: a style document (vector) or a TileJSON descriptor (raster), loaded by the browser in the light theme and in the dark theme when no dark style is configured. The empty string is the shipped state: no source, and the map draws its pins over the stated absence of a base layer. 
    // String tilejsonUrl
    test('to test the property `tilejsonUrl`', () async {
      // TODO
    });

    // The dark tile source URL, same shapes as the light one. The empty string means the dark theme reuses `tilejsonUrl` -- one source, one look in both themes. A configured value is loaded by the dark theme only. 
    // String darkTilejsonUrl
    test('to test the property `darkTilejsonUrl`', () async {
      // TODO
    });

  });
}
