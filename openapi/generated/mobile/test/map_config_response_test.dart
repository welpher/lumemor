import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for MapConfigResponse
void main() {
  final instance = MapConfigResponseBuilder();
  // TODO add properties to the builder and call build()

  group(MapConfigResponse, () {
    // The light tile source URL the browser loads directly from the tile host (a style document or a TileJSON descriptor); null means no source is configured and the map renders its stated absence. The dark theme falls back to this source when `darkTilejsonUrl` is null. The pin data never crosses the tile line: it comes from `GET /places`. 
    // String tilejsonUrl
    test('to test the property `tilejsonUrl`', () async {
      // TODO
    });

    // The dark tile source URL, loaded by the dark theme instead of `tilejsonUrl`; null (or, equivalently, the instance not configuring one) means the dark theme reuses the light source. The light theme never loads it. 
    // String darkTilejsonUrl
    test('to test the property `darkTilejsonUrl`', () async {
      // TODO
    });

  });
}
