import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for BearerSession
void main() {
  final instance = BearerSessionBuilder();
  // TODO add properties to the builder and call build()

  group(BearerSession, () {
    // Bearer credential of the new mobile session; send it as `Authorization: Bearer <token>`. It appears only in this response, and the server stores only its hash: a device that loses it must pair again. 
    // String token
    test('to test the property `token`', () async {
      // TODO
    });

    // SessionInfo session
    test('to test the property `session`', () async {
      // TODO
    });

  });
}
