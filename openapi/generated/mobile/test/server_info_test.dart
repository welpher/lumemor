import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ServerInfo
void main() {
  final instance = ServerInfoBuilder();
  // TODO add properties to the builder and call build()

  group(ServerInfo, () {
    // Server version (semantic versioning).
    // String version
    test('to test the property `version`', () async {
      // TODO
    });

    // True once the instance has an administrator account. Clients use it to decide between the first-run wizard and the login screen. 
    // bool initialized
    test('to test the property `initialized`', () async {
      // TODO
    });

    // Instance feature switches (model slots, search modes, pairing). Unknown keys must be ignored by clients. 
    // BuiltMap<String, bool> features
    test('to test the property `features`', () async {
      // TODO
    });

  });
}
