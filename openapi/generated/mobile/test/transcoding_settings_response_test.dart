import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TranscodingSettingsResponse
void main() {
  final instance = TranscodingSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(TranscodingSettingsResponse, () {
    // TranscodePreset preset
    test('to test the property `preset`', () async {
      // TODO
    });

    // How many video re-encodes may run at once: 1 (low load), 2 (the recommended default) or 4 (high load). 
    // int concurrency
    test('to test the property `concurrency`', () async {
      // TODO
    });

    // Whether the video thumbnail job captures a poster frame (the poster IS a video's thumbnail). Off skips the frame capture; the video's pipeline still completes. 
    // bool thumbnailFrame
    test('to test the property `thumbnailFrame`', () async {
      // TODO
    });

    // The re-encode failure policy: on, a failed re-encode completes the job quietly and the original stays in place and downloadable; off, the job fails and retries. 
    // bool keepOriginal
    test('to test the property `keepOriginal`', () async {
      // TODO
    });

  });
}
