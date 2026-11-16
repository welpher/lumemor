import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ModelParams
void main() {
  final instance = ModelParamsBuilder();
  // TODO add properties to the builder and call build()

  group(ModelParams, () {
    // The face detection's minimum confidence, 0 to 1 inclusive: a candidate scoring below it never enters the library. The shipped default is the pipeline's own 0.6; the admin value overrides the active model's manifest entry for this knob. Contract granularity is steps of 0.05 (the page's stepper); the domain is what validates. 
    // num faceDetectMinScore
    test('to test the property `faceDetectMinScore`', () async {
      // TODO
    });

    // The face recognition's clustering threshold, 0 to 1 inclusive: two embeddings whose cosine distance is below it belong to one person. The shipped default is the pipeline's own 0.55; the admin value overrides the active model's manifest entry for this knob. Contract granularity is steps of 0.05 (the page's stepper); the domain is what validates. 
    // num faceRecognizeThreshold
    test('to test the property `faceRecognizeThreshold`', () async {
      // TODO
    });

  });
}
