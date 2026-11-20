# lumemor_api.model.PipelineJobCounts

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**waiting** | **int** | Jobs waiting for a worker: available, scheduled (a retry backoff or a pause's snooze), or retryable.  | 
**active** | **int** | Jobs a worker is running now. | 
**failed** | **int** | Jobs the queue discarded after their attempts ran out -- the failed work the row's danger line counts.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


