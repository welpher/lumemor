import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for JobRunResponse
void main() {
  final instance = JobRunResponseBuilder();
  // TODO add properties to the builder and call build()

  group(JobRunResponse, () {
    // UUIDv7 identifier of the run's `job_runs` row.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // JobRunKind kind
    test('to test the property `kind`', () async {
      // TODO
    });

    // JobRunState state
    test('to test the property `state`', () async {
      // TODO
    });

    // The run's whole work, in the run's own units: the download group's pinned bytes, or the rerun scope's asset count. 
    // int total
    test('to test the property `total`', () async {
      // TODO
    });

    // The work completed so far, in the same units as `total`. 
    // int done
    test('to test the property `done`', () async {
      // TODO
    });

    // The failed work count: failed models for the download, assets whose rerun failed for a rerun. 
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
