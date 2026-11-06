import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for CameraFacetCount
void main() {
  final instance = CameraFacetCountBuilder();
  // TODO add properties to the builder and call build()

  group(CameraFacetCount, () {
    // The round-trip filter value: `make|model` with both halves trimmed, empty where the EXIF row carries no value. 
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // The EXIF camera make, null when absent.
    // String make
    test('to test the property `make`', () async {
      // TODO
    });

    // The EXIF camera model, null when absent.
    // String model
    test('to test the property `model`', () async {
      // TODO
    });

    // The number of assets of the current filtered set (minus the camera filter) shot with this camera. 
    // int assetCount
    test('to test the property `assetCount`', () async {
      // TODO
    });

  });
}
