# lumemor_api.model.UpdateCheckResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enabled** | **bool** | Whether a check runs at all: the settings switch AND the deployment's own switch, both closed for `true`.  | 
**status** | [**UpdateCheckStatus**](UpdateCheckStatus.md) |  | 
**currentVersion** | **String** | The running server version -- the build's version string, verbatim (development builds report `dev`, which the comparison cannot rank; such an instance stays `unknown`).  | 
**latestVersion** | **String** | The newest tag the feed named, verbatim including its `v` prefix, or null when no probe has succeeded yet.  | 
**checkedAt** | [**DateTime**](DateTime.md) | When the last probe completed, successful or not; null before the first probe. The settings face renders it as the quiet \"last checked\" line when a probe failed.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


