import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AdminUserDeleteRequest
void main() {
  final instance = AdminUserDeleteRequestBuilder();
  // TODO add properties to the builder and call build()

  group(AdminUserDeleteRequest, () {
    // Identifier of the account that receives every photo, album and memory the departing account owned. Must name another existing account: the departing account itself answers 409 USER_TRANSFER_TO_SELF, an unknown id 400 VALIDATION_FAILED. 
    // String transferTo
    test('to test the property `transferTo`', () async {
      // TODO
    });

    // The alternative to `transferTo`: the departing account's photos enter the calling administrator's trash, where the instance's retention window governs their permanent removal. 
    // String dispose
    test('to test the property `dispose`', () async {
      // TODO
    });

  });
}
