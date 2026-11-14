import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ModelCatalogEntry
void main() {
  final instance = ModelCatalogEntryBuilder();
  // TODO add properties to the builder and call build()

  group(ModelCatalogEntry, () {
    // The manifest model's stable id.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // ModelSlot slot
    test('to test the property `slot`', () async {
      // TODO
    });

    // The display name the manifest's notice carries, derived from the model id.
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

    // The upstream license identifier, empty when undeclared.
    // String license
    test('to test the property `license`', () async {
      // TODO
    });

    // The embedding dimension; absent for models that do not embed.
    // int dim
    test('to test the property `dim`', () async {
      // TODO
    });

    // The pinned byte count of the entry's files.
    // int bytes
    test('to test the property `bytes`', () async {
      // TODO
    });

    // Whether the entry's files are present on disk (a presence probe, not a digest check).
    // bool downloaded
    test('to test the property `downloaded`', () async {
      // TODO
    });

  });
}
