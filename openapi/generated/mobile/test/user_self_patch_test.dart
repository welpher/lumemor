import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for UserSelfPatch
void main() {
  final instance = UserSelfPatchBuilder();
  // TODO add properties to the builder and call build()

  group(UserSelfPatch, () {
    // The new display name; 1 to 100 runes after trimming (an empty name is refused -- the profile always carries one). 
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

    // The new login address, parsed and lower-cased like every creation path; an address another account already owns answers 409 USER_EMAIL_TAKEN. 
    // String email
    test('to test the property `email`', () async {
      // TODO
    });

    // UserTheme theme
    test('to test the property `theme`', () async {
      // TODO
    });

    // UserLocale locale
    test('to test the property `locale`', () async {
      // TODO
    });

    // DownloadFormat downloadFormat
    test('to test the property `downloadFormat`', () async {
      // TODO
    });

    // The download packaging's video switch.
    // bool downloadIncludeVideos
    test('to test the property `downloadIncludeVideos`', () async {
      // TODO
    });

    // The download packaging's sidecar switch.
    // bool downloadIncludeSidecars
    test('to test the property `downloadIncludeSidecars`', () async {
      // TODO
    });

  });
}
