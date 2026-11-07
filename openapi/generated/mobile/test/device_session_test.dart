import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for DeviceSession
void main() {
  final instance = DeviceSessionBuilder();
  // TODO add properties to the builder and call build()

  group(DeviceSession, () {
    // UUIDv7 identifier of the session.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // Where the session credential lives: `web` sessions ride the `lumemor_session` cookie, `mobile` sessions ride a bearer token. 
    // String kind
    test('to test the property `kind`', () async {
      // TODO
    });

    // Human-readable device label: the browser's user agent captured at login for web sessions, the name supplied during pairing for mobile ones. 
    // String deviceName
    test('to test the property `deviceName`', () async {
      // TODO
    });

    // When the session was last used (RFC 3339 UTC).
    // DateTime lastSeenAt
    test('to test the property `lastSeenAt`', () async {
      // TODO
    });

    // The address the session was created from (the login's or pairing exchange's socket peer), captured at creation time. Omitted when the row predates the column or no address was captured; an audit hint, not a geolocation claim. 
    // String ipAddress
    test('to test the property `ipAddress`', () async {
      // TODO
    });

    // Whether this row is the session the request itself authenticates with -- the device the caller is managing from. Exactly one entry of a list carries `true`, and that entry cannot be revoked through `DELETE /users/me/sessions/{id}` (the sign-out flow is its exit). 
    // bool isCurrent
    test('to test the property `isCurrent`', () async {
      // TODO
    });

  });
}
