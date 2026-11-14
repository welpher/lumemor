# lumemor_api.model.ModelProbeResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**probeId** | **String** |  | 
**state** | **String** |  | 
**progress** | [**ModelProbeResponseProgress**](ModelProbeResponseProgress.md) |  | 
**dims** | **int** | The model's output dimension, read off the graph. | 
**inputSize** | [**ModelProbeResponseInputSize**](ModelProbeResponseInputSize.md) |  | 
**sha256** | **String** | The staged bytes' digest, set when the probe finished (ok or warn). | 
**hostDeclared** | **bool** | Whether the source's host is on the download sources list. | 
**reason** | **String** | The failing check, named, for state `error`. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


