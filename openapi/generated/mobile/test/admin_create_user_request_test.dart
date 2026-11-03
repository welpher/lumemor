import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AdminCreateUserRequest
void main() {
  final instance = AdminCreateUserRequestBuilder();
  // TODO add properties to the builder and call build()

  group(AdminCreateUserRequest, () {
    // Optional display name; defaults to the email address.
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

    // Login address of the new account; unique instance-wide.
    // String email
    test('to test the property `email`', () async {
      // TODO
    });

    // Account password; stored only as an Argon2id hash.
    // String password
    test('to test the property `password`', () async {
      // TODO
    });

    // UserRole role
    test('to test the property `role`', () async {
      // TODO
    });

  });
}
