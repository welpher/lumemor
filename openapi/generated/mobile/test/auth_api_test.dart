import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for AuthApi
void main() {
  final instance = LumemorApi().getAuthApi();

  group(AuthApi, () {
    // Exchange a pairing token for a mobile session.
    //
    // Completes device pairing: the mobile app scans the QR code and posts the token together with its device metadata. The server burns the one-time token and returns the bearer credential of a new mobile session of the user who created the token.  Bearer sessions never expire; the app holds the token until the session is revoked -- from the device list, by logging the device out, or by a password change, which revokes every session of the user. The server stores only the hash of the token, so it appears exactly once, in this response; a device that loses it must pair again.  Failed attempts are rate limited to stop guessing tokens. 
    //
    //Future<BearerSession> consumePairingToken(PairingConsumeRequest pairingConsumeRequest) async
    test('test consumePairingToken', () async {
      // TODO
    });

    // Create a one-time device pairing token.
    //
    // Issues the secret behind the add-a-family-device QR code: a signed-in user opens the pairing page, and the new device scans the rendered code and redeems it at `POST /auth/pairing/consume` to gain a session of that user.  The token encodes 128 random bits as a URL-safe string, can be consumed once within five minutes, and is stored only as a hash, so a leaked database cannot redeem it. Creation is rate limited per user to stop farming QR codes. 
    //
    //Future<PairingTokenResponse> createPairingToken() async
    test('test createPairingToken', () async {
      // TODO
    });

    // List the caller's active sessions.
    //
    // Returns every active session of the authenticated user across devices -- web cookie sessions and mobile bearer sessions -- most recently used first. The device management view (signing out other devices) is this list plus `DELETE /auth/sessions/{id}`. 
    //
    //Future<BuiltList<SessionInfo>> listSessions() async
    test('test listSessions', () async {
      // TODO
    });

    // Exchange credentials for a web session.
    //
    // The login endpoint of web clients: a valid email and password create a new web session. The session credential is the `lumemor_session` cookie set by this response (HttpOnly, SameSite=Strict, Secure over HTTPS); the body describes the session for the account view and the device list. Mobile devices never log in here -- they pair through `POST /auth/pairing/consume` and hold a bearer token instead.  Web sessions slide their expiry by 30 days on use. Failed attempts are rate limited per address and account: after a burst of failures the endpoint answers 429 and the lockout grows with each further attempt. 
    //
    //Future<SessionInfo> login(LoginRequest loginRequest) async
    test('test login', () async {
      // TODO
    });

    // Revoke the current session.
    //
    // Signs the caller out by revoking the session that authenticates the request -- the web cookie session and the mobile bearer session are the same session records. On web clients the response also clears the session cookie. A request made with an already-revoked session fails authentication with 401 before reaching this handler. 
    //
    //Future logout() async
    test('test logout', () async {
      // TODO
    });

    // Revoke one of the caller's sessions.
    //
    // Signs out one device by revoking the session with the given id, for example from the device management list. Sessions are scoped to their owner and the delete follows set semantics: an id that is unknown, belongs to another user, or is already revoked still answers 204, without revealing whether such a session exists. Revoking the session of the current request behaves like logout. 
    //
    //Future revokeSession(String id) async
    test('test revokeSession', () async {
      // TODO
    });

  });
}
