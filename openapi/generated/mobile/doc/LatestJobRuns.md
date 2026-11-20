# lumemor_api.model.LatestJobRuns

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**rebuildDerivatives** | [**JobRunSummary**](JobRunSummary.md) | The latest derivative rebuild run, or null. | 
**placesRerun** | [**JobRunSummary**](JobRunSummary.md) | The latest place re-resolution run, or null. | 
**albumRescanAll** | [**JobRunSummary**](JobRunSummary.md) | The latest all-albums rescan sweep, or null. | 
**folderSync** | [**JobRunSummary**](JobRunSummary.md) | The latest folder sweep run. The folder sweep is the activity-gated periodic diff and records no run row by design, so this slot answers null on every instance; the catalog's row states the cadence instead of a run.  | 
**modelDownload** | [**JobRunSummary**](JobRunSummary.md) | The latest model download run, or null. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


