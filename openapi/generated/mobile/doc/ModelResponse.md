# lumemor_api.model.ModelResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | The model's stable slug identifier -- also the name of its directory under the models dir.  | 
**slot** | [**ModelSlot**](ModelSlot.md) |  | 
**version** | **String** | The upstream release tag of the model family, as the manifest declares it.  | 
**source_** | [**ModelSource**](ModelSource.md) |  | 
**license** | **String** | The upstream license identifier, as the model's manifest declares it (empty when the manifest is unreadable -- the row still lists so the administrator can clean it up).  | 
**notice** | **String** | The model's provenance notice line, as the manifest declares it (null when the manifest is unreadable -- the fact is unknowable; empty when the manifest declares no notice). The admin surface displays it beside the license so the attribution stays honest.  | 
**dim** | **int** | The embedding dimension the model emits. Null for models that do not embed (detectors, OCR).  | 
**inputSize** | [**ModelResponseInputSize**](ModelResponseInputSize.md) |  | 
**isActive** | **bool** | True when the model is the active one of its slot. At most one model per slot is active.  | 
**filesPresent** | **bool** | True when the model's on-disk manifest is readable and every file it pins exists -- a presence probe only. The sha256 verification runs at activation and load time, not here.  | 
**params** | [**BuiltMap&lt;String, JsonObject&gt;**](JsonObject.md) | The model's inference defaults (thresholds, normalization constants) as its manifest declares them; empty when it declares none.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


