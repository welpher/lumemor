import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AddTagAssetsResponse
void main() {
  final instance = AddTagAssetsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(AddTagAssetsResponse, () {
    // The assignment rows the request inserted. Zero for an all-skipped request, which is still a 200 -- batch semantics, never an error. 
    // int added
    test('to test the property `added`', () async {
      // TODO
    });

    // The distinct ids that did not land: unknown ids, another account's unshared assets, any id the caller cannot see, and the ids already carrying the tag (the idempotent add). 
    // int skipped
    test('to test the property `skipped`', () async {
      // TODO
    });

  });
}
