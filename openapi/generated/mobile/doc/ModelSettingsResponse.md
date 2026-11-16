# lumemor_api.model.ModelSettingsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**autoDownload** | **bool** | Whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own. It never fills an unconfigured slot: the first-run download stays the administrator's own act.  | 
**allowedHosts** | **BuiltList&lt;String&gt;** | The hosts the model download may travel through. A URL whose host matches no entry -- entries are exact hostnames or a `*.` wildcard prefix for a whole domain suffix -- fails the download with the host named, and the answer points at the sources panel that can add it. The shipped default list covers the manifest's own hosts; the manifest's pinned URLs stay the identity of what is downloaded, this list is only where the bytes may come from.  | 
**mirrorBaseUrl** | **String** | The download mirror's base host, the empty string when no mirror is configured. Non-empty, it replaces the manifest's pinned `huggingface.co` hostname wholesale on every candidate URL before the fetch, and its host joins `allowedHosts` automatically at save time (a mirror the list refuses would fail on its first hop). Custom model imports are not mirror-rewritten.  | 
**params** | [**ModelParams**](ModelParams.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


