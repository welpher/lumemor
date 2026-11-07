import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for PasswordChangeRequest
void main() {
  final instance = PasswordChangeRequestBuilder();
  // TODO add properties to the builder and call build()

  group(PasswordChangeRequest, () {
    // The account's current password; it must verify against the stored Argon2id hash before anything is written. 
    // String current
    test('to test the property `current`', () async {
      // TODO
    });

    // The new password; 8 to 128 runes, stored as a fresh Argon2id hash.
    // String next
    test('to test the property `next`', () async {
      // TODO
    });

  });
}
