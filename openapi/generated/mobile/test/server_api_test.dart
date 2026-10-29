import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for ServerApi
void main() {
  final instance = LumemorApi().getServerApi();

  group(ServerApi, () {
    // Create the first administrator account.
    //
    // One-time first-run wizard endpoint, available only while the instance has no administrator. Once an admin exists the endpoint answers 409 `SETUP_ALREADY_COMPLETED` and the client should route to login.  The password is stored as an Argon2id hash; it never appears in any response. 
    //
    //Future<User> createAdmin(CreateAdminRequest createAdminRequest) async
    test('test createAdmin', () async {
      // TODO
    });

    // Report instance version and feature switches.
    //
    // Unauthenticated capability discovery. Clients call it before login to locate the API, render the first-run wizard when the instance is not yet initialized, and adapt to the features the instance enables.  `initialized` is derived from the existence of an administrator account. 
    //
    //Future<ServerInfo> getServerInfo() async
    test('test getServerInfo', () async {
      // TODO
    });

  });
}
