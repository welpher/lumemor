import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumAudienceResponse
void main() {
  final instance = AlbumAudienceResponseBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumAudienceResponse, () {
    // Identifier of the album the audience belongs to.
    // String albumId
    test('to test the property `albumId`', () async {
      // TODO
    });

    // The grantee list after the write, one entry per `album_shares` row in the grant order (`grantedAt` ascending) -- the same list the owner's album detail carries as `shares`, the edit dialog's chips. 
    // BuiltList<AlbumShareEntry> audience
    test('to test the property `audience`', () async {
      // TODO
    });

    // The ids whose rows THIS write landed -- empty for the idempotent repeat of the stored set. One `album.shared` event fired per entry. 
    // BuiltList<String> granted
    test('to test the property `granted`', () async {
      // TODO
    });

    // The ids whose rows THIS write removed -- empty for the idempotent repeat. One `album.unshared` event fired per entry. 
    // BuiltList<String> revoked
    test('to test the property `revoked`', () async {
      // TODO
    });

    // The requested ids that stayed ungranted, each with its reason. Empty when the whole request was actionable. 
    // BuiltList<AlbumAudienceSkip> skipped
    test('to test the property `skipped`', () async {
      // TODO
    });

  });
}
