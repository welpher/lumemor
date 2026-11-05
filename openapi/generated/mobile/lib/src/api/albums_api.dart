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
import 'package:lumemor_api/src/model/album_summary.dart';
import 'package:lumemor_api/src/model/album_wall_response.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/remove_album_items_response.dart';

class AlbumsApi {

  final Dio _dio;

  final Serializers _serializers;

  const AlbumsApi(this._dio, this._serializers);

  /// Add assets to one virtual album.
  /// Adds assets to the caller&#39;s virtual album with the batch surface&#39;s set semantics: one request, one list of at most 200 asset ids, one honest report &#x60;{added, skipped}&#x60;.  VIRTUAL albums only: directory albums manage their own membership (the rescan flow, a later milestone) and sync albums are no album resources; both answer the single-shape 404 like an unknown id.  Skip semantics (the batch surface&#39;s missing-ids discipline): every id must belong to a live asset the CALLER owns -- the owner-based live predicate (not trashed, not hidden, not a Live Photo component row) until the union switchover of a later task. A foreign account&#39;s asset, a trashed, hidden or component row, and an unknown id are silently skipped, never errors. An id that already sits in the album is skipped too: membership is a set, the add is idempotent. An all-skipped request still answers 200 with &#x60;added&#x60; 0. Repeated ids collapse to one application; the counts cover the distinct ids. The inserted rows record the caller as &#x60;added_by&#x60;.  The locked gate applies: a locked album refuses the write with 409 &#x60;ALBUM_LOCKED&#x60;. Authorization is the writes&#39; two-tier answer (invisible 404, shared-to-caller 403, the owner or an admin override writes).  The write publishes one &#x60;album.items-changed&#x60; event exactly when at least one row was added -- the idempotent no-op stays silent. 
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
  /// Deletes the caller&#39;s virtual album: the row goes, and every &#x60;album_items&#x60; membership row and every &#x60;album_shares&#x60; row follows through the schema&#39;s cascades. The answer is 204 with no body, and the write publishes one &#x60;album.deleted&#x60; event.  VIRTUAL albums only: a directory album answers the single-shape 404 (directory removal is the directory album&#39;s own flow, not a generic delete -- it arrives with a later milestone), and a sync album is no album resource at all. The same 404 answers an unknown id.  The locked gate does NOT apply to the delete: a locked album can be deleted by its owner -- locking is a curation freeze, not a tombstone.  Authorization is the writes&#39; two-tier answer: an album the caller cannot see answers the single-shape 404; an album shared to the caller answers 403 &#x60;AUTH_FORBIDDEN&#x60; (the share is view-only); the owner -- or an administrator overriding the owner check -- deletes. 
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
  /// The detail view of one album the caller can see: the caller owns it, or it is shared to the caller through an &#x60;album_shares&#x60; row. &#x60;callerRole&#x60; says which -- &#x60;owner&#x60; for the owner&#39;s write affordances, &#x60;viewer&#x60; for the read-only share.  Sync albums are not album resources: the id of a sync album -- the caller&#39;s own or anyone else&#39;s -- answers the same 404 as an unknown id. The sync album is the timeline (&#x60;GET /assets/timeline&#x60;); its stream surfaces there, never through this face.  404 is single-shape by design: an unknown id, another account&#39;s unshared album, and a sync album all answer the same &#x60;ALBUM_NOT_FOUND&#x60; byte for byte -- the anti-enumeration discipline of &#x60;ASSET_NOT_FOUND&#x60; applied to the album face. An album whose share to the caller was revoked before the request simply sits in that same invisible region.  &#x60;itemCount&#x60; counts the album&#39;s live members; &#x60;items&#x60; is one keyset page of them, ordered &#x60;takenAt&#x60; descending with the asset id as the tiebreak -- the timeline&#39;s own ordering and cursor pair, so the album grid pages with the timeline&#39;s walking machinery. Every item carries the membership&#39;s &#x60;addedAt&#x60; and the minimal asset context the grid renders; &#x60;thumbnail&#x60; is the asset thumbnail endpoint&#39;s &#x60;tile&#x60;-size URL. Live members only: trashed, hidden and Live Photo component rows never appear (A5 -- a hidden asset is invisible to the whole family).  &#x60;coverAssetId&#x60; is the raw pinned column: the single asset the owner chose, or null when unset -- the wall derives its collage fallback from the members, the detail carries the pin itself.  &#x60;limit&#x60; bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 &#x60;VALIDATION_FAILED&#x60;. Pages continue through the opaque &#x60;cursor&#x60;, the timeline&#39;s exact keyset semantics: &#x60;nextCursor&#x60; is present exactly when another page follows, and the last page carries none. 
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

  /// The caller&#39;s album wall.
  /// The album wall of the calling user: every album that deserves a wall card, in two groups concatenated into one array -- the web client renders its sections off the &#x60;callerRole&#x60; discriminator each item carries.  The own group comes first: the albums the caller owns of type &#x60;virtual&#x60; or &#x60;directory&#x60;, the most recently updated first (the album id descending as the tiebreak). The shared group follows: every album shared to the caller through an &#x60;album_shares&#x60; row, the caller&#39;s most recently granted share first (the album id descending as the tiebreak). Shared items carry &#x60;owner&#x60; -- the honest attribution of another account&#39;s album: the owner&#39;s id and email.  Sync albums NEVER appear on the wall, in either group. The sync album is the account&#39;s core backup stream and lives as the timeline (&#x60;GET /assets/timeline&#x60;), not as an album resource; a sync album another account shared is likewise no wall card. The caller&#39;s own sync album is invisible here and answers 404 through &#x60;GET /albums/{albumId}&#x60;.  &#x60;coverAssetIds&#x60; is the wall card&#39;s collage source: the pinned &#x60;coverAssetId&#x60; as a single-element list when the owner set one, otherwise the album&#39;s most recent live member assets -- up to four, &#x60;takenAt&#x60; descending with the asset id as the tiebreak, fewer when the album holds fewer members, empty for an album with none. &#x60;itemCount&#x60; counts live members under the same predicate: trashed, hidden and Live Photo component rows never count (A5 -- a hidden asset is invisible to the whole family).  &#x60;isShared&#x60; reports whether the album carries at least one &#x60;album_shares&#x60; row -- the owner&#39;s \&quot;someone else sees this\&quot; badge. It is true by construction on every shared-group item (the caller&#39;s share row is the reason the item exists). 
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
