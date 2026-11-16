import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TrashBatchRequest
void main() {
  final instance = TrashBatchRequestBuilder();
  // TODO add properties to the builder and call build()

  group(TrashBatchRequest, () {
    // The action to apply to every id in the caller's trash. `restore` clears the trash mark (the Live Photo pair moving as one asset, the external row's file moving back to its recorded original location); `delete` removes the rows and their files permanently (the shared-blob rule applies, the component video rows go with their photos). 
    // String action
    test('to test the property `action`', () async {
      // TODO
    });

    // The asset ids to apply the action to: at least 1, at most 1000. 
    // BuiltList<String> ids
    test('to test the property `ids`', () async {
      // TODO
    });

  });
}
