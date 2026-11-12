# lumemor_api.model.MemoryCreateRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**title** | **String** | The memory's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length).  | 
**audience** | **BuiltList&lt;String&gt;** | The accounts the memory starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


