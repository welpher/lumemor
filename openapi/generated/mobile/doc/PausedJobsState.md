# lumemor_api.model.PausedJobsState

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**thumbnail** | **bool** | Holds new thumbnail derivations -- and, with them, the derivative rebuild (its work is thumbnail-class).  | 
**metadata** | **bool** | Holds new metadata jobs. | 
**transcode** | **bool** | Holds new video transcodes. | 
**ml** | **bool** | Holds new machine learning work: the four per-asset inference jobs and the slot reruns.  | 
**places** | **bool** | Holds new place re-resolution runs. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


