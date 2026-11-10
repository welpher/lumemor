# lumemor_api.model.AlbumPathCompletionsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**completions** | [**BuiltList&lt;AlbumPathCompletion&gt;**](AlbumPathCompletion.md) | The pickable directories of the container's visible filesystem -- outside the server's managed tree, real directories (symlinked entries are never listed). Deduplicated, sorted ascending by path, at most 20 rows. The empty array is an honest \"nothing matches\", never an error. Every row carries its write fact: the probe file was written, read back and removed cleanly (`writable` true), or any step refused (false -- a read-only directory is a legitimate source, and a false fact is information, never an error).  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


