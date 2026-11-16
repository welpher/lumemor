import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ModelSettingsResponse
void main() {
  final instance = ModelSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(ModelSettingsResponse, () {
    // Whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own. It never fills an unconfigured slot: the first-run download stays the administrator's own act. 
    // bool autoDownload
    test('to test the property `autoDownload`', () async {
      // TODO
    });

    // The hosts the model download may travel through. A URL whose host matches no entry -- entries are exact hostnames or a `*.` wildcard prefix for a whole domain suffix -- fails the download with the host named, and the answer points at the sources panel that can add it. The shipped default list covers the manifest's own hosts; the manifest's pinned URLs stay the identity of what is downloaded, this list is only where the bytes may come from. 
    // BuiltList<String> allowedHosts
    test('to test the property `allowedHosts`', () async {
      // TODO
    });

    // The download mirror's base host, the empty string when no mirror is configured. Non-empty, it replaces the manifest's pinned `huggingface.co` hostname wholesale on every candidate URL before the fetch, and its host joins `allowedHosts` automatically at save time (a mirror the list refuses would fail on its first hop). Custom model imports are not mirror-rewritten. 
    // String mirrorBaseUrl
    test('to test the property `mirrorBaseUrl`', () async {
      // TODO
    });

    // ModelParams params
    test('to test the property `params`', () async {
      // TODO
    });

  });
}
