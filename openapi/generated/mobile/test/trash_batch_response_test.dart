import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TrashBatchResponse
void main() {
  final instance = TrashBatchResponseBuilder();
  // TODO add properties to the builder and call build()

  group(TrashBatchResponse, () {
    // How many distinct request ids were in the caller's trash at the time of the request and were processed. The trash batch's guard is trash-scoped, so an id already live or already purged lands in `missing` instead of counting here (the deliberate difference from the assets batch, whose state-agnostic guard counts an already-in-state row as affected): the trash view's rows are the caller's whole world, and a row that left it between render and click is a missing row, not a no-op. 
    // int affected
    test('to test the property `affected`', () async {
      // TODO
    });

    // The distinct request ids that were not in the caller's trash at the time of the request -- unknown ids, other accounts' assets (indistinguishable by design), rows already restored to the live views and rows the expiry job already purged -- in the order of their first appearance in the request. Empty when every id was still in the trash. 
    // BuiltList<String> missing
    test('to test the property `missing`', () async {
      // TODO
    });

  });
}
