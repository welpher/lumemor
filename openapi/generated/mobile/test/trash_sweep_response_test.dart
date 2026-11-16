import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TrashSweepResponse
void main() {
  final instance = TrashSweepResponseBuilder();
  // TODO add properties to the builder and call build()

  group(TrashSweepResponse, () {
    // The number of trashed rows the sweep applied: the rows the restore-all returned to the live views (the events the sweep published), or the rows the empty permanently deleted. The trash is owner-scoped, so the sweep never touches another account's rows. 
    // int affected
    test('to test the property `affected`', () async {
      // TODO
    });

  });
}
