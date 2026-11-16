# lumemor_api.model.ModelParams

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**faceDetectMinScore** | **num** | The face detection's minimum confidence, 0 to 1 inclusive: a candidate scoring below it never enters the library. The shipped default is the pipeline's own 0.6; the admin value overrides the active model's manifest entry for this knob. Contract granularity is steps of 0.05 (the page's stepper); the domain is what validates.  | 
**faceRecognizeThreshold** | **num** | The face recognition's clustering threshold, 0 to 1 inclusive: two embeddings whose cosine distance is below it belong to one person. The shipped default is the pipeline's own 0.55; the admin value overrides the active model's manifest entry for this knob. Contract granularity is steps of 0.05 (the page's stepper); the domain is what validates.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


