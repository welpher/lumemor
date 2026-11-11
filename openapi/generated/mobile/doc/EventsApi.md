# lumemor_api.api.EventsApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getEvents**](EventsApi.md#getevents) | **GET** /events | Stream instance activity over server-sent events.


# **getEvents**
> String getEvents(lastEventID)

Stream instance activity over server-sent events.

The activity stream of the instance, authenticated like the REST API: the web session cookie and the mobile bearer token both open the stream.  Every event is one `text/event-stream` block: `event:` carries the event name, `id:` a monotonically increasing number, and `data:` the JSON payload -- the resource id plus a summary of the changed fields. Event names follow `<resource>.<past-tense verb>`:  - `asset.uploaded`, `asset.ready`, `asset.added`,   `asset.updated`, `asset.trashed`, `asset.restored`,   `asset.deleted` - `album.created`, `album.updated`, `album.deleted`,   `album.items-changed`, `album.shared`, `album.unshared` - `auth.login_failed` - `face.clustered` - `job.started`, `job.progress`, `job.finished` - `pairing.token_created` - `person.labeled`, `person.unlabeled`, `person.hidden`,   `person.merged`, `person.face_moved` - `session.started`, `session.revoked` - `server.settings-changed`  The `person.*` events report the people surface's label, hidden, merge and face-move writes. Every payload carries ONLY the acting user's `userId` beside the cluster (or face) id: the label text and the hidden switch are one account's private state and never ride the shared stream. A subscriber filters by the ids it renders, never by another account's change. `person.merged` carries the dissolved source cluster ids comma-joined as `mergedClusterIds`, and `person.face_moved` carries the face's new `clusterId`.  The `album.*` events report the album write surface: creation (`album.created`), the property patch (`album.updated`), the delete (`album.deleted`), the membership writes (`album.items-changed`) and the audience writes (`album.shared`, `album.unshared`). Every payload carries ONLY the album's id and one user's `userId`: titles, covers, the deletability switch, the source path and the membership itself are album state the subscribers refetch, never bus payload. `album.updated` fires on every successful patch (the convergent refetch makes the unconditional fire harmless), `album.items-changed` exactly when a membership write changed at least one row (the removal arm, or the rescan's add arm growing the set), and `album.shared`/`album.unshared` exactly when the audience write (or a creation's initial audience) landed or removed a row -- one event per row transition, every idempotent no-op stays silent. On the two audience events the `userId` is the GRANTEE whose view of the album changed, not the acting owner: the owner is derivable server-side, the grantee is the state that changed, and the grant time and the grantee's email never ride the bus.  The timeline-preference writes publish nothing at all: the per-user preference is self-affecting state -- the caller's own timeline is the only view that changes, and their own response carries the new value, so no other session has anything to refetch.  The `asset.added` event reports the directory rescan's add arm: one event per file the scan ingested by reference (the row, its membership and its pipeline jobs landed), the same id-plus-kind payload every asset event carries. The rescan's other two arms reuse the family's existing transitions: the vanish arm fires `asset.trashed` (the row's state transition -- though the orphan never appears in the trash view, the file being the source's own), and the change arm fires `asset.updated` (the snapshot changed and the metadata pipeline re-runs). A rescan that changed nothing publishes none of them. The membership the add arm grew announces through `album.items-changed` exactly when at least one membership row landed -- the same predicate the removal arm fires under, widened to both directions of the membership write.  The server keeps the last five minutes of events in a ring buffer. Browser EventSource clients reconnect automatically and resend `Last-Event-ID`; the server replays everything newer than that id, and events that fell out of the ring are skipped. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getEventsApi();
final String lastEventID = lastEventID_example; // String | The id of the last received event, resent automatically by browser EventSource implementations on reconnect. When present, the server replays the events newer than this id from its five-minute ring buffer; events that fell out of the ring are skipped. 

try {
    final response = api.getEvents(lastEventID);
    print(response);
} on DioException catch (e) {
    print('Exception when calling EventsApi->getEvents: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **lastEventID** | **String**| The id of the last received event, resent automatically by browser EventSource implementations on reconnect. When present, the server replays the events newer than this id from its five-minute ring buffer; events that fell out of the ring are skipped.  | [optional] 

### Return type

**String**

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/event-stream, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

