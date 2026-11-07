//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/add_album_items_response.dart';
import 'package:lumemor_api/src/model/album_create_request.dart';
import 'package:lumemor_api/src/model/album_detail.dart';
import 'package:lumemor_api/src/model/album_items_request.dart';
import 'package:lumemor_api/src/model/album_patch.dart';
import 'package:lumemor_api/src/model/album_share_grant_request.dart';
import 'package:lumemor_api/src/model/album_share_response.dart';
import 'package:lumemor_api/src/model/album_summary.dart';
import 'package:lumemor_api/src/model/album_timeline_preference_request.dart';
import 'package:lumemor_api/src/model/album_timeline_preference_response.dart';
import 'package:lumemor_api/src/model/album_wall_response.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/remove_album_items_response.dart';

class AlbumsApi {

  final Dio _dio;

  final Serializers _serializers;

  const AlbumsApi(this._dio, this._serializers);

  /// Add assets to one virtual album.
  /// Adds assets to the caller&#39;s virtual album with the batch surface&#39;s set semantics: one request, one list of at most 200 asset ids, one honest report &#x60;{added, skipped}&#x60;.  VIRTUAL albums only: directory albums manage their own membership (the rescan flow, a later milestone) and sync albums are no album resources; both answer the single-shape 404 like an unknown id.  Skip semantics (the batch surface&#39;s missing-ids discipline): every id must belong to a live asset the CALLER owns -- the owner-based live predicate (not trashed, not hidden, not a Live Photo component row). Curating another account&#39;s shared photos into a virtual album is the re-share feature and arrives with its own milestone: the union deliberately does not carry curation yet. A foreign account&#39;s asset, a trashed, hidden or component row, and an unknown id are silently skipped, never errors. An id that already sits in the album is skipped too: membership is a set, the add is idempotent. An all-skipped request still answers 200 with &#x60;added&#x60; 0. Repeated ids collapse to one application; the counts cover the distinct ids. The inserted rows record the caller as &#x60;added_by&#x60;.  The locked gate applies: a locked album refuses the write with 409 &#x60;ALBUM_LOCKED&#x60;. Authorization is the writes&#39; two-tier answer (invisible 404, shared-to-caller 403, the owner or an admin override writes).  The write publishes one &#x60;album.items-changed&#x60; event exactly when at least one row was added -- the idempotent no-op stays silent. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [albumItemsRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AddAlbumItemsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AddAlbumItemsResponse>> addAlbumItems({ 
    required String albumId,
    required AlbumItemsRequest albumItemsRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}/items'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
    final _options = Options(
      method: r'POST',
      headers: <String, dynamic>{
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(AlbumItemsRequest);
      _bodyData = _serializers.serialize(albumItemsRequest, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    AddAlbumItemsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AddAlbumItemsResponse),
      ) as AddAlbumItemsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AddAlbumItemsResponse>(
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

  /// Create one virtual album.
  /// Creates one virtual album owned by the caller -- the album the wall lists, the detail shows and the curation endpoints edit. The virtual album is the ONLY creatable type through this face, so the request carries no type field: directory albums arrive with their own milestone (they mirror a filesystem path the server scans), and every account&#39;s sync album is born with the account.  The title is trimmed and must span 1-64 Unicode characters (the count is over runes, so multi-byte titles stay within the bound at the same length); the description is optional bounded free text (10,000 characters, the asset description&#39;s own bound) and defaults to the empty string.  The response is the created album&#39;s wall card -- the summary exactly as &#x60;GET /albums&#x60; lists it: &#x60;callerRole&#x60; &#x60;owner&#x60;, &#x60;isShared&#x60; false, an empty &#x60;coverAssetIds&#x60; array (never null) and &#x60;itemCount&#x60; 0, nothing being a member yet. The write publishes one &#x60;album.created&#x60; event. 
  ///
  /// Parameters:
  /// * [albumCreateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumSummary] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumSummary>> createAlbum({ 
    required AlbumCreateRequest albumCreateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums';
    final _options = Options(
      method: r'POST',
      headers: <String, dynamic>{
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(AlbumCreateRequest);
      _bodyData = _serializers.serialize(albumCreateRequest, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    AlbumSummary? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumSummary),
      ) as AlbumSummary;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumSummary>(
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

  /// Delete one virtual album.
  /// Deletes the caller&#39;s virtual album: the row goes, and every &#x60;album_items&#x60; membership row and every &#x60;album_shares&#x60; row follows through the schema&#39;s cascades. The answer is 204 with no body, and the write publishes one &#x60;album.deleted&#x60; event.  VIRTUAL albums only: a directory album answers the single-shape 404 (directory removal is the directory album&#39;s own flow, not a generic delete -- it arrives with a later milestone), and a sync album is no album resource at all. The same 404 answers an unknown id.  The locked gate applies to the delete like every other write: the permission matrix pins deletion to the UNLOCKED state, so a locked album answers 409 &#x60;ALBUM_LOCKED&#x60; (the owner unlocks first -- one step, the freeze is a state to clear, not a wall to climb).  Authorization is the writes&#39; two-tier answer: an album the caller cannot see answers the single-shape 404; an album shared to the caller answers 403 &#x60;AUTH_FORBIDDEN&#x60; (the share is view-only); the owner -- or an administrator overriding the owner check -- deletes. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> deleteAlbum({ 
    required String albumId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
    final _options = Options(
      method: r'DELETE',
      headers: <String, dynamic>{
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

    return _response;
  }

  /// One album&#39;s detail and one page of its items.
  /// The detail view of one album the caller can see: the caller owns it, or it is shared to the caller through an &#x60;album_shares&#x60; row. &#x60;callerRole&#x60; says which -- &#x60;owner&#x60; for the owner&#39;s write affordances, &#x60;viewer&#x60; for the read-only share -- with the one exception below.  The caller&#39;s OWN sync album answers 200: the default album&#39;s read-only detail. &#x60;callerRole&#x60; is &#x60;owner&#x60;, the live members page through the same keyset as any album&#39;s (the grid and the viewer work like any album&#39;s), but the album is system-managed: NO write affordance applies to it -- the property patch, the membership curation, the sharing face, the timeline preference and the delete all keep refusing a sync album with their unchanged 404/400 answers, whatever this read admits. &#x60;inTimeline&#x60; is absent (the core stream is never excludable) and the owner&#39;s &#x60;shares&#x60; list rides the response always empty (no share row can name a sync album). The client renders the localized display name (en \&quot;My photos\&quot;, zh-Hans \&quot;我的照片\&quot;) keyed by &#x60;type: &#39;sync&#39;&#x60; -- the stored title is the internal system label and never renders.  A sync album of ANOTHER account answers the same 404 as an unknown id, whatever share rows exist: invisible by design.  404 is single-shape by design: an unknown id, another account&#39;s unshared album, and another account&#39;s sync album all answer the same &#x60;ALBUM_NOT_FOUND&#x60; byte for byte -- the anti-enumeration discipline of &#x60;ASSET_NOT_FOUND&#x60; applied to the album face. An album whose share to the caller was revoked before the request simply sits in that same invisible region.  &#x60;itemCount&#x60; counts the album&#39;s live members; &#x60;items&#x60; is one keyset page of them, ordered &#x60;takenAt&#x60; descending with the asset id as the tiebreak -- the timeline&#39;s own ordering and cursor pair, so the album grid pages with the timeline&#39;s walking machinery. Every item carries the membership&#39;s &#x60;addedAt&#x60; and the minimal asset context the grid renders; &#x60;thumbnail&#x60; is the asset thumbnail endpoint&#39;s &#x60;tile&#x60;-size URL. Live members only: trashed, hidden and Live Photo component rows never appear (A5 -- a hidden asset is invisible to the whole family).  &#x60;coverAssetId&#x60; is the raw pinned column: the single asset the owner chose, or null when unset -- the wall derives its collage fallback from the members, the detail carries the pin itself.  &#x60;shares&#x60; rides only the owner&#39;s view: the album&#39;s grantee list, one entry per &#x60;album_shares&#x60; row in the grant order (&#x60;grantedAt&#x60; ascending), each carrying the grantee&#39;s id, email and grant time -- the share management dialog&#39;s list. The owner&#39;s response always carries the field, an empty array when the album holds no grants; viewer responses never carry it at all (a sharee sees the album, never the ACL).  &#x60;inTimeline&#x60; is the caller&#39;s timeline preference of this album (the exclusion toggle&#39;s source of truth): present on the excludable types -- the caller&#39;s own &#x60;directory&#x60; albums and the albums shared to them -- and absent on the caller&#39;s own &#x60;virtual&#x60; albums. The preference scopes the aggregate: it never filters this album&#39;s own items, so browsing a switched-off album still shows everything.  &#x60;limit&#x60; bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 &#x60;VALIDATION_FAILED&#x60;. Pages continue through the opaque &#x60;cursor&#x60;, the timeline&#39;s exact keyset semantics: &#x60;nextCursor&#x60; is present exactly when another page follows, and the last page carries none. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [cursor] - The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
  /// * [limit] - Page size: the number of album items one response carries at most. Defaults to 60; values below 1 or over 200 answer 400. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumDetail] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumDetail>> getAlbum({ 
    required String albumId,
    String? cursor,
    int? limit = 60,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
    final _options = Options(
      method: r'GET',
      headers: <String, dynamic>{
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

    final _queryParameters = <String, dynamic>{
      if (cursor != null) r'cursor': encodeQueryParameter(_serializers, cursor, const FullType(String)),
      if (limit != null) r'limit': encodeQueryParameter(_serializers, limit, const FullType(int)),
    };

    final _response = await _dio.request<Object>(
      _path,
      options: _options,
      queryParameters: _queryParameters,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    AlbumDetail? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumDetail),
      ) as AlbumDetail;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumDetail>(
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

  /// Grant one account the view ACL of an album.
  /// Grants an album&#39;s view ACL to another account: one &#x60;album_shares&#x60; row, one more viewer. The grantee&#39;s read access is immediate -- their wall lists the album in the shared group, the detail answers with &#x60;callerRole&#x60; &#x60;viewer&#x60; -- and the share is view-only forever: every write face of the album stays the owner&#39;s.  Authorization is the writes&#39; two-tier answer: an album the caller cannot see at all -- an unknown id, another account&#39;s unshared album, every sync album -- answers the single-shape 404 &#x60;ALBUM_NOT_FOUND&#x60;, byte-identical to the read face, so album ids cannot be probed; an album the caller CAN see but does not own (an album shared to the caller) answers the honest 403 &#x60;AUTH_FORBIDDEN&#x60; -- a sharee cannot share onward. The owner grants, and an administrator may override the owner check.  VIRTUAL and DIRECTORY albums are shareable; sync albums are never -- a sync album id folds into the same single-shape 404 (the timeline is the sync album&#39;s face). &#x60;locked&#x60; does NOT gate the grant: the freeze is a curation gate, and a viewer only ever reads.  The grant is idempotent: re-granting to a current grantee answers 200 with the existing row -- the same &#x60;grantedAt&#x60;, the original grantor untouched -- and stays silent on the activity stream.  Target rules (both answer 400 &#x60;VALIDATION_FAILED&#x60; with a message key in &#x60;details.key&#x60;): the album&#39;s own owner is never a grantee (&#x60;album_share_to_owner&#x60; -- the owner sees the album by construction, a share row would be noise); an unknown or disabled account is not a share target (&#x60;album_share_user_invalid&#x60; -- the two cases answer the same shape, so user ids cannot be probed through this face).  The write publishes one &#x60;album.shared&#x60; event exactly when the grant landed a new row: the payload carries the album&#39;s id and the GRANTEE&#39;s user id, nothing else. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [albumShareGrantRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumShareResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumShareResponse>> grantAlbumShare({ 
    required String albumId,
    required AlbumShareGrantRequest albumShareGrantRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}/shares'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
    final _options = Options(
      method: r'POST',
      headers: <String, dynamic>{
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(AlbumShareGrantRequest);
      _bodyData = _serializers.serialize(albumShareGrantRequest, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    AlbumShareResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumShareResponse),
      ) as AlbumShareResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumShareResponse>(
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

  /// The caller&#39;s album wall.
  /// The album wall of the calling user: every album that deserves a wall card, in two groups concatenated into one array -- the web client renders its sections off the &#x60;callerRole&#x60; discriminator each item carries.  The caller&#39;s sync album is pinned FIRST, before the own group: the one card of type &#x60;sync&#x60;, the caller&#39;s own core upload stream surfaced as the default \&quot;My photos\&quot; album -- every upload lands in it, so a fresh account&#39;s wall is never empty. The own group follows: the albums the caller owns of type &#x60;virtual&#x60; or &#x60;directory&#x60;, the most recently updated first (the album id descending as the tiebreak). The shared group follows last: every album shared to the caller through an &#x60;album_shares&#x60; row, the caller&#39;s most recently granted share first (the album id descending as the tiebreak). Shared items carry &#x60;owner&#x60; -- the honest attribution of another account&#39;s album: the owner&#39;s id and email.  A sync album of ANOTHER account never appears, in either group: the shared arm carries &#x60;virtual&#x60; and &#x60;directory&#x60; albums only, whatever share rows exist. The pinned card&#39;s &#x60;title&#x60; is the system&#39;s internal fixed label (&#39;Sync&#39;) -- the display name is the CLIENT&#39;s: &#x60;type: &#39;sync&#39;&#x60; keys the localized default-album name (en \&quot;My photos\&quot;, zh-Hans \&quot;我的照片\&quot;), and the stored title never renders.  &#x60;coverAssetIds&#x60; is the wall card&#39;s collage source: the pinned &#x60;coverAssetId&#x60; as a single-element list when the owner set one, otherwise the album&#39;s most recent live member assets -- up to four, &#x60;takenAt&#x60; descending with the asset id as the tiebreak, fewer when the album holds fewer members, empty for an album with none. &#x60;itemCount&#x60; counts live members under the same predicate: trashed, hidden and Live Photo component rows never count (A5 -- a hidden asset is invisible to the whole family).  &#x60;isShared&#x60; reports whether the album carries at least one &#x60;album_shares&#x60; row -- the owner&#39;s \&quot;someone else sees this\&quot; badge. It is true by construction on every shared-group item (the caller&#39;s share row is the reason the item exists).  &#x60;inTimeline&#x60; rides the excludable cards -- the own &#x60;directory&#x60; items and every shared item: the caller&#39;s timeline preference of that album, true when no preference row exists (the default of feeding the timeline). The own &#x60;virtual&#x60; items omit the field: they never feed the caller&#39;s own timeline. The pinned &#x60;sync&#x60; card omits it too: the core stream is never excludable (the preference face answers 400 for it). The field is the toggle&#39;s source of truth for the wall&#39;s \&quot;excluded\&quot; badge, kept current by the &#x60;PUT/DELETE /albums/{albumId}/timeline-preference&#x60; endpoints. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumWallResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumWallResponse>> listAlbums({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums';
    final _options = Options(
      method: r'GET',
      headers: <String, dynamic>{
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

    AlbumWallResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumWallResponse),
      ) as AlbumWallResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumWallResponse>(
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

  /// Remove assets from one virtual album.
  /// Removes membership rows from the caller&#39;s virtual album: &#x60;{removed, skipped}&#x60;, the same batch semantics as the add. An id that is no member of the album -- never was, or already removed -- is skipped; an all-skipped request still answers 200 with &#x60;removed&#x60; 0. Repeated ids collapse; the counts cover the distinct ids.  The removal carries no live predicate: a trashed member&#39;s row still leaves, the honest cleanup of the set (the live views filter trashed members out of the renders; removing the row tidies the set itself).  VIRTUAL albums only (directory and sync albums answer the single-shape 404); the locked gate applies (409 &#x60;ALBUM_LOCKED&#x60;); authorization is the writes&#39; two-tier answer (invisible 404, shared-to-caller 403, the owner or an admin override writes).  The write publishes one &#x60;album.items-changed&#x60; event exactly when at least one row was removed -- the idempotent no-op stays silent. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [albumItemsRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [RemoveAlbumItemsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<RemoveAlbumItemsResponse>> removeAlbumItems({ 
    required String albumId,
    required AlbumItemsRequest albumItemsRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}/items'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
    final _options = Options(
      method: r'DELETE',
      headers: <String, dynamic>{
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(AlbumItemsRequest);
      _bodyData = _serializers.serialize(albumItemsRequest, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    RemoveAlbumItemsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(RemoveAlbumItemsResponse),
      ) as RemoveAlbumItemsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<RemoveAlbumItemsResponse>(
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

  /// Reset the caller&#39;s timeline preference of one album.
  /// Removes the calling user&#39;s preference row of one album: the album returns to the default state of feeding the caller&#39;s timeline (a missing row means kept in). The eligibility scope is the PUT&#39;s own -- the caller&#39;s own directory albums and the albums shared to the caller; the same 400 &#x60;album_pref_not_excludable&#x60; answers the caller&#39;s own sync and own virtual albums, and the same single-shape 404 answers an album the caller cannot see.  The delete is idempotent: an album with no preference row (never set, or already reset) still answers 204 -- the row is gone either way, which is the state the caller asked for. Like the PUT it publishes no event: the preference is the caller&#39;s own state, and no other session has anything to refetch. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> resetAlbumTimelinePreference({ 
    required String albumId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}/timeline-preference'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
    final _options = Options(
      method: r'DELETE',
      headers: <String, dynamic>{
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

    return _response;
  }

  /// Revoke one account&#39;s view ACL of an album.
  /// Removes one &#x60;album_shares&#x60; row: the grantee&#39;s read access ends immediately -- the album leaves their wall, the detail answers the single-shape 404.  The owner (or an administrator) revokes any grantee. A grantee may revoke their OWN row -- leaving a shared album is the sharee&#39;s own right, the family semantics -- through this same endpoint: the caller&#39;s own user id in the path is the departure, whoever granted the row. A sharee cannot revoke ANOTHER account&#39;s grant: 403 &#x60;AUTH_FORBIDDEN&#x60;, the honest refusal of a caller who can see the album but holds no power over its ACL.  Everything else mirrors the grant: the single-shape 404 for the invisible ids (an unknown album, another account&#39;s unshared album, every sync album), and &#x60;locked&#x60; never gates -- revocation is not curation.  The 204 is idempotent: revoking a share that is already gone -- never granted, already revoked, or an id that names no account -- is the state the caller asked for, so it answers 204 again and stays silent on the activity stream.  The write publishes one &#x60;album.unshared&#x60; event exactly when a row left: the payload carries the album&#39;s id and the revoked grantee&#39;s user id, nothing else. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [userId] - Identifier of the account whose share row leaves: any grantee for the owner or an administrator, the caller's own id for a sharee's departure. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> revokeAlbumShare({ 
    required String albumId,
    required String userId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}/shares/{userId}'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString()).replaceAll('{' r'userId' '}', encodeQueryParameter(_serializers, userId, const FullType(String)).toString());
    final _options = Options(
      method: r'DELETE',
      headers: <String, dynamic>{
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

    return _response;
  }

  /// Set the caller&#39;s timeline preference of one album.
  /// Upserts the calling user&#39;s per-user timeline preference of one album: the &#x60;user_album_prefs&#x60; row keyed (userId, albumId) whose &#x60;inTimeline&#x60; switch says whether the album feeds the caller&#39;s aggregated timeline. The preference is the CALLER&#39;s own state over an album they can see -- self-affecting only, which is why these writes publish no event: the caller&#39;s own response is the only view that changed, and no other session has anything to refetch. The scope is the albums whose membership a caller could want out of their timeline feed: the caller&#39;s own &#x60;directory&#x60; albums, and every album shared to the caller (whatever the owner&#39;s type). Two kinds of album answer 400 instead, with the &#x60;album_pref_not_excludable&#x60; message key in the details:  - the caller&#39;s own &#x60;sync&#x60; album: it IS the core stream, and   the core stream is never excludable (the aggregation   formula exempts the sync arm from the preference row --   a stray row there excludes nothing); - the caller&#39;s own &#x60;virtual&#x60; albums: they never feed the   caller&#39;s own timeline at all (in-stream curation would be   pure duplication), so there is nothing to exclude.  An album the caller cannot see -- unknown, another account&#39;s unshared album, a revoked share -- answers the same byte-identical 404 &#x60;ALBUM_NOT_FOUND&#x60; as every album face, so album ids cannot be probed for existence through this endpoint. There is no administrator override: the preference is always the caller&#39;s own row over their own view, never a write onto another account&#39;s state.  &#x60;inTimeline&#x60; is required and must be a JSON boolean: a body that is not valid JSON, or one without the field, answers 400 &#x60;VALIDATION_FAILED&#x60;. The write is an upsert: repeating it with the same value converges to the same row, and repeating it with the other value flips the switch -- the response always carries the stored state. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [albumTimelinePreferenceRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumTimelinePreferenceResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumTimelinePreferenceResponse>> setAlbumTimelinePreference({ 
    required String albumId,
    required AlbumTimelinePreferenceRequest albumTimelinePreferenceRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}/timeline-preference'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
    final _options = Options(
      method: r'PUT',
      headers: <String, dynamic>{
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(AlbumTimelinePreferenceRequest);
      _bodyData = _serializers.serialize(albumTimelinePreferenceRequest, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    AlbumTimelinePreferenceResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumTimelinePreferenceResponse),
      ) as AlbumTimelinePreferenceResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumTimelinePreferenceResponse>(
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

  /// Patch one album&#39;s properties.
  /// The owner&#39;s partial edit of the album&#39;s properties: &#x60;title&#x60;, &#x60;description&#x60;, &#x60;coverAssetId&#x60; and &#x60;locked&#x60;, any subset, at least one field (a body that carries none -- JSON null counts as absent, the patch discipline of the asset surface -- is a client bug and answers 400). The response is the album&#39;s wall card after the edit.  Authorization is the two-tier answer every album write pins: an album the caller cannot see at all -- an unknown id, another account&#39;s unshared album, every sync album -- answers the single-shape 404 &#x60;ALBUM_NOT_FOUND&#x60;, byte-identical to the read face, so album ids cannot be probed; an album the caller CAN see but does not own (an album shared to the caller) is an honest authorization failure -- 403 &#x60;AUTH_FORBIDDEN&#x60;, the share is view-only. The owner writes, and an administrator may override the owner check (the admin role bypasses it, the permanent delete&#39;s own discipline).  The locked gate: a locked album refuses the property and curation fields (&#x60;title&#x60;, &#x60;description&#x60;, &#x60;coverAssetId&#x60;) with 409 &#x60;ALBUM_LOCKED&#x60; -- unlock first. The &#x60;locked&#x60; field itself is never gated: a patch of only &#x60;locked: false&#x60; IS the unlock.  Cover validation (the spec&#39;s application-layer pin): a present &#x60;coverAssetId&#x60; must be a LIVE member of THIS album -- the membership row exists and the asset passes the live predicate (not trashed, not hidden, not a Live Photo component row). Any other id -- an asset that is no member, a trashed, hidden or component row, another account&#39;s asset -- answers 400 &#x60;VALIDATION_FAILED&#x60; carrying the message key &#x60;album_cover_not_member&#x60;. A &#x60;coverAssetId&#x60; that is absent (or JSON null, which decodes as absent) leaves the pin unchanged.  The write publishes one &#x60;album.updated&#x60; event on every successful patch, whatever fields changed -- the convergent refetch makes the unconditional fire harmless. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [albumPatch] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumSummary] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumSummary>> updateAlbum({ 
    required String albumId,
    required AlbumPatch albumPatch,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
    final _options = Options(
      method: r'PATCH',
      headers: <String, dynamic>{
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
      contentType: 'application/json',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      const _type = FullType(AlbumPatch);
      _bodyData = _serializers.serialize(albumPatch, specifiedType: _type);

    } catch(error, stackTrace) {
      throw DioException(
         requestOptions: _options.compose(
          _dio.options,
          _path,
        ),
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    final _response = await _dio.request<Object>(
      _path,
      data: _bodyData,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    AlbumSummary? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumSummary),
      ) as AlbumSummary;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumSummary>(
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
