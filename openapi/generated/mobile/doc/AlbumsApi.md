# lumemor_api.api.AlbumsApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**addAlbumItems**](AlbumsApi.md#addalbumitems) | **POST** /albums/{albumId}/items | Add references to one memory.
[**completeAlbumPath**](AlbumsApi.md#completealbumpath) | **GET** /albums/path-completions | Complete a directory album&#39;s source path.
[**createAlbum**](AlbumsApi.md#createalbum) | **POST** /albums | Create one directory album from an existing path.
[**createMemory**](AlbumsApi.md#creatememory) | **POST** /memories | Create one memory.
[**deleteAlbum**](AlbumsApi.md#deletealbum) | **DELETE** /albums/{albumId} | Remove one album.
[**getAlbum**](AlbumsApi.md#getalbum) | **GET** /albums/{albumId} | One album&#39;s detail and one page of its items.
[**listAlbums**](AlbumsApi.md#listalbums) | **GET** /albums | The caller&#39;s album wall.
[**removeAlbumItems**](AlbumsApi.md#removealbumitems) | **DELETE** /albums/{albumId}/items | Remove assets from one memory album.
[**rescanAlbum**](AlbumsApi.md#rescanalbum) | **POST** /albums/{albumId}/rescan | Start one rescan of a directory album&#39;s source.
[**resetAlbumTimelinePreference**](AlbumsApi.md#resetalbumtimelinepreference) | **DELETE** /albums/{albumId}/timeline-preference | Reset the caller&#39;s timeline preference of one album.
[**setAlbumTimelinePreference**](AlbumsApi.md#setalbumtimelinepreference) | **PUT** /albums/{albumId}/timeline-preference | Set the caller&#39;s timeline preference of one album.
[**updateAlbum**](AlbumsApi.md#updatealbum) | **PATCH** /albums/{albumId} | Patch one album&#39;s properties.
[**updateAlbumAudience**](AlbumsApi.md#updatealbumaudience) | **PUT** /albums/{albumId}/audience | Set one album&#39;s audience (the full grantee set).


# **addAlbumItems**
> AddAlbumItemsResponse addAlbumItems(albumId, albumItemsRequest)

Add references to one memory.

Adds membership rows to the caller's memory: `{added, skipped}`, the batch surface's set semantics -- the honest report, never per-item errors. Repeated ids collapse; the counts cover the distinct ids.  Every id the request names must pass the re-share ground: the creator must CURRENTLY see the photo (it is a member of at least one album that feeds the caller's view -- their sync stream, their directory albums, or an album shared to them) AND it must be a live row (not trashed, not hidden, not a Live Photo component). An id failing either check is skipped -- the row never lands -- and so is an id that is already a member (the set's idempotent no-op); an all-skipped request still answers 200 with `added` 0. The check runs at ADD time, once: what the write admits is what the creator could see when they made the request.  MEMORY albums only: the membership write is the memory's own curation face. A DIRECTORY album refuses it with the single-shape 404 -- a directory album is a source, and its membership is its own scan's verdict, never a hand-picked set; the sync album folds into the same 404 (it is no album resource). Authorization is the writes' two-tier answer (invisible 404, shared-to-caller 403 -- the share is view-only, and curation is the owner's -- the owner or an admin override writes).  The write publishes one `album.items-changed` event exactly when at least one row landed -- the idempotent no-op stays silent. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumItemsRequest albumItemsRequest = {"assetIds":["0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6072","0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6073"]}; // AlbumItemsRequest | 

try {
    final response = api.addAlbumItems(albumId, albumItemsRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->addAlbumItems: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumItemsRequest** | [**AlbumItemsRequest**](AlbumItemsRequest.md)|  | 

### Return type

[**AddAlbumItemsResponse**](AddAlbumItemsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **completeAlbumPath**
> AlbumPathCompletionsResponse completeAlbumPath(prefix)

Complete a directory album's source path.

The source-path completion of the album creation dialog: the typed prefix in, the pickable directories out. The call is ADMIN-scoped: mounting an existing photo directory is the administrator's act, so the completion that browses the filesystem on the dialog's behalf is the administrator's too -- a signed-in member answers 403 `AUTH_FORBIDDEN` before the prefix is even read (the members' contribution path is the upload stream; a member never needs to see the container's filesystem).  The completion domain is the container's REAL visible filesystem: whatever the operating system lets the server read, drilled segment by segment. The compose file's mounts ARE the boundary -- the operator decides what the container sees, and the completion honestly reflects that visibility instead of a software-layer allowlist (the retirement: the mount-root allowlist that used to confine this endpoint died whole, with its settings face and its validation branches). The ONE exclusion that stays is the server's own managed data tree: no row ever names the application's originals, derivatives or trash, whatever prefix would match them -- the same confine the creation's `ALBUM_PATH_IN_MANAGED_TREE` branch enforces, kept honest on both sides of the pair.  The matching rule, driven by the typed prefix:  - an empty (or absent) prefix lists the filesystem root   itself (`/`) -- the dialog's first open, \"start here\"; - a prefix of exactly `/` lists the root's top-level   directories (the separator says \"I am at the root, show me   what it holds\"); - otherwise the entries of the prefix's parent directory   whose names start with the prefix's last segment are   listed as full paths (typing `/mnt/fam` lists   `/mnt/family-archive`); - a trailing separator switches to the children of the   cleaned prefix itself (typing `/mnt/family-archive/`   drills into it), and no extra row rides along then -- the   trailing slash says \"I am inside this directory\".  Only real directories are listed (symlinked entries are not), the rows are deduplicated, sorted ascending, and capped at 20 -- past the cap the user narrows by typing; there is no pagination. A prefix that matches nothing answers the empty list, never an error -- a relative prefix and an unreadable parent both answer `completions: []`.  Every row carries its write fact: as the listing lands, the server probes each row's directory once with the creation-time probe's own three steps (write a uniquely named probe file, read it back, remove it). A probe step's refusal is a FACT (`writable: false`), never an error -- the row lists alongside the others and the dialog renders the deletability switch only for the `writable` rows.  The completion runs a bounded read: at most one directory listing and at most 20 per-row probes, and the response carries at most 20 rows. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String prefix = /mnt/fam; // String | The path fragment typed so far. Absent or empty lists the filesystem root itself. Values past 4,096 bytes answer 400 -- the filesystem's own path ceiling, so no honest typing ever hits it. 

try {
    final response = api.completeAlbumPath(prefix);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->completeAlbumPath: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **prefix** | **String**| The path fragment typed so far. Absent or empty lists the filesystem root itself. Values past 4,096 bytes answer 400 -- the filesystem's own path ceiling, so no honest typing ever hits it.  | [optional] 

### Return type

[**AlbumPathCompletionsResponse**](AlbumPathCompletionsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **createAlbum**
> AlbumCreateResponse createAlbum(albumCreateRequest)

Create one directory album from an existing path.

Creates one directory album owned by the calling ADMINISTRATOR: the album IS a source -- the request names an existing directory the server validates and references in place, the files stay where they are (zero copy, zero move; the rescan that fills the album's membership is a later flow). No type field exists: this endpoint IS directory creation -- the curation-into-a-container story is gone (a memory is born through its own every-member face, POST /memories), and every account's sync album is born with the account.  Mounting an existing photo directory is a CONFIGURATION act: it points the application at a filesystem location, and a plain member never sees the container's filesystem at all. Creation is therefore the administrator's alone (the boundary ruling): a signed-in member whose role is not `admin` answers 403 `AUTH_FORBIDDEN` before any part of the request is considered -- the refusal-matrix semantics of the admin surfaces. Members contribute photos through the upload stream and are untouched by this guard. The trust boundary is the compose file's own mounts: the operator controls what the container can see, and the application layer neither can nor needs to restrict the operator further -- the mount-root allowlist of the previous design retired whole with this ruling, and no software-layer allowlist stands behind it.  The title is trimmed and must span 1-64 Unicode characters (the count is over runes, so multi-byte titles stay within the bound at the same length).  `rootPath` is the source directory, and it runs the path validation matrix before anything is written, each branch its own 400 error code:  - the path must be absolute (after a server-side clean; a   relative, empty or over-long value is a plain   `VALIDATION_FAILED`), - it must exist (`ALBUM_PATH_NOT_FOUND`), - it must be a directory (`ALBUM_PATH_NOT_DIRECTORY`), - the server must be able to list it   (`ALBUM_PATH_NOT_READABLE`), - the RESOLVED path (symlinks evaluated) must never sit   inside the server's own managed data tree   (`ALBUM_PATH_IN_MANAGED_TREE`) -- the one confine that   survived the allowlist retirement: the application's own   originals, derivatives and trash never become an album   source, whatever the container can see. The   allowlist-domain and root-overlap branches of the previous   matrix died with the allowlist: with creation in the   administrator's hands, a second album over the same tree   is an operator's visible choice, not an attack to refuse.  The album row stores the RESOLVED absolute path, so a later symlink swap can never repoint an existing album.  After the matrix passes, the server probes the directory's writability exactly ONCE: it writes a uniquely named probe file, reads it back, and removes it. A directory that refuses any step is a perfectly legitimate album source -- the probe's failure is NOT a creation failure -- and the fact lands on the row as `sourceWritable: false`, honestly reported in the response: photos of a read-only directory cannot be deleted through the application, and the creation answer says so instead of every later deletion discovering it. The truth is discovered at creation and never re-probed at runtime.  `audience` is the album's audience, the full grantee set of view ACL rows the creation lands in the same request: the add-style picker's chips of the dialog, empty (or absent) meaning \"only me\". Its semantics are the audience write's: every id that cannot be granted -- an unknown or disabled account, or the caller's own id (the owner sees the album by construction) -- is skipped, never an error, and the response's `skipped` list reports each with its reason.  The two dialog switches ride the same request. The deletability switch (`allowAppDeletion`) lands on the row -- on by default, rendered by the dialog only where the source is writable (the completion row's `writable` fact; the read-only sources' stored value governs nothing). The creator's timeline exclusion (`inTimeline: false`) lands the caller's own preference row in the same creation -- the album is born out of its creator's aggregated timeline, the audience's own feeds untouched.  The response is the created album's wall card -- the summary exactly as `GET /albums` lists it: `callerRole` `owner`, `type` `directory`, an empty `coverAssetIds` array (never null) and `itemCount` 0, nothing being a member yet -- plus the honest audience report and the `sourceWritable` fact. The card's `inTimeline` carries the exclusion's landed state. The write publishes one `album.created` event and one `album.shared` event per audience row that landed. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final AlbumCreateRequest albumCreateRequest = {"title":"Family archive","rootPath":"/Volumes/Photos/family","audience":["0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70"]}; // AlbumCreateRequest | 

try {
    final response = api.createAlbum(albumCreateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->createAlbum: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumCreateRequest** | [**AlbumCreateRequest**](AlbumCreateRequest.md)|  | 

### Return type

[**AlbumCreateResponse**](AlbumCreateResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **createMemory**
> MemoryCreateResponse createMemory(memoryCreateRequest)

Create one memory.

Creates one memory owned by the calling user: a reference set the owner curates photos into -- the photos themselves live in the albums they were ingested through, the memory holds references to them. The request carries a title and an audience, nothing else: a memory owns no `rootPath` (no source to name -- the directory album's own field is absent from this request and from the memory's every response shape), no deletability switch (a memory's delete never moves files), and no timeline switch (a memory never feeds its creator's own timeline, so there is nothing to switch).  EVERY member may create. Curation is personal expression, not configuration: a memory names no filesystem location, touches no source, and borrows nothing but photos the creator already sees -- the exact opposite gate of the directory album's administrator-only creation. Any signed-in account's POST succeeds; there is no role check to refuse.  The title is trimmed and must span 1-64 Unicode characters (the count is over runes, so multi-byte titles stay within the bound at the same length).  `audience` is the memory's audience, the full grantee set of view ACL rows the creation lands in the same request: absent (or empty) means \"only me\". Its semantics are the audience write's: every id that cannot be granted -- an unknown or disabled account, or the caller's own id (the owner sees the memory by construction) -- is skipped, never an error, and the response's `skipped` list reports each with its reason.  The memory is born out of its creator's aggregated timeline: the creation writes the creator's own timeline preference row (`inTimeline: false`) in the same write, so the aggregate never feeds the memory's photos whatever future arms admit. The exclusion is the creator's own -- the audience members' feeds are governed by their own preference rows, never by this write.  The response is the created memory's wall card -- the summary exactly as `GET /albums` lists it: `callerRole` `owner`, `type` `memory`, an empty `coverAssetIds` array (never null), `itemCount` 0 (nothing is a member yet), no `inTimeline` field (an own memory card omits it) -- widened with the honest audience report. The write publishes one `album.created` event and one `album.shared` event per audience row that landed. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final MemoryCreateRequest memoryCreateRequest = {"title":"Road trip 2026","audience":["0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70"]}; // MemoryCreateRequest | 

try {
    final response = api.createMemory(memoryCreateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->createMemory: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **memoryCreateRequest** | [**MemoryCreateRequest**](MemoryCreateRequest.md)|  | 

### Return type

[**MemoryCreateResponse**](MemoryCreateResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteAlbum**
> deleteAlbum(albumId)

Remove one album.

Removes the caller's album. The answer is 204 with no body, and the write publishes one `album.deleted` event -- the wall and the detail converge through it on every connected client.  The removal's database half is ONE transaction: the album row goes, and every `album_items` membership row, every `album_shares` row and every per-user timeline preference row follows through the schema's cascades.  What happens to the album's MEMBERS splits by the album's type. A `memory` album holds references, not photos: removing it takes only its own rows -- the album row, its membership rows, its audience and its per-user preference rows go, and NOTHING else does. The photos are not the memory's: they live in the albums they were ingested through, and every one of their states (the asset rows, the per-user favorites and tags, the faces, the generated files) is untouched, whatever their storage kind -- no guard on other albums' references runs, nothing is purged, no file is contacted. A `directory` album is a source, and removing it is the UN-INGEST of what its scans brought in: every member asset that NO OTHER album references (the album union -- the sync album's permanent membership covers every uploaded row, and a member shared with a second album survives on that reference) is cleaned from the library entirely -- the asset row and its per-user states (favorites, tag assignments) go, its faces follow (a face cluster left empty dissolves with its labels, a surviving member re-homes the cluster's representative), its CLIP embedding and OCR text go, and the GENERATED files (the thumbnails and the transcode outputs) are deleted. The row deletion is a hard delete: the library data is gone, not trashed. The member's SOURCE file is NEVER touched -- the album only ever referenced it where it lies, and the file remains the family's own. An external member currently in the trash (its file in the server's trash area) loses the row and the trash copy together. The file removals run after the transaction commits and are best-effort: a file the operating system refuses is left behind as inert garbage, never as a row that names it. Every purged member publishes one `asset.deleted` event (the Live Photo component rows stay silent, exactly like every other purge), so the timeline, the search and the people surfaces converge on the removal while it happens.  The sync album is no album resource: it answers the single-shape 404, whatever the caller's role. The same 404 answers an unknown id.  Authorization is the writes' two-tier answer: an album the caller cannot see answers the single-shape 404; an album shared to the caller answers 403 `AUTH_FORBIDDEN` (the share is view-only); the owner -- or an administrator overriding the owner check -- removes. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.

try {
    api.deleteAlbum(albumId);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->deleteAlbum: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getAlbum**
> AlbumDetail getAlbum(albumId, cursor, limit)

One album's detail and one page of its items.

The detail view of one album the caller can see: the caller owns it, or it is shared to the caller through an `album_shares` row. `callerRole` says which -- `owner` for the owner's write affordances, `viewer` for the read-only share -- with the one exception below.  The caller's OWN sync album answers 200: the default album's read-only detail. `callerRole` is `owner`, the live members page through the same keyset as any album's (the grid and the viewer work like any album's), but the album is system-managed: NO write affordance applies to it -- the property patch, the membership curation, the sharing face, the timeline preference and the delete all keep refusing a sync album with their unchanged 404/400 answers, whatever this read admits. `inTimeline` is absent (the core stream is never excludable) and the owner's `shares` list rides the response always empty (no share row can name a sync album). The client renders the localized display name (en \"My photos\", zh-Hans \"我的照片\") keyed by `type: 'sync'` -- the stored title is the internal system label and never renders.  A sync album of ANOTHER account answers the same 404 as an unknown id, whatever share rows exist: invisible by design.  404 is single-shape by design: an unknown id, another account's unshared album, and another account's sync album all answer the same `ALBUM_NOT_FOUND` byte for byte -- the anti-enumeration discipline of `ASSET_NOT_FOUND` applied to the album face. An album whose share to the caller was revoked before the request simply sits in that same invisible region.  `itemCount` counts the album's live members; `items` is one keyset page of them, ordered `takenAt` descending with the asset id as the tiebreak -- the timeline's own ordering and cursor pair, so the album grid pages with the timeline's walking machinery. Every item carries the membership's `addedAt` and the minimal asset context the grid renders; `thumbnail` is the asset thumbnail endpoint's `tile`-size URL. Live members only: trashed, hidden and Live Photo component rows never appear (a hidden asset is invisible to the whole family).  For a `memory` seen by a VIEWER (a memory shared to the caller), the set is narrower: the count and the pages carry the live members the memory's owner CURRENTLY sees -- the re-share ground, revalidated at read time. When the owner's sight of a photo dies (the album that carried it to the owner was unshared, the row was trashed or hidden), the photo leaves every shared view of the memory live: the next read drops it, with no snapshot residue. The owner's own detail keeps the plain live membership -- the curation is the owner's own view of the set, and the owner's rows can still be tidied through the membership remove.  `coverAssetId` is the raw pinned column: the single asset the owner chose, or null when unset -- the wall derives its collage fallback from the members, the detail carries the pin itself.  `rootPath` rides the `directory` albums only: the album's source, the resolved absolute path the row stores -- the edit dialog's read-only source line (the source is pinned at creation; the property patch carries no field for it). Memory and sync albums omit it: a memory has no source -- its photos live in the albums they were ingested through.  `sourceWritable` rides beside it, on the `directory` albums only: the creation-time write probe's fact, false for a read-only source. `allowAppDeletion` and `deletable` ride the owner's view of a `directory` album only: the deletability switch's stored state (the edit dialog's toggle) and the wall's delete-affordance derivation (owner, writable source, switch on -- computed server-side, pure data, no runtime probing). A viewer's response carries none of the three faces a write would need, and a read-only source never carries a true `deletable` whatever the switch says.  `shares` rides only the owner's view: the album's grantee list, one entry per `album_shares` row in the grant order (`grantedAt` ascending), each carrying the grantee's id, email and grant time -- the share management dialog's list. The owner's response always carries the field, an empty array when the album holds no grants; viewer responses never carry it at all (a sharee sees the album, never the ACL).  `inTimeline` is the caller's timeline preference of this album (the exclusion toggle's source of truth): present on the excludable types -- the caller's own `directory` albums and the albums shared to them -- and absent on the caller's own `memory` albums. The preference scopes the aggregate: it never filters this album's own items, so browsing a switched-off album still shows everything.  `limit` bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 `VALIDATION_FAILED`. Pages continue through the opaque `cursor`, the timeline's exact keyset semantics: `nextCursor` is present exactly when another page follows, and the last page carries none. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final String cursor = GN4WpRPw0AABkvShW2x9jp8KKzxNXm9w; // String | The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
final int limit = 60; // int | Page size: the number of album items one response carries at most. Defaults to 60; values below 1 or over 200 answer 400. 

try {
    final response = api.getAlbum(albumId, cursor, limit);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->getAlbum: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **cursor** | **String**| The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients).  | [optional] 
 **limit** | **int**| Page size: the number of album items one response carries at most. Defaults to 60; values below 1 or over 200 answer 400.  | [optional] [default to 60]

### Return type

[**AlbumDetail**](AlbumDetail.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listAlbums**
> AlbumWallResponse listAlbums()

The caller's album wall.

The album wall of the calling user: every album that deserves a wall card, in two groups concatenated into one array -- the web client renders its sections off the `callerRole` discriminator each item carries.  The caller's sync album is pinned FIRST, before the own group: the one card of type `sync`, the caller's own core upload stream surfaced as the default \"My photos\" album -- every upload lands in it, so a fresh account's wall is never empty. The own group follows: the albums the caller owns of type `memory` or `directory`, the most recently updated first (the album id descending as the tiebreak). The shared group follows last: every album shared to the caller through an `album_shares` row, the caller's most recently granted share first (the album id descending as the tiebreak). Shared items carry `owner` -- the honest attribution of another account's album: the owner's id and email.  A sync album of ANOTHER account never appears, in either group: the shared arm carries `memory` and `directory` albums only, whatever share rows exist. The pinned card's `title` is the system's internal fixed label ('Sync') -- the display name is the CLIENT's: `type: 'sync'` keys the localized default-album name (en \"My photos\", zh-Hans \"我的照片\"), and the stored title never renders.  `coverAssetIds` is the wall card's collage source: the pinned `coverAssetId` as a single-element list when the owner set one, otherwise the album's most recent live member assets -- up to four, `takenAt` descending with the asset id as the tiebreak, fewer when the album holds fewer members, empty for an album with none. `itemCount` counts live members under the same predicate: trashed, hidden and Live Photo component rows never count (a hidden asset is invisible to the whole family). For a shared-group `memory` card the predicate widens the same way the detail's does: the count and the collage tiles are the live members the memory's owner currently sees -- the re-share ground, revalidated at read time, so a revoked photo leaves the sharee's card and collage with no snapshot residue.  `isShared` reports whether the album carries at least one `album_shares` row -- the owner's \"someone else sees this\" badge. It is true by construction on every shared-group item (the caller's share row is the reason the item exists).  `inTimeline` rides the excludable cards -- the own `directory` items and every shared item: the caller's timeline preference of that album, true when no preference row exists (the default of feeding the timeline). The own `memory` items omit the field: they never feed the caller's own timeline. The pinned `sync` card omits it too: the core stream is never excludable (the preference face answers 400 for it). The field is the toggle's source of truth for the wall's \"excluded\" badge, kept current by the `PUT/DELETE /albums/{albumId}/timeline-preference` endpoints. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();

try {
    final response = api.listAlbums();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->listAlbums: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**AlbumWallResponse**](AlbumWallResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **removeAlbumItems**
> RemoveAlbumItemsResponse removeAlbumItems(albumId, albumItemsRequest)

Remove assets from one memory album.

Removes membership rows from the caller's memory album: `{removed, skipped}`, the batch surface's set semantics -- the honest report, never per-item errors. An id that is no member of the album -- never was, or already removed -- is skipped; an all-skipped request still answers 200 with `removed` 0. Repeated ids collapse; the counts cover the distinct ids.  The rows this face removes are the rows the add lands: the remove is the set's own tidy-up, dropping the references the owner no longer wants beside them.  The removal carries no live predicate: a trashed member's row still leaves, the honest cleanup of the set (the live views filter trashed members out of the renders; removing the row tidies the set itself).  MEMORY albums only (directory and sync albums answer the single-shape 404); authorization is the writes' two-tier answer (invisible 404, shared-to-caller 403, the owner or an admin override writes).  The write publishes one `album.items-changed` event exactly when at least one row was removed -- the idempotent no-op stays silent. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumItemsRequest albumItemsRequest = {"assetIds":["0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6072","0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6073"]}; // AlbumItemsRequest | 

try {
    final response = api.removeAlbumItems(albumId, albumItemsRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->removeAlbumItems: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumItemsRequest** | [**AlbumItemsRequest**](AlbumItemsRequest.md)|  | 

### Return type

[**RemoveAlbumItemsResponse**](RemoveAlbumItemsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **rescanAlbum**
> AlbumRescanResponse rescanAlbum(albumId)

Start one rescan of a directory album's source.

The rescan trigger: one run that walks the directory album's source root on the server and reconciles the album's membership with what the filesystem holds. The walk never follows symlinks -- a symlinked entry is skipped, a symlinked subdirectory is never descended -- and each walk entry and each live member resolves through exactly one arm of the three-arm diff:  - a file on disk no live member matches is ADDED by   reference-style ingest: a new asset row   (`storage_kind` `external`, `file_path` the file's real   absolute path, zero copy zero move) and its album   membership land in one transaction, and the existing   pipeline jobs (metadata, thumbnail, and for a video the   transcode) run over that file exactly like an   upload's. An owner+checksum duplicate of content the owner   already holds never creates a second row: a live duplicate   gains the album membership, a trashed duplicate is left   alone (no revival semantics). - a live member whose file is gone from disk is marked   `deleted_at` directly (the orphan arm's pinned semantics:   the file is not the server's and the trash has no meaning   for it) -- no missing state, no revival. If the file   returns, the next scan's add arm ingests it as a NEW row. - a member whose mtime+size snapshot changed has its   checksum re-read and stored, the snapshot re-stamped, and   its metadata pipeline re-run (EXIF re-extracted, thumbnail   regenerated, the playable copy of a video re-encoded); the   ML arm follows the pipeline's own re-run path, which does   not re-fire -- readiness is already stamped.  The run's `job_runs` row (`kind` `album_rescan`, carrying the album's id) surfaces progress the way the places re-resolution run does: `job.started` / `job.progress` / `job.finished` on the activity stream, `total`/`done` counting resolved walk entries plus vanished members and `failed` the entries the run could not resolve. The idempotent rescan is the pinned contract: a second run over an unchanged directory performs zero asset writes and publishes zero asset events (the run's own `job_runs` bookkeeping is the progress surface, not product state).  The endpoint is idempotent the way the places re-resolution is, scoped to the album: a rescan of THIS album still in flight (`queued` or `running`) answers 200 with its identifier and starts nothing; another album's rescan never blocks this one.  Permission is the album writes' own: the owner (or an administrator) proceeds, a sharee answers the honest 403, everyone else folds into the album 404. A non-directory album -- the caller's own sync album, a memory album -- answers the same 404: the rescan is the directory album's own flow (a memory holds references to photos that live elsewhere -- there is no source to walk), and ids stay unproammable. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the directory album to rescan.

try {
    final response = api.rescanAlbum(albumId);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->rescanAlbum: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the directory album to rescan. | 

### Return type

[**AlbumRescanResponse**](AlbumRescanResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **resetAlbumTimelinePreference**
> resetAlbumTimelinePreference(albumId)

Reset the caller's timeline preference of one album.

Removes the calling user's preference row of one album: the album returns to the default state of feeding the caller's timeline (a missing row means kept in). The eligibility scope is the PUT's own -- the caller's own directory albums and the albums shared to the caller; the same 400 `album_pref_not_excludable` answers the caller's own sync and own memory albums, and the same single-shape 404 answers an album the caller cannot see.  The delete is idempotent: an album with no preference row (never set, or already reset) still answers 204 -- the row is gone either way, which is the state the caller asked for. Like the PUT it publishes no event: the preference is the caller's own state, and no other session has anything to refetch. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.

try {
    api.resetAlbumTimelinePreference(albumId);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->resetAlbumTimelinePreference: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **setAlbumTimelinePreference**
> AlbumTimelinePreferenceResponse setAlbumTimelinePreference(albumId, albumTimelinePreferenceRequest)

Set the caller's timeline preference of one album.

Upserts the calling user's per-user timeline preference of one album: the `user_album_prefs` row keyed (userId, albumId) whose `inTimeline` switch says whether the album feeds the caller's aggregated timeline. The preference is the CALLER's own state over an album they can see -- self-affecting only, which is why these writes publish no event: the caller's own response is the only view that changed, and no other session has anything to refetch. The scope is the albums whose membership a caller could want out of their timeline feed: the caller's own `directory` albums, and every album shared to the caller (whatever the owner's type). Two kinds of album answer 400 instead, with the `album_pref_not_excludable` message key in the details:  - the caller's own `sync` album: it IS the core stream, and   the core stream is never excludable (the aggregation   formula exempts the sync arm from the preference row --   a stray row there excludes nothing); - the caller's own `memory` albums: they never feed the   caller's own timeline at all (in-stream curation would be   pure duplication), so there is nothing to exclude.  An album the caller cannot see -- unknown, another account's unshared album, a revoked share -- answers the same byte-identical 404 `ALBUM_NOT_FOUND` as every album face, so album ids cannot be probed for existence through this endpoint. There is no administrator override: the preference is always the caller's own row over their own view, never a write onto another account's state.  `inTimeline` is required and must be a JSON boolean: a body that is not valid JSON, or one without the field, answers 400 `VALIDATION_FAILED`. The write is an upsert: repeating it with the same value converges to the same row, and repeating it with the other value flips the switch -- the response always carries the stored state. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumTimelinePreferenceRequest albumTimelinePreferenceRequest = {"inTimeline":false}; // AlbumTimelinePreferenceRequest | 

try {
    final response = api.setAlbumTimelinePreference(albumId, albumTimelinePreferenceRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->setAlbumTimelinePreference: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumTimelinePreferenceRequest** | [**AlbumTimelinePreferenceRequest**](AlbumTimelinePreferenceRequest.md)|  | 

### Return type

[**AlbumTimelinePreferenceResponse**](AlbumTimelinePreferenceResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateAlbum**
> AlbumSummary updateAlbum(albumId, albumPatch)

Patch one album's properties.

The owner's partial edit of the album's properties: `title`, `coverAssetId` and `allowAppDeletion`, any subset, at least one field (a body that carries none -- JSON null counts as absent, the patch discipline of the asset surface -- is a client bug and answers 400). The response is the album's wall card after the edit.  Authorization is the two-tier answer every album write pins: an album the caller cannot see at all -- an unknown id, another account's unshared album, every sync album -- answers the single-shape 404 `ALBUM_NOT_FOUND`, byte-identical to the read face, so album ids cannot be probed; an album the caller CAN see but does not own (an album shared to the caller) is an honest authorization failure -- 403 `AUTH_FORBIDDEN`, the share is view-only. The owner writes, and an administrator may override the owner check (the admin role bypasses it, the permanent delete's own discipline).  Cover validation (the application-layer check): a present `coverAssetId` must be a LIVE member of THIS album -- the membership row exists and the asset passes the live predicate (not trashed, not hidden, not a Live Photo component row). Any other id -- an asset that is no member, a trashed, hidden or component row, another account's asset -- answers 400 `VALIDATION_FAILED` carrying the message key `album_cover_not_member`. A `coverAssetId` that is absent (or JSON null, which decodes as absent) leaves the pin unchanged.  The deletability switch (`allowAppDeletion`) is the directory album's own field: a patch that carries it on a memory album answers 400 `VALIDATION_FAILED` with the message key `album_deletable_not_directory` (a memory album's deletions never move files, so the switch has no home there). The write is idempotent -- repeating the stored value converges.  The write publishes one `album.updated` event on every successful patch, whatever fields changed -- the convergent refetch makes the unconditional fire harmless. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumPatch albumPatch = {"title":"Road trip 2026 -- rerun"}; // AlbumPatch | 

try {
    final response = api.updateAlbum(albumId, albumPatch);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->updateAlbum: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumPatch** | [**AlbumPatch**](AlbumPatch.md)|  | 

### Return type

[**AlbumSummary**](AlbumSummary.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateAlbumAudience**
> AlbumAudienceResponse updateAlbumAudience(albumId, albumAudienceRequest)

Set one album's audience (the full grantee set).

The album's audience as ONE declarative write: the request carries the complete set of accounts that may view the album, and the server converges the `album_shares` rows onto exactly that set -- grants land for the ids that are not audience yet, rows leave for the grantees the set no longer names. \"Who can see\" is an album property; this face is its whole write surface (the one-row grant/revoke pair retired with it -- one concept, one endpoint).  The write is IDEMPOTENT: the same set twice answers the same state, the second request lands no row and fires no event (`granted` and `revoked` both empty). An empty (or all-skipped) set is the honest \"only me\": every row leaves -- and the owner's own access is untouched by construction, because the owner's visibility is the ownership row, never a share row: the owner is not removable through this face whatever the set says.  Skipped entries -- the requested ids that cannot become audience -- are reported honestly, never errors: an unknown or disabled account (`invalid_account` -- one reason for both, so user ids cannot be probed through this face) and the album's own owner (`owner` -- the owner sees the album by construction; a row for them would be noise). Everything else the request names lands or stays.  Authorization is the writes' two-tier answer: an album the caller cannot see at all -- an unknown id, another account's unshared album, every sync album (the caller's own included: the sync stream's face is the timeline, and its audience edit folds into the same single-shape 404) -- answers the reads' byte-identical `ALBUM_NOT_FOUND`, so album ids cannot be probed; an album the caller CAN see but does not own (an album shared to the caller) answers the honest 403 `AUTH_FORBIDDEN` -- the share is view-only, a sharee cannot edit the ACL. The owner writes, and an administrator may override the owner check. The write's own authorization is the whole gate: no album state gates or filters this face.  The write publishes one `album.shared` event per row that landed and one `album.unshared` event per row that left -- the grant/revoke events' discipline carried over: the payload carries the album's id and the GRANTEE's user id (the account whose view of the album changed), and the idempotent no-op stays silent. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumAudienceRequest albumAudienceRequest = {"audience":["0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70","0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f71"]}; // AlbumAudienceRequest | 

try {
    final response = api.updateAlbumAudience(albumId, albumAudienceRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->updateAlbumAudience: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumAudienceRequest** | [**AlbumAudienceRequest**](AlbumAudienceRequest.md)|  | 

### Return type

[**AlbumAudienceResponse**](AlbumAudienceResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

