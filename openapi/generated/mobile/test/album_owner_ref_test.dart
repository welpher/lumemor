import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumOwnerRef
void main() {
  final instance = AlbumOwnerRefBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumOwnerRef, () {
    // Identifier of the owning account.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // The email address of the owning account: the fallback the client renders when `name` is empty, and the disambiguator between two accounts that share a name. 
    // String email
    test('to test the property `email`', () async {
      // TODO
    });

    // The owning account's display name, exactly as stored. The empty string when the account set none -- the client then renders `email` instead. Never treated as a unique key: two accounts may share a name, and the email is what tells them apart. 
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

  });
}
