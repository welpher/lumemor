# lumemor_api.model.MountRootsSettingsUpdateRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**roots** | **BuiltList&lt;String&gt;** | Every mount root of the new allowlist, at most 32 entries. Each must be an absolute path within the 4,096-byte bound; the server cleans each value and collapses duplicates after the clean. Existence is not required -- a root may be configured before the volume is mounted.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


