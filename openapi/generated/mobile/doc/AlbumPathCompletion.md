# lumemor_api.model.AlbumPathCompletion

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**path** | **String** | Full absolute directory path of the container's visible filesystem, never inside the server's managed tree.  | 
**writable** | **bool** | The directory's write fact as the listing discovered it: true when the probe file landed, read back and removed cleanly; false when any step refused. The fact is information, never an error -- a false row lists alongside the true ones, and picking it creates a legitimate read-only-source album.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


