# lumemor_api.model.AlbumCreateResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | Identifier of the album. | 
**title** | **String** | The album's display title. | 
**type** | **String** | The album's type. `memory` albums are reference sets curated by their owner -- photos live in the albums they were ingested through, a memory holds references to them; `directory` albums mirror a filesystem path. The one `sync` card, when present, is the caller's own default album pinned first (never another account's): its `title` is the internal system label ('Sync') and clients render the localized display name instead (en \"My photos\", zh-Hans \"我的照片\").  | 
**isShared** | **bool** | True when the album carries at least one `album_shares` row -- someone besides the owner can see it. Always true on shared-group items by construction.  | 
**coverAssetIds** | **BuiltList&lt;String&gt;** | The collage source of the wall card: the pinned cover asset as a single-element list when the owner set one, otherwise the album's most recent live member assets (up to four, `takenAt` descending, the asset id as the tiebreak). Empty for an album with no live members.  | 
**itemCount** | **int** | The album's live member count: trashed, hidden and Live Photo component rows never count (a hidden asset is invisible to the whole family).  | 
**updatedAt** | [**DateTime**](DateTime.md) | When the album row last changed (RFC 3339 UTC). | 
**callerRole** | **String** | The caller's relation to this album: `owner` items belong to the caller's own group on the wire, `viewer` items to the shared group. The client renders ONE wall list and keys each card's face off this discriminator (the read-only mark is the member's, the shared state the owner's).  | 
**owner** | [**AlbumOwnerRef**](AlbumOwnerRef.md) | The owner attribution of a shared item: the account that owns the album. Present exactly when `callerRole` is `viewer`; own items omit it (the owner is the caller).  | [optional] 
**inTimeline** | **bool** | The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly on the excludable cards -- the own `directory` items and every shared item; the own `memory` items omit it (they never feed the caller's own timeline, so there is nothing to exclude), and the pinned `sync` card omits it too (the core stream is never excludable). The value is the preference row's switch, true when no row exists (the default).  | [optional] 
**sourceWritable** | **bool** | The creation-time write probe's fact of the album this response just created: `true` when the probe file landed, read back and removed cleanly; `false` for a readable but read-only source. The creation SUCCEEDED either way -- a read-only directory is a legitimate album -- and `false` is the honest feedback the dialog renders (photos of this directory cannot be deleted through the application).  | 
**skipped** | [**BuiltList&lt;AlbumAudienceSkip&gt;**](AlbumAudienceSkip.md) | The requested audience ids that stayed ungranted: the album's own owner and every id that names no usable account. Empty when the whole audience landed.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


