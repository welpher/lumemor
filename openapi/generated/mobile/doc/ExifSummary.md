# lumemor_api.model.ExifSummary

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cameraMake** | **String** | Camera manufacturer (EXIF Make). | 
**cameraModel** | **String** | Camera model (EXIF Model). | 
**lens** | **String** | Lens identifier (LensModel, LensID or Lens). | 
**iso** | **int** | ISO sensitivity. | 
**aperture** | **num** | Aperture as the f-number. | 
**shutter** | **String** | Exposure time in photo notation, for example `1/250` or `2.5s`.  | 
**focal** | **num** | Focal length in millimeters. | 
**lat** | **double** | GPS latitude in degrees. | 
**lon** | **double** | GPS longitude in degrees. | 
**alt** | **num** | GPS altitude in meters. | 
**tzOffset** | **String** | The offset that resolved the capture time, in EXIF notation (`+05:30`); null when the capture time did not resolve through EXIF.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


