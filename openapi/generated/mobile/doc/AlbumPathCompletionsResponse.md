# lumemor_api.model.AlbumPathCompletionsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**paths** | **BuiltList&lt;String&gt;** | Full absolute directory paths inside the completion domain -- at or under a configured mount root, outside the server's managed tree, real directories (symlinked entries are never listed). Deduplicated, sorted ascending, at most 20 rows. The empty array is an honest \"nothing matches\", never an error.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


