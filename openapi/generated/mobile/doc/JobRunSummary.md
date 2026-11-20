# lumemor_api.model.JobRunSummary

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier of the run's `job_runs` row. | 
**state** | [**JobRunState**](JobRunState.md) |  | 
**total** | **int** | The run's whole work in its own units: albums for the all-albums rescan, assets for the rebuild and the place re-resolution, pinned bytes for the model download.  | 
**done** | **int** | The work completed so far, in the same units. | 
**failed** | **int** | The failed work count, in the same units. | 
**startedAt** | [**DateTime**](DateTime.md) | When the run's driver first picked it up (RFC 3339 UTC). Null while the run sits `queued`.  | 
**finishedAt** | [**DateTime**](DateTime.md) | When the run closed (RFC 3339 UTC). Null until then. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


