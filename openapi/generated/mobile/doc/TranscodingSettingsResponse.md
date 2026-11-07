# lumemor_api.model.TranscodingSettingsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**preset** | [**TranscodePreset**](TranscodePreset.md) |  | 
**concurrency** | **int** | How many video re-encodes may run at once: 1 (low load), 2 (the recommended default) or 4 (high load).  | 
**thumbnailFrame** | **bool** | Whether the video thumbnail job captures a poster frame (the poster IS a video's thumbnail). Off skips the frame capture; the video's pipeline still completes.  | 
**keepOriginal** | **bool** | The re-encode failure policy: on, a failed re-encode completes the job quietly and the original stays in place and downloadable; off, the job fails and retries.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


