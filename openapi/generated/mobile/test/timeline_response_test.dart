import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TimelineResponse
void main() {
  final instance = TimelineResponseBuilder();
  // TODO add properties to the builder and call build()

  group(TimelineResponse, () {
    // The timeline items of this page, ordered by `takenAt` descending with the id as the tiebreak. 
    // BuiltList<TimelineItem> items
    test('to test the property `items`', () async {
      // TODO
    });

    // The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page. 
    // String nextCursor
    test('to test the property `nextCursor`', () async {
      // TODO
    });

  });
}
