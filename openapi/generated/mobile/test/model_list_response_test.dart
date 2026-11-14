import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ModelListResponse
void main() {
  final instance = ModelListResponseBuilder();
  // TODO add properties to the builder and call build()

  group(ModelListResponse, () {
    // Every registered model, grouped by slot then id. 
    // BuiltList<ModelResponse> models
    test('to test the property `models`', () async {
      // TODO
    });

    // The most recent `model_download` run of whatever state, or null before the first download was ever started. 
    // JobRunResponse download
    test('to test the property `download`', () async {
      // TODO
    });

    // One row per pipeline slot (the contract's enum order): the four-state view the page's cards render, the slot's active model, its source facts and the download run's progress while a run is in flight. 
    // BuiltList<SlotStateResponse> slots
    test('to test the property `slots`', () async {
      // TODO
    });

    // The manifest's catalog: the validated list the switch dialog's official group renders, every entry with its download state. The image is not the entry -- the mirror carries no weights; `downloaded` is the presence probe and the digest check stays the activation's business. 
    // BuiltList<ModelCatalogEntry> official
    test('to test the property `official`', () async {
      // TODO
    });

  });
}
