import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumAudienceSkip
void main() {
  final instance = AlbumAudienceSkipBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumAudienceSkip, () {
    // The requested id that stayed ungranted.
    // String userId
    test('to test the property `userId`', () async {
      // TODO
    });

    // Why the id was skipped: `owner` -- the album's own owner, whose visibility is the ownership row, never a share row; `invalid_account` -- an unknown or disabled account (the same reason for both, so user ids cannot be probed). 
    // String reason
    test('to test the property `reason`', () async {
      // TODO
    });

  });
}
