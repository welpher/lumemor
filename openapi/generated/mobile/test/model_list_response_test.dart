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

  });
}
