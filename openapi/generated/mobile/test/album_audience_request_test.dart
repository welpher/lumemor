import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumAudienceRequest
void main() {
  final instance = AlbumAudienceRequestBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumAudienceRequest, () {
    // Every account that may view the album after this write, between 0 and 200 entries. Repeated ids collapse to one application. The owner's own id is skippable noise (the owner sees the album by construction), and ids that name no usable account are skipped with a reason -- never errors. 
    // BuiltList<String> audience
    test('to test the property `audience`', () async {
      // TODO
    });

  });
}
