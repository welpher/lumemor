import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ModelResponse
void main() {
  final instance = ModelResponseBuilder();
  // TODO add properties to the builder and call build()

  group(ModelResponse, () {
    // The model's stable slug identifier -- also the name of its directory under the models dir. 
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // ModelSlot slot
    test('to test the property `slot`', () async {
      // TODO
    });

    // The upstream release tag of the model family, as the manifest declares it. 
    // String version
    test('to test the property `version`', () async {
      // TODO
    });

    // ModelSource source_
    test('to test the property `source_`', () async {
      // TODO
    });

    // The upstream license identifier, as the model's manifest declares it (empty when the manifest is unreadable -- the row still lists so the administrator can clean it up). 
    // String license
    test('to test the property `license`', () async {
      // TODO
    });

    // The embedding dimension the model emits. Null for models that do not embed (detectors, OCR). 
    // int dim
    test('to test the property `dim`', () async {
      // TODO
    });

    // ModelResponseInputSize inputSize
    test('to test the property `inputSize`', () async {
      // TODO
    });

    // True when the model is the active one of its slot. At most one model per slot is active. 
    // bool isActive
    test('to test the property `isActive`', () async {
      // TODO
    });

    // True when the model's on-disk manifest is readable and every file it pins exists -- a presence probe only. The sha256 verification runs at activation and load time, not here. 
    // bool filesPresent
    test('to test the property `filesPresent`', () async {
      // TODO
    });

    // The model's inference defaults (thresholds, normalization constants) as its manifest declares them; empty when it declares none. 
    // BuiltMap<String, JsonObject> params
    test('to test the property `params`', () async {
      // TODO
    });

  });
}
