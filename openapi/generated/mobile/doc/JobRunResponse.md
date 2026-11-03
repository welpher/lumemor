# lumemor_api.model.JobRunResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | UUIDv7 identifier of the run's `job_runs` row. | 
**kind** | [**JobRunKind**](JobRunKind.md) |  | 
**state** | [**JobRunState**](JobRunState.md) |  | 
**total** | **int** | The run's whole work, in the run's own units: the download group's pinned bytes, or the rerun scope's asset count.  | 
**done** | **int** | The work completed so far, in the same units as `total`.  | 
**failed** | **int** | The failed work count: failed models for the download, assets whose rerun failed for a rerun.  | 
**startedAt** | [**DateTime**](DateTime.md) | When the run's driver first picked it up (RFC 3339 UTC). Null while the run sits `queued`.  | 
**finishedAt** | [**DateTime**](DateTime.md) | When the run closed (RFC 3339 UTC). Null until then.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


