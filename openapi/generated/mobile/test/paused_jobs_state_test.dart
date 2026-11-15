import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for PausedJobsState
void main() {
  final instance = PausedJobsStateBuilder();
  // TODO add properties to the builder and call build()

  group(PausedJobsState, () {
    // Holds new thumbnail derivations -- and, with them, the derivative rebuild (its work is thumbnail-class). 
    // bool thumbnail
    test('to test the property `thumbnail`', () async {
      // TODO
    });

    // Holds new metadata jobs.
    // bool metadata
    test('to test the property `metadata`', () async {
      // TODO
    });

    // Holds new video transcodes.
    // bool transcode
    test('to test the property `transcode`', () async {
      // TODO
    });

    // Holds new machine learning work: the four per-asset inference jobs and the slot reruns. 
    // bool ml
    test('to test the property `ml`', () async {
      // TODO
    });

    // Holds new place re-resolution runs.
    // bool places
    test('to test the property `places`', () async {
      // TODO
    });

  });
}
