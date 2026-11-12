import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AssetAlbumRef
void main() {
  final instance = AssetAlbumRefBuilder();
  // TODO add properties to the builder and call build()

  group(AssetAlbumRef, () {
    // Identifier of the album (the detail face's deep link).
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // The album's display title (the sync card's internal label).
    // String title
    test('to test the property `title`', () async {
      // TODO
    });

    // The album's type. A `memory` entry is one of the viewer's own or shared reference sets; `directory` entries are sources. The one `sync` entry, when present, is the viewer's own default album (never another account's). 
    // String type
    test('to test the property `type`', () async {
      // TODO
    });

    // The caller's relation to this album: `owner` items are the caller's own (the sync stream's entry is always one), `viewer` items the albums shared to them. 
    // String callerRole
    test('to test the property `callerRole`', () async {
      // TODO
    });

  });
}
