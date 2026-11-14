# lumemor_api.model.UpdateCheckSettingsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enabled** | **bool** | The switch in force at the settings layer. `false` stops every scheduled and manual check; the cached answer is kept, so switching it back on resumes from the same state.  | 
**endpoint** | **String** | The release feed URL the server probes: a JSON document whose newest tag rides `tag` (or `tag_name`). The server fetches it with a bounded timeout, through the environment's forward proxy, refusing internal addresses and never following a redirect.  | 
**intervalHours** | **int** | The cadence in hours, 1-168: the server probes at most once per interval and caches the answer until the next one.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


