import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for JobRunSummary
void main() {
  final instance = JobRunSummaryBuilder();
  // TODO add properties to the builder and call build()

  group(JobRunSummary, () {
    // UUIDv7 identifier of the run's `job_runs` row.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // JobRunState state
    test('to test the property `state`', () async {
      // TODO
    });

    // The run's whole work in its own units: albums for the all-albums rescan, assets for the rebuild and the place re-resolution, pinned bytes for the model download. 
    // int total
    test('to test the property `total`', () async {
      // TODO
    });

    // The work completed so far, in the same units.
    // int done
    test('to test the property `done`', () async {
      // TODO
    });

    // The failed work count, in the same units.
    // int failed
    test('to test the property `failed`', () async {
      // TODO
    });

    // When the run's driver first picked it up (RFC 3339 UTC). Null while the run sits `queued`. 
    // DateTime startedAt
    test('to test the property `startedAt`', () async {
      // TODO
    });

    // When the run closed (RFC 3339 UTC). Null until then.
    // DateTime finishedAt
    test('to test the property `finishedAt`', () async {
      // TODO
    });

  });
}
