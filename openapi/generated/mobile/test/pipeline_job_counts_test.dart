import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for PipelineJobCounts
void main() {
  final instance = PipelineJobCountsBuilder();
  // TODO add properties to the builder and call build()

  group(PipelineJobCounts, () {
    // Jobs waiting for a worker: available, scheduled (a retry backoff or a pause's snooze), or retryable. 
    // int waiting
    test('to test the property `waiting`', () async {
      // TODO
    });

    // Jobs a worker is running now.
    // int active
    test('to test the property `active`', () async {
      // TODO
    });

    // Jobs the queue discarded after their attempts ran out -- the failed work the row's danger line counts. 
    // int failed
    test('to test the property `failed`', () async {
      // TODO
    });

  });
}
