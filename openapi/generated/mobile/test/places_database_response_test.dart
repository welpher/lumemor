import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for PlacesDatabaseResponse
void main() {
  final instance = PlacesDatabaseResponseBuilder();
  // TODO add properties to the builder and call build()

  group(PlacesDatabaseResponse, () {
    // The data-version marker the startup import recorded -- the sha256 digest of the exact loaded bytes. An empty string means the instance carries no bake (nothing resolves). A 64-character value is a content digest, not a release date: the card displays its first eight characters. 
    // String version
    test('to test the property `version`', () async {
      // TODO
    });

    // The display-name locales the bake carries, in the product's language-set order. The places read's `locale` parameter accepts exactly this set. 
    // BuiltList<String> languages
    test('to test the property `languages`', () async {
      // TODO
    });

    // The full re-resolution scope: every asset whose exif row carries GPS coordinates, whatever its lifecycle state. A row with a manual location is excluded -- the walk leaves those rows to the owner's own decision, so the count and the walk cover the same rows. 
    // int gpsAssets
    test('to test the property `gpsAssets`', () async {
      // TODO
    });

    // The fill-in scope: the GPS-bearing assets whose place is absent (the subset a `missing` run walks), with the same manual-location exclusion as `gpsAssets`. 
    // int missingPlaces
    test('to test the property `missingPlaces`', () async {
      // TODO
    });

  });
}
