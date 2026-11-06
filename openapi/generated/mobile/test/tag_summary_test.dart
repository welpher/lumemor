import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TagSummary
void main() {
  final instance = TagSummaryBuilder();
  // TODO add properties to the builder and call build()

  group(TagSummary, () {
    // UUIDv7 identifier of the tag (the caller's own row).
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // The tag's display name, unique within the caller's vocabulary. Trimmed on write; 1-64 Unicode characters (counted over runes). 
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

    // The tag's assignment rows joined to caller-visible live assets: the album union plus the timeline's live filters (not trashed, not hidden, no Live Photo component). A trashed or hidden member stops counting until it returns; the assignment row survives underneath, exactly like a favorite row. 
    // int assetCount
    test('to test the property `assetCount`', () async {
      // TODO
    });

    // When the tag row was created (RFC 3339 UTC).
    // DateTime createdAt
    test('to test the property `createdAt`', () async {
      // TODO
    });

    // When the tag's name last truly changed (RFC 3339 UTC). The idempotent create and the no-op rename leave it untouched -- only a real rename moves it. 
    // DateTime updatedAt
    test('to test the property `updatedAt`', () async {
      // TODO
    });

  });
}
