//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:lumemor_api/src/model/error.dart';

class EventsApi {

  final Dio _dio;

  final Serializers _serializers;

  const EventsApi(this._dio, this._serializers);

  /// Stream instance activity over server-sent events.
  /// The activity stream of the instance, authenticated like the REST API: the web session cookie and the mobile bearer token both open the stream.  Every event is one &#x60;text/event-stream&#x60; block: &#x60;event:&#x60; carries the event name, &#x60;id:&#x60; a monotonically increasing number, and &#x60;data:&#x60; the JSON payload -- the resource id plus a summary of the changed fields. Event names follow &#x60;&lt;resource&gt;.&lt;past-tense verb&gt;&#x60;:  - &#x60;asset.uploaded&#x60;, &#x60;asset.ready&#x60;, &#x60;asset.added&#x60;,   &#x60;asset.updated&#x60;, &#x60;asset.trashed&#x60;, &#x60;asset.restored&#x60;,   &#x60;asset.deleted&#x60; - &#x60;album.created&#x60;, &#x60;album.updated&#x60;, &#x60;album.deleted&#x60;,   &#x60;album.items-changed&#x60;, &#x60;album.shared&#x60;, &#x60;album.unshared&#x60; - &#x60;auth.login_failed&#x60; - &#x60;face.clustered&#x60; - &#x60;job.started&#x60;, &#x60;job.progress&#x60;, &#x60;job.finished&#x60; - &#x60;pairing.token_created&#x60; - &#x60;person.labeled&#x60;, &#x60;person.unlabeled&#x60;, &#x60;person.hidden&#x60;,   &#x60;person.merged&#x60;, &#x60;person.face_moved&#x60;,   &#x60;person.face_clustered&#x60;, &#x60;person.face_detached&#x60; - &#x60;session.started&#x60;, &#x60;session.revoked&#x60; - &#x60;server.settings-changed&#x60;  The &#x60;person.*&#x60; events report the people surface&#39;s label, hidden, merge and face-write writes. Every payload carries ONLY the acting user&#39;s &#x60;userId&#x60; beside the cluster (or face) id: the label text and the hidden switch are one account&#39;s private state and never ride the shared stream. A subscriber filters by the ids it renders, never by another account&#39;s change. &#x60;person.merged&#x60; carries the dissolved source cluster ids comma-joined as &#x60;mergedClusterIds&#x60;, &#x60;person.face_moved&#x60; carries the face&#39;s new &#x60;clusterId&#x60;, &#x60;person.face_clustered&#x60; the newly created cluster&#39;s id, and &#x60;person.face_detached&#x60; the cluster the face left.  The &#x60;album.*&#x60; events report the album write surface: the directory album&#39;s and the memory&#39;s creation (&#x60;album.created&#x60;), the property patch (&#x60;album.updated&#x60;), the delete (&#x60;album.deleted&#x60;), the membership writes (&#x60;album.items-changed&#x60;) and the audience writes (&#x60;album.shared&#x60;, &#x60;album.unshared&#x60;). Every payload carries ONLY the album&#39;s id and one user&#39;s &#x60;userId&#x60;: titles, covers, the deletability switch, the source path and the membership itself are album state the subscribers refetch, never bus payload. &#x60;album.updated&#x60; fires on every successful patch (the convergent refetch makes the unconditional fire harmless), &#x60;album.items-changed&#x60; exactly when a membership write changed at least one row (the memory&#39;s curation arms, or the rescan&#39;s add arm growing the set), and &#x60;album.shared&#x60;/&#x60;album.unshared&#x60; exactly when the audience write (or a creation&#39;s initial audience) landed or removed a row -- one event per row transition, every idempotent no-op stays silent. On the two audience events the &#x60;userId&#x60; is the GRANTEE whose view of the album changed, not the acting owner: the owner is derivable server-side, the grantee is the state that changed, and the grant time and the grantee&#39;s email never ride the bus.  The timeline-preference writes publish nothing at all: the per-user preference is self-affecting state -- the caller&#39;s own timeline is the only view that changes, and their own response carries the new value, so no other session has anything to refetch.  The &#x60;asset.added&#x60; event reports the directory rescan&#39;s add arm: one event per file the scan ingested by reference (the row, its membership and its pipeline jobs landed), the same id-plus-kind payload every asset event carries. The rescan&#39;s other two arms reuse the family&#39;s existing transitions: the vanish arm fires &#x60;asset.trashed&#x60; (the row&#39;s state transition -- though the orphan never appears in the trash view, the file being the source&#39;s own), and the change arm fires &#x60;asset.updated&#x60; (the snapshot changed and the metadata pipeline re-runs). A rescan that changed nothing publishes none of them. The membership the add arm grew announces through &#x60;album.items-changed&#x60; exactly when at least one membership row landed -- the same predicate the removal arm fires under, widened to both directions of the membership write.  The server keeps the last five minutes of events in a ring buffer. Browser EventSource clients reconnect automatically and resend &#x60;Last-Event-ID&#x60;; the server replays everything newer than that id, and events that fell out of the ring are skipped. 
  ///
  /// Parameters:
  /// * [lastEventID] - The id of the last received event, resent automatically by browser EventSource implementations on reconnect. When present, the server replays the events newer than this id from its five-minute ring buffer; events that fell out of the ring are skipped. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [String] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<String>> getEvents({ 
    String? lastEventID,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/events';
    final _options = Options(
      method: r'GET',
      headers: <String, dynamic>{
        if (lastEventID != null) r'Last-Event-ID': lastEventID,
        ...?headers,
      },
      extra: <String, dynamic>{
        'secure': <Map<String, String>>[
          {
            'type': 'apiKey',
            'name': 'cookieAuth',
            'keyName': 'lumemor_session',
            'where': '',
          },{
            'type': 'http',
            'scheme': 'bearer',
            'name': 'bearerAuth',
          },
        ],
        ...?extra,
      },
      validateStatus: validateStatus,
    );

    final _response = await _dio.request<Object>(
      _path,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    String? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : rawResponse as String;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<String>(
      data: _responseData,
      headers: _response.headers,
      isRedirect: _response.isRedirect,
      requestOptions: _response.requestOptions,
      redirects: _response.redirects,
      statusCode: _response.statusCode,
      statusMessage: _response.statusMessage,
      extra: _response.extra,
    );
  }

}
