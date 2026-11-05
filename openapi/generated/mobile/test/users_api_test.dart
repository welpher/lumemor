import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for UsersApi
void main() {
  final instance = LumemorApi().getUsersApi();

  group(UsersApi, () {
    // The instance's member list.
    //
    // The minimal member list of the instance: one entry per account, `email` ascending. Any authenticated account may enumerate it -- the family-instance posture: the accounts of a shared photo home are not secret from one another, and the album share dialog's pick list (the owner choosing whom to grant) is the list's first consumer.  The entries carry exactly the identity a pick list renders: `id`, `email`, `role` and `isDisabled` -- no display name, no timestamps, and a password hash never appears in any response. The disabled switch rides the entry so callers can present accounts that cannot sign in for what they are (granting to one still answers 400 through the share endpoint's own target rule).  The list is small by construction (a family instance holds a handful of accounts) and grows only through the admin surface's own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. This is NOT the admin management face (`GET /admin/users`): it is the minimal, non-managing pick list every authenticated account may read. 
    //
    //Future<UserListResponse> listInstanceUsers() async
    test('test listInstanceUsers', () async {
      // TODO
    });

  });
}
