# lumemor_api.model.ImageSettingsResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**tileSize** | **int** | The timeline tile's bounding box in pixels: 250 is the shipped default, 320 reads sharper on high-density screens.  | 
**previewSize** | **int** | The viewer's inline preview long edge in pixels: 1080 is the shipped default (enough to look at, cheapest to derive), 1440 and 2048 zoom sharper at a bigger encode.  | 
**format** | [**DerivativeFormat**](DerivativeFormat.md) |  | 
**fullSize** | **bool** | Whether derivations also produce the full-resolution serving copy. Shipped OFF: the viewer reads the preview, and the full-size copy is disk the family never opens.  | 
**rawEmbedded** | **bool** | Whether a RAW source derives from its embedded preview. Shipped ON.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


