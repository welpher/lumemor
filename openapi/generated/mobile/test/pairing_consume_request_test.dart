import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for PairingConsumeRequest
void main() {
  final instance = PairingConsumeRequestBuilder();
  // TODO add properties to the builder and call build()

  group(PairingConsumeRequest, () {
    // The pairing token read from the QR code.
    // String token
    test('to test the property `token`', () async {
      // TODO
    });

    // Human-readable device label shown in the session list, for example `Pixel 9`. 
    // String deviceName
    test('to test the property `deviceName`', () async {
      // TODO
    });

    // Stable client-generated identifier of the physical device, for example a UUID the app persists; it lets the server recognize the same device across pairings. 
    // String deviceId
    test('to test the property `deviceId`', () async {
      // TODO
    });

  });
}
