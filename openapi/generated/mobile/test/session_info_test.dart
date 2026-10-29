import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for SessionInfo
void main() {
  final instance = SessionInfoBuilder();
  // TODO add properties to the builder and call build()

  group(SessionInfo, () {
    // UUIDv7 identifier of the session.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // Where the session credential lives: `web` sessions ride the `lumemor_session` cookie and slide their expiry by 30 days on use; `mobile` sessions ride a bearer token and are revoked explicitly. 
    // String kind
    test('to test the property `kind`', () async {
      // TODO
    });

    // Human-readable device label of the session. For web sessions it is the browser user agent captured at login; mobile sessions carry the name supplied during pairing. 
    // String deviceName
    test('to test the property `deviceName`', () async {
      // TODO
    });

    // When the session was created (RFC 3339 UTC).
    // DateTime createdAt
    test('to test the property `createdAt`', () async {
      // TODO
    });

    // When the session was last used (RFC 3339 UTC).
    // DateTime lastSeenAt
    test('to test the property `lastSeenAt`', () async {
      // TODO
    });

    // When the session expires (RFC 3339 UTC). Null for bearer sessions, which are revoked explicitly instead of expiring. 
    // DateTime expiresAt
    test('to test the property `expiresAt`', () async {
      // TODO
    });

  });
}
