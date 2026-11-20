import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for LatestJobRuns
void main() {
  final instance = LatestJobRunsBuilder();
  // TODO add properties to the builder and call build()

  group(LatestJobRuns, () {
    // The latest derivative rebuild run, or null.
    // JobRunSummary rebuildDerivatives
    test('to test the property `rebuildDerivatives`', () async {
      // TODO
    });

    // The latest place re-resolution run, or null.
    // JobRunSummary placesRerun
    test('to test the property `placesRerun`', () async {
      // TODO
    });

    // The latest all-albums rescan sweep, or null.
    // JobRunSummary albumRescanAll
    test('to test the property `albumRescanAll`', () async {
      // TODO
    });

    // The latest folder sweep run. The folder sweep is the activity-gated periodic diff and records no run row by design, so this slot answers null on every instance; the catalog's row states the cadence instead of a run. 
    // JobRunSummary folderSync
    test('to test the property `folderSync`', () async {
      // TODO
    });

    // The latest model download run, or null.
    // JobRunSummary modelDownload
    test('to test the property `modelDownload`', () async {
      // TODO
    });

  });
}
