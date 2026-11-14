import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for UpdateCheckResponse
void main() {
  final instance = UpdateCheckResponseBuilder();
  // TODO add properties to the builder and call build()

  group(UpdateCheckResponse, () {
    // Whether a check runs at all: the settings switch AND the deployment's own switch, both closed for `true`. 
    // bool enabled
    test('to test the property `enabled`', () async {
      // TODO
    });

    // UpdateCheckStatus status
    test('to test the property `status`', () async {
      // TODO
    });

    // The running server version -- the build's version string, verbatim (development builds report `dev`, which the comparison cannot rank; such an instance stays `unknown`). 
    // String currentVersion
    test('to test the property `currentVersion`', () async {
      // TODO
    });

    // The newest tag the feed named, verbatim including its `v` prefix, or null when no probe has succeeded yet. 
    // String latestVersion
    test('to test the property `latestVersion`', () async {
      // TODO
    });

    // When the last probe completed, successful or not; null before the first probe. The settings face renders it as the quiet \"last checked\" line when a probe failed. 
    // DateTime checkedAt
    test('to test the property `checkedAt`', () async {
      // TODO
    });

  });
}
