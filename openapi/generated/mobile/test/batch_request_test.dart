import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for BatchRequest
void main() {
  final instance = BatchRequestBuilder();
  // TODO add properties to the builder and call build()

  group(BatchRequest, () {
    // The set operation to apply. `favorite` / `unfavorite` write or remove the caller's per-user favorites rows; `hide` / `unhide` flip the owner-level visibility switch (an archived asset leaves every live view until unarchived); `trash` / `restore` run the trash lifecycle of the trash endpoints, the Live Photo pair moving as one asset. 
    // String action
    test('to test the property `action`', () async {
      // TODO
    });

    // The asset ids to apply the action to: at least 1, at most 1000. 
    // BuiltList<String> ids
    test('to test the property `ids`', () async {
      // TODO
    });

  });
}
