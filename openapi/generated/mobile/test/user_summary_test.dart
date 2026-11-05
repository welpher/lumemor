import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for UserSummary
void main() {
  final instance = UserSummaryBuilder();
  // TODO add properties to the builder and call build()

  group(UserSummary, () {
    // UUIDv7 identifier.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // Login address.
    // String email
    test('to test the property `email`', () async {
      // TODO
    });

    // UserRole role
    test('to test the property `role`', () async {
      // TODO
    });

    // The disabled switch: a disabled account cannot sign in and its sessions carry no authority until re-enabled. 
    // bool isDisabled
    test('to test the property `isDisabled`', () async {
      // TODO
    });

  });
}
