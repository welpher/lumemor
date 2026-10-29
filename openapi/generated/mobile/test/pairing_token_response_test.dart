import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for PairingTokenResponse
void main() {
  final instance = PairingTokenResponseBuilder();
  // TODO add properties to the builder and call build()

  group(PairingTokenResponse, () {
    // One-time pairing secret, 128 random bits as a URL-safe string. Render it as a QR code and never store or log it; the server keeps only its hash. 
    // String token
    test('to test the property `token`', () async {
      // TODO
    });

    // When the token expires (RFC 3339 UTC); five minutes after issuance. 
    // DateTime expiresAt
    test('to test the property `expiresAt`', () async {
      // TODO
    });

  });
}
