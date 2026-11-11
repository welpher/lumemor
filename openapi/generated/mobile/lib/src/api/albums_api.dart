//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/album_audience_request.dart';
import 'package:lumemor_api/src/model/album_audience_response.dart';
import 'package:lumemor_api/src/model/album_create_request.dart';
import 'package:lumemor_api/src/model/album_create_response.dart';
import 'package:lumemor_api/src/model/album_detail.dart';
import 'package:lumemor_api/src/model/album_items_request.dart';
import 'package:lumemor_api/src/model/album_patch.dart';
import 'package:lumemor_api/src/model/album_path_completions_response.dart';
import 'package:lumemor_api/src/model/album_rescan_response.dart';
import 'package:lumemor_api/src/model/album_summary.dart';
import 'package:lumemor_api/src/model/album_timeline_preference_request.dart';
import 'package:lumemor_api/src/model/album_timeline_preference_response.dart';
import 'package:lumemor_api/src/model/album_wall_response.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/memory_create_request.dart';
import 'package:lumemor_api/src/model/memory_create_response.dart';
import 'package:lumemor_api/src/model/remove_album_items_response.dart';

class AlbumsApi {

  final Dio _dio;

  final Serializers _serializers;

  const AlbumsApi(this._dio, this._serializers);

  /// Complete a directory album&#39;s source path.
  /// The source-path completion of the album creation dialog: the typed prefix in, the pickable directories out. The call is ADMIN-scoped: mounting an existing photo directory is the administrator&#39;s act, so the completion that browses the filesystem on the dialog&#39;s behalf is the administrator&#39;s too -- a signed-in member answers 403 &#x60;AUTH_FORBIDDEN&#x60; before the prefix is even read (the members&#39; contribution path is the upload stream; a member never needs to see the container&#39;s filesystem).  The completion domain is the container&#39;s REAL visible filesystem: whatever the operating system lets the server read, drilled segment by segment. The compose file&#39;s mounts ARE the boundary -- the operator decides what the container sees, and the completion honestly reflects that visibility instead of a software-layer allowlist (the retirement: the mount-root allowlist that used to confine this endpoint died whole, with its settings face and its validation branches). The ONE exclusion that stays is the server&#39;s own managed data tree: no row ever names the application&#39;s originals, derivatives or trash, whatever prefix would match them -- the same confine the creation&#39;s &#x60;ALBUM_PATH_IN_MANAGED_TREE&#x60; branch enforces, kept honest on both sides of the pair.  The matching rule, driven by the typed prefix:  - an empty (or absent) prefix lists the filesystem root   itself (&#x60;/&#x60;) -- the dialog&#39;s first open, \&quot;start here\&quot;; - a prefix of exactly &#x60;/&#x60; lists the root&#39;s top-level   directories (the separator says \&quot;I am at the root, show me   what it holds\&quot;); - otherwise the entries of the prefix&#39;s parent directory   whose names start with the prefix&#39;s last segment are   listed as full paths (typing &#x60;/mnt/fam&#x60; lists   &#x60;/mnt/family-archive&#x60;); - a trailing separator switches to the children of the   cleaned prefix itself (typing &#x60;/mnt/family-archive/&#x60;   drills into it), and no extra row rides along then -- the   trailing slash says \&quot;I am inside this directory\&quot;.  Only real directories are listed (symlinked entries are not), the rows are deduplicated, sorted ascending, and capped at 20 -- past the cap the user narrows by typing; there is no pagination. A prefix that matches nothing answers the empty list, never an error -- a relative prefix and an unreadable parent both answer &#x60;completions: []&#x60;.  Every row carries its write fact: as the listing lands, the server probes each row&#39;s directory once with the creation-time probe&#39;s own three steps (write a uniquely named probe file, read it back, remove it). A probe step&#39;s refusal is a FACT (&#x60;writable: false&#x60;), never an error -- the row lists alongside the others and the dialog renders the deletability switch only for the &#x60;writable&#x60; rows.  The completion runs a bounded read: at most one directory listing and at most 20 per-row probes, and the response carries at most 20 rows. 
  ///
  /// Parameters:
  /// * [prefix] - The path fragment typed so far. Absent or empty lists the filesystem root itself. Values past 4,096 bytes answer 400 -- the filesystem's own path ceiling, so no honest typing ever hits it. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumPathCompletionsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumPathCompletionsResponse>> completeAlbumPath({ 
    String? prefix,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/path-completions';
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
      if (prefix != null) r'prefix': encodeQueryParameter(_serializers, prefix, const FullType(String)),
    };

    final _response = await _dio.request<Object>(
      _path,
      options: _options,
      queryParameters: _queryParameters,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    AlbumPathCompletionsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumPathCompletionsResponse),
      ) as AlbumPathCompletionsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumPathCompletionsResponse>(
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

  /// Create one directory album from an existing path.
  /// Creates one directory album owned by the calling ADMINISTRATOR: the album IS a source -- the request names an existing directory the server validates and references in place, the files stay where they are (zero copy, zero move; the rescan that fills the album&#39;s membership is a later flow). No type field exists: this endpoint IS directory creation -- the curation-into-a-container story is gone (a memory is born through its own every-member face, POST /memories), and every account&#39;s sync album is born with the account.  Mounting an existing photo directory is a CONFIGURATION act: it points the application at a filesystem location, and a plain member never sees the container&#39;s filesystem at all. Creation is therefore the administrator&#39;s alone (the boundary ruling): a signed-in member whose role is not &#x60;admin&#x60; answers 403 &#x60;AUTH_FORBIDDEN&#x60; before any part of the request is considered -- the refusal-matrix semantics of the admin surfaces. Members contribute photos through the upload stream and are untouched by this guard. The trust boundary is the compose file&#39;s own mounts: the operator controls what the container can see, and the application layer neither can nor needs to restrict the operator further -- the mount-root allowlist of the previous design retired whole with this ruling, and no software-layer allowlist stands behind it.  The title is trimmed and must span 1-64 Unicode characters (the count is over runes, so multi-byte titles stay within the bound at the same length).  &#x60;rootPath&#x60; is the source directory, and it runs the path validation matrix before anything is written, each branch its own 400 error code:  - the path must be absolute (after a server-side clean; a   relative, empty or over-long value is a plain   &#x60;VALIDATION_FAILED&#x60;), - it must exist (&#x60;ALBUM_PATH_NOT_FOUND&#x60;), - it must be a directory (&#x60;ALBUM_PATH_NOT_DIRECTORY&#x60;), - the server must be able to list it   (&#x60;ALBUM_PATH_NOT_READABLE&#x60;), - the RESOLVED path (symlinks evaluated) must never sit   inside the server&#39;s own managed data tree   (&#x60;ALBUM_PATH_IN_MANAGED_TREE&#x60;) -- the one confine that   survived the allowlist retirement: the application&#39;s own   originals, derivatives and trash never become an album   source, whatever the container can see. The   allowlist-domain and root-overlap branches of the previous   matrix died with the allowlist: with creation in the   administrator&#39;s hands, a second album over the same tree   is an operator&#39;s visible choice, not an attack to refuse.  The album row stores the RESOLVED absolute path, so a later symlink swap can never repoint an existing album.  After the matrix passes, the server probes the directory&#39;s writability exactly ONCE: it writes a uniquely named probe file, reads it back, and removes it. A directory that refuses any step is a perfectly legitimate album source -- the probe&#39;s failure is NOT a creation failure -- and the fact lands on the row as &#x60;sourceWritable: false&#x60;, honestly reported in the response: photos of a read-only directory cannot be deleted through the application, and the creation answer says so instead of every later deletion discovering it. The truth is discovered at creation and never re-probed at runtime.  &#x60;audience&#x60; is the album&#39;s audience, the full grantee set of view ACL rows the creation lands in the same request: the add-style picker&#39;s chips of the dialog, empty (or absent) meaning \&quot;only me\&quot;. Its semantics are the audience write&#39;s: every id that cannot be granted -- an unknown or disabled account, or the caller&#39;s own id (the owner sees the album by construction) -- is skipped, never an error, and the response&#39;s &#x60;skipped&#x60; list reports each with its reason.  The two dialog switches ride the same request. The deletability switch (&#x60;allowAppDeletion&#x60;) lands on the row -- on by default, rendered by the dialog only where the source is writable (the completion row&#39;s &#x60;writable&#x60; fact; the read-only sources&#39; stored value governs nothing). The creator&#39;s timeline exclusion (&#x60;inTimeline: false&#x60;) lands the caller&#39;s own preference row in the same creation -- the album is born out of its creator&#39;s aggregated timeline, the audience&#39;s own feeds untouched.  The response is the created album&#39;s wall card -- the summary exactly as &#x60;GET /albums&#x60; lists it: &#x60;callerRole&#x60; &#x60;owner&#x60;, &#x60;type&#x60; &#x60;directory&#x60;, an empty &#x60;coverAssetIds&#x60; array (never null) and &#x60;itemCount&#x60; 0, nothing being a member yet -- plus the honest audience report and the &#x60;sourceWritable&#x60; fact. The card&#39;s &#x60;inTimeline&#x60; carries the exclusion&#39;s landed state. The write publishes one &#x60;album.created&#x60; event and one &#x60;album.shared&#x60; event per audience row that landed. 
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
  /// Returns a [Future] containing a [Response] with a [AlbumCreateResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumCreateResponse>> createAlbum({ 
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

    AlbumCreateResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumCreateResponse),
      ) as AlbumCreateResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumCreateResponse>(
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

  /// Create one memory.
  /// Creates one memory owned by the calling user: a reference set the owner curates photos into -- the photos themselves live in the albums they were ingested through, the memory holds references to them. The request carries a title and an audience, nothing else: a memory owns no &#x60;rootPath&#x60; (no source to name -- the directory album&#39;s own field is absent from this request and from the memory&#39;s every response shape), no deletability switch (a memory&#39;s delete never moves files), and no timeline switch (a memory never feeds its creator&#39;s own timeline, so there is nothing to switch).  EVERY member may create. Curation is personal expression, not configuration: a memory names no filesystem location, touches no source, and borrows nothing but photos the creator already sees -- the exact opposite gate of the directory album&#39;s administrator-only creation. Any signed-in account&#39;s POST succeeds; there is no role check to refuse.  The title is trimmed and must span 1-64 Unicode characters (the count is over runes, so multi-byte titles stay within the bound at the same length).  &#x60;audience&#x60; is the memory&#39;s audience, the full grantee set of view ACL rows the creation lands in the same request: absent (or empty) means \&quot;only me\&quot;. Its semantics are the audience write&#39;s: every id that cannot be granted -- an unknown or disabled account, or the caller&#39;s own id (the owner sees the memory by construction) -- is skipped, never an error, and the response&#39;s &#x60;skipped&#x60; list reports each with its reason.  The memory is born out of its creator&#39;s aggregated timeline: the creation writes the creator&#39;s own timeline preference row (&#x60;inTimeline: false&#x60;) in the same write, so the aggregate never feeds the memory&#39;s photos whatever future arms admit. The exclusion is the creator&#39;s own -- the audience members&#39; feeds are governed by their own preference rows, never by this write.  The response is the created memory&#39;s wall card -- the summary exactly as &#x60;GET /albums&#x60; lists it: &#x60;callerRole&#x60; &#x60;owner&#x60;, &#x60;type&#x60; &#x60;memory&#x60;, an empty &#x60;coverAssetIds&#x60; array (never null), &#x60;itemCount&#x60; 0 (nothing is a member yet), no &#x60;inTimeline&#x60; field (an own memory card omits it) -- widened with the honest audience report. The write publishes one &#x60;album.created&#x60; event and one &#x60;album.shared&#x60; event per audience row that landed. 
  ///
  /// Parameters:
  /// * [memoryCreateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [MemoryCreateResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<MemoryCreateResponse>> createMemory({ 
    required MemoryCreateRequest memoryCreateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/memories';
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
      const _type = FullType(MemoryCreateRequest);
      _bodyData = _serializers.serialize(memoryCreateRequest, specifiedType: _type);

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

    MemoryCreateResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(MemoryCreateResponse),
      ) as MemoryCreateResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<MemoryCreateResponse>(
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

  /// Remove one album.
  /// Removes the caller&#39;s album. The answer is 204 with no body, and the write publishes one &#x60;album.deleted&#x60; event -- the wall and the detail converge through it on every connected client.  The removal&#39;s database half is ONE transaction: the album row goes, and every &#x60;album_items&#x60; membership row, every &#x60;album_shares&#x60; row and every per-user timeline preference row follows through the schema&#39;s cascades.  What happens to the album&#39;s MEMBERS splits by the album&#39;s type. A &#x60;memory&#x60; album is a reference set over existing photos: removing it is the reference set&#39;s own plain cascade -- the album row, its membership rows, its audience and its per-user preference rows go, and NOTHING else does. The photos are not the memory&#39;s: they live in the albums they were ingested through, and every one of their states (the asset rows, the per-user favorites and tags, the faces, the generated files) is untouched, whatever their storage kind -- no reference guard runs, nothing is purged, no file is contacted. A &#x60;directory&#x60; album is a source, and removing it is the UN-INGEST of what its scans brought in: every member asset that NO OTHER album references (the album union -- the sync album&#39;s permanent membership covers every uploaded row, and a member shared with a second album survives on that reference) is cleaned from the library entirely -- the asset row and its per-user states (favorites, tag assignments) go, its faces follow (a face cluster left empty dissolves with its labels, a surviving member re-homes the cluster&#39;s representative), its CLIP embedding and OCR text go, and the GENERATED files (the thumbnails and the transcode outputs) are deleted. The row deletion is a hard delete: the library data is gone, not trashed. The member&#39;s SOURCE file is NEVER touched -- the album only ever referenced it where it lies, and the file remains the family&#39;s own. An external member currently in the trash (its file in the server&#39;s trash area) loses the row and the trash copy together. The file removals run after the transaction commits and are best-effort: a file the operating system refuses is left behind as inert garbage, never as a row that names it. Every purged member publishes one &#x60;asset.deleted&#x60; event (the Live Photo component rows stay silent, exactly like every other purge), so the timeline, the search and the people surfaces converge on the removal while it happens.  The sync album is no album resource: it answers the single-shape 404, whatever the caller&#39;s role. The same 404 answers an unknown id.  Authorization is the writes&#39; two-tier answer: an album the caller cannot see answers the single-shape 404; an album shared to the caller answers 403 &#x60;AUTH_FORBIDDEN&#x60; (the share is view-only); the owner -- or an administrator overriding the owner check -- removes. 
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
  /// The detail view of one album the caller can see: the caller owns it, or it is shared to the caller through an &#x60;album_shares&#x60; row. &#x60;callerRole&#x60; says which -- &#x60;owner&#x60; for the owner&#39;s write affordances, &#x60;viewer&#x60; for the read-only share -- with the one exception below.  The caller&#39;s OWN sync album answers 200: the default album&#39;s read-only detail. &#x60;callerRole&#x60; is &#x60;owner&#x60;, the live members page through the same keyset as any album&#39;s (the grid and the viewer work like any album&#39;s), but the album is system-managed: NO write affordance applies to it -- the property patch, the membership curation, the sharing face, the timeline preference and the delete all keep refusing a sync album with their unchanged 404/400 answers, whatever this read admits. &#x60;inTimeline&#x60; is absent (the core stream is never excludable) and the owner&#39;s &#x60;shares&#x60; list rides the response always empty (no share row can name a sync album). The client renders the localized display name (en \&quot;My photos\&quot;, zh-Hans \&quot;我的照片\&quot;) keyed by &#x60;type: &#39;sync&#39;&#x60; -- the stored title is the internal system label and never renders.  A sync album of ANOTHER account answers the same 404 as an unknown id, whatever share rows exist: invisible by design.  404 is single-shape by design: an unknown id, another account&#39;s unshared album, and another account&#39;s sync album all answer the same &#x60;ALBUM_NOT_FOUND&#x60; byte for byte -- the anti-enumeration discipline of &#x60;ASSET_NOT_FOUND&#x60; applied to the album face. An album whose share to the caller was revoked before the request simply sits in that same invisible region.  &#x60;itemCount&#x60; counts the album&#39;s live members; &#x60;items&#x60; is one keyset page of them, ordered &#x60;takenAt&#x60; descending with the asset id as the tiebreak -- the timeline&#39;s own ordering and cursor pair, so the album grid pages with the timeline&#39;s walking machinery. Every item carries the membership&#39;s &#x60;addedAt&#x60; and the minimal asset context the grid renders; &#x60;thumbnail&#x60; is the asset thumbnail endpoint&#39;s &#x60;tile&#x60;-size URL. Live members only: trashed, hidden and Live Photo component rows never appear (a hidden asset is invisible to the whole family).  &#x60;coverAssetId&#x60; is the raw pinned column: the single asset the owner chose, or null when unset -- the wall derives its collage fallback from the members, the detail carries the pin itself.  &#x60;rootPath&#x60; rides the &#x60;directory&#x60; albums only: the album&#39;s source, the resolved absolute path the row stores -- the edit dialog&#39;s read-only source line (the source is pinned at creation; the property patch carries no field for it). Memory and sync albums omit it: a memory has no source -- its photos live in the albums they were ingested through.  &#x60;sourceWritable&#x60; rides beside it, on the &#x60;directory&#x60; albums only: the creation-time write probe&#39;s fact, false for a read-only source. &#x60;allowAppDeletion&#x60; and &#x60;deletable&#x60; ride the owner&#39;s view of a &#x60;directory&#x60; album only: the deletability switch&#39;s stored state (the edit dialog&#39;s toggle) and the wall&#39;s delete-affordance derivation (owner, writable source, switch on -- computed server-side, pure data, no runtime probing). A viewer&#39;s response carries none of the three faces a write would need, and a read-only source never carries a true &#x60;deletable&#x60; whatever the switch says.  &#x60;shares&#x60; rides only the owner&#39;s view: the album&#39;s grantee list, one entry per &#x60;album_shares&#x60; row in the grant order (&#x60;grantedAt&#x60; ascending), each carrying the grantee&#39;s id, email and grant time -- the share management dialog&#39;s list. The owner&#39;s response always carries the field, an empty array when the album holds no grants; viewer responses never carry it at all (a sharee sees the album, never the ACL).  &#x60;inTimeline&#x60; is the caller&#39;s timeline preference of this album (the exclusion toggle&#39;s source of truth): present on the excludable types -- the caller&#39;s own &#x60;directory&#x60; albums and the albums shared to them -- and absent on the caller&#39;s own &#x60;memory&#x60; albums. The preference scopes the aggregate: it never filters this album&#39;s own items, so browsing a switched-off album still shows everything.  &#x60;limit&#x60; bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 &#x60;VALIDATION_FAILED&#x60;. Pages continue through the opaque &#x60;cursor&#x60;, the timeline&#39;s exact keyset semantics: &#x60;nextCursor&#x60; is present exactly when another page follows, and the last page carries none. 
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
  /// The album wall of the calling user: every album that deserves a wall card, in two groups concatenated into one array -- the web client renders its sections off the &#x60;callerRole&#x60; discriminator each item carries.  The caller&#39;s sync album is pinned FIRST, before the own group: the one card of type &#x60;sync&#x60;, the caller&#39;s own core upload stream surfaced as the default \&quot;My photos\&quot; album -- every upload lands in it, so a fresh account&#39;s wall is never empty. The own group follows: the albums the caller owns of type &#x60;memory&#x60; or &#x60;directory&#x60;, the most recently updated first (the album id descending as the tiebreak). The shared group follows last: every album shared to the caller through an &#x60;album_shares&#x60; row, the caller&#39;s most recently granted share first (the album id descending as the tiebreak). Shared items carry &#x60;owner&#x60; -- the honest attribution of another account&#39;s album: the owner&#39;s id and email.  A sync album of ANOTHER account never appears, in either group: the shared arm carries &#x60;memory&#x60; and &#x60;directory&#x60; albums only, whatever share rows exist. The pinned card&#39;s &#x60;title&#x60; is the system&#39;s internal fixed label (&#39;Sync&#39;) -- the display name is the CLIENT&#39;s: &#x60;type: &#39;sync&#39;&#x60; keys the localized default-album name (en \&quot;My photos\&quot;, zh-Hans \&quot;我的照片\&quot;), and the stored title never renders.  &#x60;coverAssetIds&#x60; is the wall card&#39;s collage source: the pinned &#x60;coverAssetId&#x60; as a single-element list when the owner set one, otherwise the album&#39;s most recent live member assets -- up to four, &#x60;takenAt&#x60; descending with the asset id as the tiebreak, fewer when the album holds fewer members, empty for an album with none. &#x60;itemCount&#x60; counts live members under the same predicate: trashed, hidden and Live Photo component rows never count (a hidden asset is invisible to the whole family).  &#x60;isShared&#x60; reports whether the album carries at least one &#x60;album_shares&#x60; row -- the owner&#39;s \&quot;someone else sees this\&quot; badge. It is true by construction on every shared-group item (the caller&#39;s share row is the reason the item exists).  &#x60;inTimeline&#x60; rides the excludable cards -- the own &#x60;directory&#x60; items and every shared item: the caller&#39;s timeline preference of that album, true when no preference row exists (the default of feeding the timeline). The own &#x60;memory&#x60; items omit the field: they never feed the caller&#39;s own timeline. The pinned &#x60;sync&#x60; card omits it too: the core stream is never excludable (the preference face answers 400 for it). The field is the toggle&#39;s source of truth for the wall&#39;s \&quot;excluded\&quot; badge, kept current by the &#x60;PUT/DELETE /albums/{albumId}/timeline-preference&#x60; endpoints. 
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

  /// Remove assets from one memory album.
  /// Removes membership rows from the caller&#39;s memory album: &#x60;{removed, skipped}&#x60;, the batch surface&#39;s set semantics -- the honest report, never per-item errors. An id that is no member of the album -- never was, or already removed -- is skipped; an all-skipped request still answers 200 with &#x60;removed&#x60; 0. Repeated ids collapse; the counts cover the distinct ids.  The removal carries no live predicate: a trashed member&#39;s row still leaves, the honest cleanup of the set (the live views filter trashed members out of the renders; removing the row tidies the set itself).  MEMORY albums only (directory and sync albums answer the single-shape 404); authorization is the writes&#39; two-tier answer (invisible 404, shared-to-caller 403, the owner or an admin override writes).  The write publishes one &#x60;album.items-changed&#x60; event exactly when at least one row was removed -- the idempotent no-op stays silent. 
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

  /// Start one rescan of a directory album&#39;s source.
  /// The rescan trigger: one run that walks the directory album&#39;s source root on the server and reconciles the album&#39;s membership with what the filesystem holds. The walk never follows symlinks -- a symlinked entry is skipped, a symlinked subdirectory is never descended -- and each walk entry and each live member resolves through exactly one arm of the three-arm diff:  - a file on disk no live member matches is ADDED by   reference-style ingest: a new asset row   (&#x60;storage_kind&#x60; &#x60;external&#x60;, &#x60;file_path&#x60; the file&#39;s real   absolute path, zero copy zero move) and its album   membership land in one transaction, and the existing   pipeline jobs (metadata, thumbnail, and for a video the   transcode) run over that file exactly like an   upload&#39;s. An owner+checksum duplicate of content the owner   already holds never creates a second row: a live duplicate   gains the album membership, a trashed duplicate is left   alone (no revival semantics). - a live member whose file is gone from disk is marked   &#x60;deleted_at&#x60; directly (the orphan arm&#39;s pinned semantics:   the file is not the server&#39;s and the trash has no meaning   for it) -- no missing state, no revival. If the file   returns, the next scan&#39;s add arm ingests it as a NEW row. - a member whose mtime+size snapshot changed has its   checksum re-read and stored, the snapshot re-stamped, and   its metadata pipeline re-run (EXIF re-extracted, thumbnail   regenerated, the playable copy of a video re-encoded); the   ML arm follows the pipeline&#39;s own re-run path, which does   not re-fire -- readiness is already stamped.  The run&#39;s &#x60;job_runs&#x60; row (&#x60;kind&#x60; &#x60;album_rescan&#x60;, carrying the album&#39;s id) surfaces progress the way the places re-resolution run does: &#x60;job.started&#x60; / &#x60;job.progress&#x60; / &#x60;job.finished&#x60; on the activity stream, &#x60;total&#x60;/&#x60;done&#x60; counting resolved walk entries plus vanished members and &#x60;failed&#x60; the entries the run could not resolve. The idempotent rescan is the pinned contract: a second run over an unchanged directory performs zero asset writes and publishes zero asset events (the run&#39;s own &#x60;job_runs&#x60; bookkeeping is the progress surface, not product state).  The endpoint is idempotent the way the places re-resolution is, scoped to the album: a rescan of THIS album still in flight (&#x60;queued&#x60; or &#x60;running&#x60;) answers 200 with its identifier and starts nothing; another album&#39;s rescan never blocks this one.  Permission is the album writes&#39; own: the owner (or an administrator) proceeds, a sharee answers the honest 403, everyone else folds into the album 404. A non-directory album -- the caller&#39;s own sync album, a memory album -- answers the same 404: the rescan is the directory album&#39;s own flow (a memory holds references to photos that live elsewhere -- there is no source to walk), and ids stay unproammable. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the directory album to rescan.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumRescanResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumRescanResponse>> rescanAlbum({ 
    required String albumId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}/rescan'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
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
      validateStatus: validateStatus,
    );

    final _response = await _dio.request<Object>(
      _path,
      options: _options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    AlbumRescanResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumRescanResponse),
      ) as AlbumRescanResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumRescanResponse>(
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
  /// Removes the calling user&#39;s preference row of one album: the album returns to the default state of feeding the caller&#39;s timeline (a missing row means kept in). The eligibility scope is the PUT&#39;s own -- the caller&#39;s own directory albums and the albums shared to the caller; the same 400 &#x60;album_pref_not_excludable&#x60; answers the caller&#39;s own sync and own memory albums, and the same single-shape 404 answers an album the caller cannot see.  The delete is idempotent: an album with no preference row (never set, or already reset) still answers 204 -- the row is gone either way, which is the state the caller asked for. Like the PUT it publishes no event: the preference is the caller&#39;s own state, and no other session has anything to refetch. 
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

  /// Set the caller&#39;s timeline preference of one album.
  /// Upserts the calling user&#39;s per-user timeline preference of one album: the &#x60;user_album_prefs&#x60; row keyed (userId, albumId) whose &#x60;inTimeline&#x60; switch says whether the album feeds the caller&#39;s aggregated timeline. The preference is the CALLER&#39;s own state over an album they can see -- self-affecting only, which is why these writes publish no event: the caller&#39;s own response is the only view that changed, and no other session has anything to refetch. The scope is the albums whose membership a caller could want out of their timeline feed: the caller&#39;s own &#x60;directory&#x60; albums, and every album shared to the caller (whatever the owner&#39;s type). Two kinds of album answer 400 instead, with the &#x60;album_pref_not_excludable&#x60; message key in the details:  - the caller&#39;s own &#x60;sync&#x60; album: it IS the core stream, and   the core stream is never excludable (the aggregation   formula exempts the sync arm from the preference row --   a stray row there excludes nothing); - the caller&#39;s own &#x60;memory&#x60; albums: they never feed the   caller&#39;s own timeline at all (in-stream curation would be   pure duplication), so there is nothing to exclude.  An album the caller cannot see -- unknown, another account&#39;s unshared album, a revoked share -- answers the same byte-identical 404 &#x60;ALBUM_NOT_FOUND&#x60; as every album face, so album ids cannot be probed for existence through this endpoint. There is no administrator override: the preference is always the caller&#39;s own row over their own view, never a write onto another account&#39;s state.  &#x60;inTimeline&#x60; is required and must be a JSON boolean: a body that is not valid JSON, or one without the field, answers 400 &#x60;VALIDATION_FAILED&#x60;. The write is an upsert: repeating it with the same value converges to the same row, and repeating it with the other value flips the switch -- the response always carries the stored state. 
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
  /// The owner&#39;s partial edit of the album&#39;s properties: &#x60;title&#x60;, &#x60;coverAssetId&#x60; and &#x60;allowAppDeletion&#x60;, any subset, at least one field (a body that carries none -- JSON null counts as absent, the patch discipline of the asset surface -- is a client bug and answers 400). The response is the album&#39;s wall card after the edit.  Authorization is the two-tier answer every album write pins: an album the caller cannot see at all -- an unknown id, another account&#39;s unshared album, every sync album -- answers the single-shape 404 &#x60;ALBUM_NOT_FOUND&#x60;, byte-identical to the read face, so album ids cannot be probed; an album the caller CAN see but does not own (an album shared to the caller) is an honest authorization failure -- 403 &#x60;AUTH_FORBIDDEN&#x60;, the share is view-only. The owner writes, and an administrator may override the owner check (the admin role bypasses it, the permanent delete&#39;s own discipline).  Cover validation (the application-layer check): a present &#x60;coverAssetId&#x60; must be a LIVE member of THIS album -- the membership row exists and the asset passes the live predicate (not trashed, not hidden, not a Live Photo component row). Any other id -- an asset that is no member, a trashed, hidden or component row, another account&#39;s asset -- answers 400 &#x60;VALIDATION_FAILED&#x60; carrying the message key &#x60;album_cover_not_member&#x60;. A &#x60;coverAssetId&#x60; that is absent (or JSON null, which decodes as absent) leaves the pin unchanged.  The deletability switch (&#x60;allowAppDeletion&#x60;) is the directory album&#39;s own field: a patch that carries it on a memory album answers 400 &#x60;VALIDATION_FAILED&#x60; with the message key &#x60;album_deletable_not_directory&#x60; (a memory album&#39;s deletions never move files, so the switch has no home there). The write is idempotent -- repeating the stored value converges.  The write publishes one &#x60;album.updated&#x60; event on every successful patch, whatever fields changed -- the convergent refetch makes the unconditional fire harmless. 
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

  /// Set one album&#39;s audience (the full grantee set).
  /// The album&#39;s audience as ONE declarative write: the request carries the complete set of accounts that may view the album, and the server converges the &#x60;album_shares&#x60; rows onto exactly that set -- grants land for the ids that are not audience yet, rows leave for the grantees the set no longer names. \&quot;Who can see\&quot; is an album property; this face is its whole write surface (the one-row grant/revoke pair retired with it -- one concept, one endpoint).  The write is IDEMPOTENT: the same set twice answers the same state, the second request lands no row and fires no event (&#x60;granted&#x60; and &#x60;revoked&#x60; both empty). An empty (or all-skipped) set is the honest \&quot;only me\&quot;: every row leaves -- and the owner&#39;s own access is untouched by construction, because the owner&#39;s visibility is the ownership row, never a share row: the owner is not removable through this face whatever the set says.  Skipped entries -- the requested ids that cannot become audience -- are reported honestly, never errors: an unknown or disabled account (&#x60;invalid_account&#x60; -- one reason for both, so user ids cannot be probed through this face) and the album&#39;s own owner (&#x60;owner&#x60; -- the owner sees the album by construction; a row for them would be noise). Everything else the request names lands or stays.  Authorization is the writes&#39; two-tier answer: an album the caller cannot see at all -- an unknown id, another account&#39;s unshared album, every sync album (the caller&#39;s own included: the sync stream&#39;s face is the timeline, and its audience edit folds into the same single-shape 404) -- answers the reads&#39; byte-identical &#x60;ALBUM_NOT_FOUND&#x60;, so album ids cannot be probed; an album the caller CAN see but does not own (an album shared to the caller) answers the honest 403 &#x60;AUTH_FORBIDDEN&#x60; -- the share is view-only, a sharee cannot edit the ACL. The owner writes, and an administrator may override the owner check. The write&#39;s own authorization is the whole gate: no album state gates or filters this face.  The write publishes one &#x60;album.shared&#x60; event per row that landed and one &#x60;album.unshared&#x60; event per row that left -- the grant/revoke events&#39; discipline carried over: the payload carries the album&#39;s id and the GRANTEE&#39;s user id (the account whose view of the album changed), and the idempotent no-op stays silent. 
  ///
  /// Parameters:
  /// * [albumId] - Identifier of the album.
  /// * [albumAudienceRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AlbumAudienceResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AlbumAudienceResponse>> updateAlbumAudience({ 
    required String albumId,
    required AlbumAudienceRequest albumAudienceRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/albums/{albumId}/audience'.replaceAll('{' r'albumId' '}', encodeQueryParameter(_serializers, albumId, const FullType(String)).toString());
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
      const _type = FullType(AlbumAudienceRequest);
      _bodyData = _serializers.serialize(albumAudienceRequest, specifiedType: _type);

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

    AlbumAudienceResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AlbumAudienceResponse),
      ) as AlbumAudienceResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AlbumAudienceResponse>(
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
