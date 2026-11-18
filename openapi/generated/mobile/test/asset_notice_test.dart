import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AssetNotice
void main() {
  final instance = AssetNoticeBuilder();
  // TODO add properties to the builder and call build()

  group(AssetNotice, () {
    // What happened. `LIVE_PHOTO_MOTION_ABSORBED`: the Live Photo's motion bytes already existed as a live standalone video of this owner; that row became the pair's motion (`assetId` names it) and no longer lists as its own asset. 
    // String code
    test('to test the property `code`', () async {
      // TODO
    });

    // The asset the notice is about: the reused row that became the Live Photo's motion. 
    // String assetId
    test('to test the property `assetId`', () async {
      // TODO
    });

    // The affected asset's original filename, so the message can name what moved. 
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

  });
}
