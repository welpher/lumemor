import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for UserSelfResponse
void main() {
  final instance = UserSelfResponseBuilder();
  // TODO add properties to the builder and call build()

  group(UserSelfResponse, () {
    // UUIDv7 identifier of the account.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // Display name.
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

    // Login address; unique instance-wide.
    // String email
    test('to test the property `email`', () async {
      // TODO
    });

    // UserRole role
    test('to test the property `role`', () async {
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

    // Whether the account's download packaging includes the original video files. Photos are the download's content and always ride along. 
    // bool downloadIncludeVideos
    test('to test the property `downloadIncludeVideos`', () async {
      // TODO
    });

    // Whether the account's download packaging includes EXIF sidecar files (`.xmp`) beside the media. 
    // bool downloadIncludeSidecars
    test('to test the property `downloadIncludeSidecars`', () async {
      // TODO
    });

    // Whether a download adds the motion a single-file \"motion photo\" embeds as a separate video file. Default `false`: the still's original already carries that motion, so the extra copy is a duplicate unless the account asks for it. A paired Live Photo's own motion file always rides along and this switch never removes it. 
    // bool downloadIncludeEmbeddedVideos
    test('to test the property `downloadIncludeEmbeddedVideos`', () async {
      // TODO
    });

  });
}
