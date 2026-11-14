# lumemor_api.model.RunBudgetSettingsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**concurrencyPreset** | [**ConcurrencyPreset**](ConcurrencyPreset.md) |  | 
**nightlyEnabled** | **bool** | Whether the night window is on. Off, the heavy work runs whenever there is work.  | 
**nightlyStart** | **String** | The window's opening time, `HH:MM` local to the server. Shipped `02:00`.  | 
**nightlyEnd** | **String** | The window's closing time, `HH:MM` local to the server. Shipped `06:00`; earlier than the start is a window that spans midnight.  | 
**pausedJobs** | [**PausedJobsState**](PausedJobsState.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


