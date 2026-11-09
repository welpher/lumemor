# lumemor_api.model.AlbumCreateRequest

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**title** | **String** | The album's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length).  | 
**rootPath** | **String** | The source directory: an existing absolute path the server validates against the path matrix (existence, directory-ness, readability, symlink resolution, the managed tree) before creating anything. The album row stores the RESOLVED absolute path -- symlink segments evaluated -- so a later symlink swap cannot repoint the album. After the matrix passes, the server probes the directory's writability once and records the fact; a read-only directory is a valid source, never a refusal.  | 
**audience** | **BuiltList&lt;String&gt;** | The accounts the album starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


