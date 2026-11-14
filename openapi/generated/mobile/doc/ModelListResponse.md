# lumemor_api.model.ModelListResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**models** | [**BuiltList&lt;ModelResponse&gt;**](ModelResponse.md) | Every registered model, grouped by slot then id.  | 
**download** | [**JobRunResponse**](JobRunResponse.md) | The most recent `model_download` run of whatever state, or null before the first download was ever started.  | 
**slots** | [**BuiltList&lt;SlotStateResponse&gt;**](SlotStateResponse.md) | One row per pipeline slot (the contract's enum order): the four-state view the page's cards render, the slot's active model, its source facts and the download run's progress while a run is in flight.  | 
**official** | [**BuiltList&lt;ModelCatalogEntry&gt;**](ModelCatalogEntry.md) | The manifest's catalog: the validated list the switch dialog's official group renders, every entry with its download state. The image is not the entry -- the mirror carries no weights; `downloaded` is the presence probe and the digest check stays the activation's business.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


