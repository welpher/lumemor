//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'dart:typed_data';
import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/asset_detail.dart';
import 'package:lumemor_api/src/model/asset_patch.dart';
import 'package:lumemor_api/src/model/asset_response.dart';
import 'package:lumemor_api/src/model/batch_request.dart';
import 'package:lumemor_api/src/model/batch_response.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/timeline_response.dart';
import 'package:lumemor_api/src/model/trash_response.dart';

class AssetsApi {

  final Dio _dio;

  final Serializers _serializers;

  const AssetsApi(this._dio, this._serializers);

  /// Apply one set action to a batch of assets.
  /// The batch surface of the curation actions: one request applies one action -- favorite, unfavorite, hide, unhide, trash or restore -- to a list of up to 1000 asset ids, in one transaction. The endpoint follows set semantics: it is a convergence operation, not a state-change counter. Every id that resolves against the action&#39;s guard is applied (an id already in the target state is the idempotent no-op and still counts as affected), and every id that does not resolve -- unknown, or one the guard does not cover, the two indistinguishable -- is ignored and reported in the response&#39;s &#x60;missing&#x60; list. The two lists partition the request&#39;s distinct ids: no individual id ever fails the request, and a repeat of a successful batch converges to the same answer.  The guards differ by action, exactly as the single-asset faces do: favorite and unfavorite resolve CALLER-VISIBLE ids -- the caller&#39;s own assets plus the assets of every album shared to them, the same union the timeline aggregates (a sharee favorites what they see) -- while hide, unhide, trash and resolve&#39;s restore arm stay OWNER-scoped: curation and the trash lifecycle are the owner&#39;s alone, so another account&#39;s asset resolves for a sharee only through the favorite arms and reports as missing through every other.  This is the deliberate trade of the list surface: the byte-identical 404 anti-enumeration of the single-asset endpoints does not apply here. The caller handed the ids over themselves, so reporting the unresolvable ones back in &#x60;missing&#x60; hides nothing -- and hands honest clients their retry list.  &#x60;ids&#x60; must carry between 1 and 1000 entries: an empty list, or one past the cap, answers 400 &#x60;VALIDATION_FAILED&#x60;, as does an &#x60;action&#x60; outside the enum or a body that is not valid JSON (an id that is not a uuid fails the body decode). Repeated ids collapse to one application.  The whole batch commits as one transaction: the action&#39;s writes land together or not at all, so a partial batch never surfaces. Failures on resolvable ids are not expected -- the per-item guards turn foreign and already-in-state ids into no-ops, not errors -- so the all-or-nothing bound is the infrastructure safety, not a client-visible case.  The actions resolve to their single-row primitives: favorite / unfavorite write or remove the caller&#39;s per-user favorites rows; hide / unhide flip the owner-level visibility switch of the row the id names (a hidden asset leaves every live view -- the timeline, the favorites -- until unhidden); trash / restore run the trash lifecycle of the trash endpoints, Live Photo pair included (trashing or restoring the photo carries its component video row along, whichever row the id names).  Events publish AFTER the commit, one per asset whose state actually changed -- the idempotent no-op stays silent, the same discipline as the single-asset restore: &#x60;asset.trashed&#x60; and &#x60;asset.restored&#x60; for trash and restore, &#x60;asset.updated&#x60; for hide and unhide (visibility is owner-level state, and every viewer of the shared timeline sees it change). favorite / unfavorite publish nothing: favorites are per-user state with no cross-user visibility effect, so no other session has anything to refresh. 
  ///
  /// Parameters:
  /// * [batchRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [BatchResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<BatchResponse>> batchAssets({ 
    required BatchRequest batchRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/batch';
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
      const _type = FullType(BatchRequest);
      _bodyData = _serializers.serialize(batchRequest, specifiedType: _type);

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

    BatchResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(BatchResponse),
      ) as BatchResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<BatchResponse>(
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

  /// Permanently delete one asset, files and all.
  /// The explicit permanent deletion -- the destructive surface beside the trash. It removes the asset row whatever its state (live or trashed: this endpoint does not route through the trash), and with it the asset&#39;s on-disk artifacts under the shared-blob rule: content addressing means another owner&#39;s row (or any other row) with the same checksum references the SAME physical file, so the original file is removed only when no other asset row references it; otherwise the row goes and the file stays. The derived serving copies (the per-asset thumbnail directory) belong to this asset id alone and are always removed. The component video row of a Live Photo is deleted with its photo (it is never an asset of its own).  The application-layer permission gate of the two-layer delete discipline: the owner of the asset, or an administrator. An id of another user&#39;s asset answers the same 404 &#x60;ASSET_NOT_FOUND&#x60; as an id that does not exist, byte for byte, so asset ids cannot be enumerated. (The file-system layer of the gate matters for directory albums, whose files move on delete; it arrives with the album domain in a later milestone -- blob originals live inside the server&#39;s own data directory.)  The ordinary client \&quot;delete\&quot; is the trash (soft delete through the batch endpoint); this endpoint is the explicit bypass: the client that already knows it wants the asset gone forever, and the empty-trash surface. Repeating the request after a successful deletion answers 404 -- the asset no longer exists, exactly like an unknown id.  The deletion is committed before the files are removed: a failure between the two (a full disk, a failing mount) leaves the row gone and the file orphaned, which is a storage leak, never data loss; it is logged server-side and the response still reports the deletion. 
  ///
  /// Parameters:
  /// * [id] - Identifier of the asset.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> deleteAsset({ 
    required String id,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/{id}'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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

  /// Fetch the detail of one owned asset.
  /// The asset detail: the identity and storage fields of the upload response plus the refinements the pipeline produced afterwards -- dimensions, duration, blurhash, the EXIF hot fields, and readiness -- the two owner-level editable attributes (&#x60;description&#x60;, &#x60;isHidden&#x60;), which the asset detail view renders and the &#x60;PATCH&#x60; endpoint edits, the asset&#39;s faces with the caller&#39;s own label of each face&#39;s cluster (&#x60;faces&#x60;), which the detail view&#39;s people section renders and the people write endpoints maintain, and the caller&#39;s own tags on the asset (&#x60;tags&#x60;, name ascending), which the detail view&#39;s tag section renders and the tags endpoints maintain -- another account&#39;s tags on the same asset never surface here (B4).  Access is the caller-visible gate: the asset&#39;s owner, and every member of an album the caller sees -- their own sync and directory albums plus every album shared to them (the same album union the timeline aggregates). An id none of those cover answers the same 404 &#x60;ASSET_NOT_FOUND&#x60; as an id that does not exist, byte for byte, so asset ids cannot be enumerated. The owner&#39;s reads keep their current shape: a trashed or hidden asset answers 200 (&#x60;isTrashed&#x60;, &#x60;isHidden&#x60; honest) because the owner&#39;s album memberships outlive both marks; for a sharee the same holds by id -- the album views and the timeline list live members only, so a hidden or trashed shared asset surfaces nowhere but stays readable at its known id until the share itself goes.  A trashed asset answers 200 with &#x60;isTrashed: true&#x60; -- the trash view reads the detail the same way. The component video row of a Live Photo is readable by its id (the detail, the original download and the transcoded play stream resolve it through &#x60;livePhotoVideoId&#x60;; it counts a transcode job of its own) but never appears in the timeline. 
  ///
  /// Parameters:
  /// * [id] - Identifier of the asset.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AssetDetail] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AssetDetail>> getAsset({ 
    required String id,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/{id}'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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

    AssetDetail? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AssetDetail),
      ) as AssetDetail;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AssetDetail>(
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

  /// Look up an uploaded asset by its device identifier.
  /// The backup idempotency probe: the app asks, before uploading a local file, whether its &#x60;deviceAssetId&#x60; already exists for this account, and skips the upload when it does.  The lookup is owner-scoped -- a &#x60;deviceAssetId&#x60; of another user&#39;s asset is invisible and answers 404. The &#x60;state&#x60; field mirrors what a re-upload of the same content would produce: &#x60;duplicate&#x60; when the asset is live, &#x60;trashed&#x60; when it sits in the trash (the client can then offer restore or save-as-new before spending bandwidth). 404 means the device has not uploaded this identifier (or only under another account). 
  ///
  /// Parameters:
  /// * [deviceAssetId] - The device-side identity the asset was uploaded with.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AssetResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AssetResponse>> getAssetByDeviceId({ 
    required String deviceAssetId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/by-device-id/{deviceAssetId}'.replaceAll('{' r'deviceAssetId' '}', encodeQueryParameter(_serializers, deviceAssetId, const FullType(String)).toString());
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

    AssetResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AssetResponse),
      ) as AssetResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AssetResponse>(
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

  /// Download the original file of one owned asset.
  /// Streams the stored original bytes as a download. It is available immediately after the upload: the original is the uploaded content-addressed file itself, not a pipeline product.  The download is always an attachment: &#x60;Content-Disposition: attachment; filename*&#x3D;UTF-8&#39;&#39;...&#x60; with the original file name escaped per RFC 5987 -- every byte outside the attribute character set is percent-encoded, which rules out header injection and in-browser rendering alike (SVG is not in the upload whitelist, so the download-only rule covers every stored format). &#x60;Content-Type&#x60; is the asset&#39;s stored MIME type. The response carries the same &#x60;ETag&#x60; and &#x60;Cache-Control&#x60; validators as the thumbnail (the original bytes are content-addressed, hence immutable).  Byte ranges follow the Go standard library&#39;s &#x60;ServeContent&#x60; semantics: a satisfiable single range answers 206 with &#x60;Content-Range&#x60;; multiple ranges answer a &#x60;multipart/byteranges&#x60; body; an unsatisfiable range answers 416 with the total size in &#x60;Content-Range&#x60;; a malformed Range header is ignored and the full body answers 200.  The same caller-visible gate as every asset endpoint: the owner, or a member of an album the caller sees -- foreign and unknown ids answer byte-identical 404 &#x60;ASSET_NOT_FOUND&#x60;. 
  ///
  /// Parameters:
  /// * [id] - Identifier of the asset.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [Uint8List] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<Uint8List>> getAssetOriginal({ 
    required String id,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/{id}/original'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
    final _options = Options(
      method: r'GET',
      responseType: ResponseType.bytes,
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

    Uint8List? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : rawResponse as Uint8List;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<Uint8List>(
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

  /// Serve a derived thumbnail, or a video&#39;s poster frame.
  /// The static serving copy of the asset:  - image assets: the AVIF thumbnail of the requested size --   &#x60;tile&#x60; (fits the 250px timeline box), &#x60;view&#x60; (long edge   capped at 1080px), &#x60;original&#x60; (the full-resolution   re-encode) -- served as &#x60;image/avif&#x60;; - video assets: the poster frame, whatever &#x60;size&#x60; says -- the   poster (the first frame, scaled into the tile box) is the   video&#39;s thumbnail, while the playable stream is the   transcoded copy of &#x60;GET /assets/{id}/play&#x60; -- served as   &#x60;image/jpeg&#x60;.  The response is a private immutable static asset: &#x60;Cache-Control: private, max-age&#x3D;31536000, immutable&#x60;, and a strong &#x60;ETag&#x60; built from the first 16 hex characters of the asset&#39;s checksum (quoted). A matching &#x60;If-None-Match&#x60; answers 304 with the same validators and no body. Conditional requests and byte ranges otherwise follow the Go standard library&#39;s &#x60;ServeContent&#x60; semantics.  The asset exists but its derived file is absent -- the thumbnail job has not run yet, or failed and is retrying -- answers 404 &#x60;ASSET_NOT_READY&#x60;: the client renders the blurhash or a placeholder and waits for the &#x60;asset.ready&#x60; event instead of treating the asset as gone. &#x60;ASSET_NOT_FOUND&#x60;, by contrast, means the id is unknown or not caller-visible: the detail endpoint&#39;s album-union gate applies here too (the owner, or a member of an album the caller sees), with byte-identical 404 bodies for foreign and unknown ids.  &#x60;size&#x60; must be one of the three values below; anything else (including a missing &#x60;size&#x60;) answers 400 &#x60;VALIDATION_FAILED&#x60;. 
  ///
  /// Parameters:
  /// * [id] - Identifier of the asset.
  /// * [size] - The serving size: `tile` for the timeline, `view` for the inline display, `original` for the full-resolution serving copy. Video assets answer the poster frame for every value. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [Uint8List] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<Uint8List>> getAssetThumbnail({ 
    required String id,
    required String size,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/{id}/thumbnail'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
    final _options = Options(
      method: r'GET',
      responseType: ResponseType.bytes,
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
      r'size': encodeQueryParameter(_serializers, size, const FullType(String)),
    };

    final _response = await _dio.request<Object>(
      _path,
      options: _options,
      queryParameters: _queryParameters,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    Uint8List? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : rawResponse as Uint8List;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<Uint8List>(
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

  /// The caller&#39;s favorites, one keyset page at a time.
  /// The favorites view: the assets the calling user favorites, filtered to the live view -- hidden assets, assets in the trash and the component video rows of Live Photos never appear -- with every item carrying &#x60;isFavorite: true&#x60; by construction (the listing joins the caller&#39;s favorites rows).  The view shares the timeline&#39;s shape, ordering and pagination: &#x60;takenAt&#x60; descending with the id as the tiebreak, &#x60;limit&#x60; absent meaning 100 and capped at 200, and the opaque &#x60;cursor&#x60; handed back as &#x60;nextCursor&#x60; precisely when another page follows, so the same wall UI that walks the timeline walks the favorites.  Favorites are per-user state written by the batch endpoint&#39;s favorite action, and the view is the caller&#39;s favorites rows crossed with CALLER-VISIBLE assets: the caller&#39;s own, plus the assets of every album shared to them -- the same album union the timeline aggregates. A sharee&#39;s favorites of shared assets appear here; a revoked share removes the assets from the view (the favorite row itself survives and resurfaces if the album is shared again). Hiding (or trashing) a favorited asset removes it from this view; it returns when unhidden (or restored), the favorite row untouched -- an idempotent per-user mark, not a copy of the asset. 
  ///
  /// Parameters:
  /// * [cursor] - The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
  /// * [limit] - Page size: the number of timeline items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TimelineResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TimelineResponse>> getFavorites({ 
    String? cursor,
    int? limit = 100,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/favorites';
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

    TimelineResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TimelineResponse),
      ) as TimelineResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TimelineResponse>(
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

  /// The caller&#39;s hidden assets, one keyset page at a time.
  /// The hidden view: the caller&#39;s own assets whose owner-level visibility switch is off. Live rows only -- a trashed asset stays in the trash view until restored or expired, and a restored one returns here still hidden -- and no Live Photo component rows (the pair is represented by its photo). The same ordering and keyset pagination as the timeline: &#x60;takenAt&#x60; descending with the id as the tiebreak, &#x60;limit&#x60; absent meaning 100 and capped at 200, &#x60;nextCursor&#x60; exactly when another page follows.  Unhiding through the batch endpoint returns an asset to every live view (the timeline, the favorites if favorited). &#x60;isFavorite&#x60; still reports the caller&#39;s favorites state of a hidden asset -- the favorite row survives the hiding, exactly like it survives the trash. 
  ///
  /// Parameters:
  /// * [cursor] - The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
  /// * [limit] - Page size: the number of timeline items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TimelineResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TimelineResponse>> getHidden({ 
    String? cursor,
    int? limit = 100,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/hidden';
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

    TimelineResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TimelineResponse),
      ) as TimelineResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TimelineResponse>(
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

  /// The aggregated timeline, one keyset page at a time.
  /// The aggregated timeline of the caller: the union of the albums that feed it -- the caller&#39;s sync album (the core backup stream, never excludable), the caller&#39;s own directory albums, and every album shared to the caller regardless of type (virtual albums included) -- minus the albums the caller turned off with the per-user \&quot;keep out of the timeline\&quot; preference (a missing preference row means kept in). The caller&#39;s own virtual albums never feed their own timeline: in-stream curation would be pure duplication. One asset visible through several albums appears once.  Every item is live by construction: hidden assets, assets in the trash, and the component video rows of Live Photos never appear.  &#x60;isFavorite&#x60; is per-user state: it reports whether the CALLING user favorites the asset (the favorites rows the batch endpoint writes), whatever the asset&#39;s owner -- the same item is a favorite for one viewer and a plain tile for another.  Pages are keyset-paginated with an opaque cursor: the response carries &#x60;nextCursor&#x60; exactly when another page exists, and the client passes it back as &#x60;cursor&#x60; to continue after the last item of the page. Cursors are stable for the ordering key (&#x60;takenAt&#x60; descending, &#x60;id&#x60; as the tiebreak); an asset uploaded with an older &#x60;takenAt&#x60; than the cursor appears only in a fresh pagination run, never in the current one. The first page omits the &#x60;cursor&#x60; parameter.  &#x60;limit&#x60; bounds the page size: absent means the default of 100, a value over 200 is answered with the maximum of 200, and a value below 1 answers 400 &#x60;VALIDATION_FAILED&#x60;.  &#x60;albumId&#x60; narrows the response to that album&#39;s own timeline instead of the aggregate: the album&#39;s items in the same ordering and filtering. The caller must be able to see the album (own album of any type, or an album shared to the caller); an album that does not exist or is not visible answers the empty list, not an error, so album ids cannot be probed for existence through this endpoint. 
  ///
  /// Parameters:
  /// * [cursor] - The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
  /// * [limit] - Page size: the number of timeline items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400. 
  /// * [albumId] - Restrict the response to the timeline of this one album (an album the caller owns or that is shared to the caller) instead of the aggregate. An unknown or invisible album answers the empty list. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TimelineResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TimelineResponse>> getTimeline({ 
    String? cursor,
    int? limit = 100,
    String? albumId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/timeline';
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
      if (albumId != null) r'albumId': encodeQueryParameter(_serializers, albumId, const FullType(String)),
    };

    final _response = await _dio.request<Object>(
      _path,
      options: _options,
      queryParameters: _queryParameters,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );

    TimelineResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TimelineResponse),
      ) as TimelineResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TimelineResponse>(
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

  /// List the caller&#39;s trashed assets with their retention countdown.
  /// The trash view of the caller: every asset the caller soft deleted, newest deletion first, each with the countdown of whole days until the expiry job removes it. Retention is the instance setting &#x60;trash_retention_days&#x60; (default 30): the expiry job runs hourly and permanently deletes rows whose &#x60;deletedAt&#x60; fell past the retention window, so an item shows &#x60;remainingDays&#x60; 0 on its way out (the row survives only until the next job run) and disappears afterwards -- restoring it then answers 404, it does not resurrect.  &#x60;remainingDays&#x60; counts the days left rounded up: an asset trashed moments ago with the default retention shows 30. The list is owner-scoped and bounded: it answers at most the 200 most recent deletions (the trash is a recovery view for recent deletes, and older items age out through expiry long before the bound could bite). The component video rows of Live Photos never appear: the pair is represented by its photo. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TrashResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TrashResponse>> getTrash({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/trash';
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

    TrashResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TrashResponse),
      ) as TrashResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TrashResponse>(
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

  /// Edit the description or hidden switch of one asset.
  /// The partial edit of the two owner-level attributes of an asset: &#x60;description&#x60;, the free text the detail panel renders, and &#x60;isHidden&#x60;, the owner-level visibility switch that keeps the asset out of every live view until it returns. Both body fields are optional but at least one must be present (a field set to JSON null counts as absent, so a body of nulls is the empty patch); a request that carries neither answers 400 &#x60;VALIDATION_FAILED&#x60;. Only the fields the request carries update -- the rest keep their values -- and the response is the full asset detail after the edit, the same shape &#x60;GET /assets/{id}&#x60; serves.  The edit follows set semantics: patching a field with the value the row already carries is the idempotent no-op -- the answer stays the 200 success with the row&#39;s current state, nothing is written (not even &#x60;updated_at&#x60;), and NO &#x60;asset.updated&#x60; event fires. A patch that actually changes either field publishes exactly ONE &#x60;asset.updated&#x60; event (the asset id and kind, after the commit): both fields are owner-level asset state, so the same announcement refreshes every viewer of the shared timeline and the detail panel. The description alone has no timeline effect -- it is detail-panel content -- but it is asset state the event covers all the same. The response serves the caller&#39;s tags on the asset beside the faces, both unchanged by this edit (the tags endpoints maintain them): the full detail shape &#x60;GET /assets/{id}&#x60; serves.  Access is owner-only, the attribute-editing row of the permission matrix: there is deliberately no administrator override here (unlike the permanent delete), so an id that is unknown or foreign answers the same 404 &#x60;ASSET_NOT_FOUND&#x60;, byte for byte, as every asset endpoint -- asset ids cannot be enumerated. A trashed asset is editable like it is readable (the detail answers 200 with &#x60;isTrashed: true&#x60;): the edit lands on the row and resurfaces with the restore. The hidden switch this endpoint flips is the same one the batch &#x60;hide&#x60; / &#x60;unhide&#x60; actions flip, so the two surfaces agree on the state of an asset.  &#x60;description&#x60; carries at most 10,000 Unicode characters (runes); above that the request answers 400 &#x60;VALIDATION_FAILED&#x60;. The value replaces the stored text wholesale -- an empty string is a valid patch and clears it. 
  ///
  /// Parameters:
  /// * [id] - Identifier of the asset.
  /// * [assetPatch] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AssetDetail] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AssetDetail>> patchAsset({ 
    required String id,
    required AssetPatch assetPatch,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/{id}'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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
      const _type = FullType(AssetPatch);
      _bodyData = _serializers.serialize(assetPatch, specifiedType: _type);

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

    AssetDetail? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AssetDetail),
      ) as AssetDetail;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AssetDetail>(
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

  /// Stream the transcoded playable copy of one owned video.
  /// Streams the web-safe transcoded MP4 of a video asset: the pipeline&#39;s transcode job re-encoded the original into H.264 video (yuv420p) plus AAC audio -- the pair every browser plays natively -- with the movie index relocated to the front of the container (&#x60;+faststart&#x60;), so playback begins while the file is still downloading. The stream is served inline as &#x60;video/mp4&#x60; with no &#x60;Content-Disposition&#x60;: it feeds the client&#39;s video element, while downloading the original bytes stays on &#x60;GET /assets/{id}/original&#x60; (the transcoded copy is a derivative, never the archival original).  The response is a private immutable static asset like the thumbnail: &#x60;Cache-Control: private, max-age&#x3D;31536000, immutable&#x60;, and a strong &#x60;ETag&#x60; built from the first 16 hex characters of the asset&#39;s checksum (quoted) -- the transcode is a deterministic function of the immutable original bytes, so the tag is stable across re-runs of the job. A matching &#x60;If-None-Match&#x60; answers 304 with the same validators and no body.  Byte ranges follow the Go standard library&#39;s &#x60;ServeContent&#x60; semantics (video seeking needs them): a satisfiable single range answers 206 with &#x60;Content-Range&#x60;; multiple ranges answer a &#x60;multipart/byteranges&#x60; body; an unsatisfiable range answers 416 with the total size in &#x60;Content-Range&#x60;; a malformed Range header is ignored and the full body answers 200.  Only the video family has a playable stream: a standalone video and a Live Photo&#39;s component video row alike (linked through &#x60;livePhotoVideoId&#x60; -- since the component counts a transcode job of its own, its playable copy arrives exactly like any video&#39;s, and the pair&#39;s motion plays through this endpoint by the component&#39;s id). An image asset answers 404 &#x60;ASSET_NOT_FOUND&#x60; -- there is nothing to play by that id, and the answer is the asset&#39;s own not-found envelope, the same anti-enumeration shape (byte for byte) as an unknown or a foreign id. A video whose transcode has not been produced yet -- the pipeline is still running, or the transcode job failed and is retrying -- answers 404 &#x60;ASSET_NOT_READY&#x60;: the client renders the poster frame (the thumbnail endpoint serves it) and waits for the &#x60;asset.ready&#x60; event instead of treating the asset as gone; for a Live Photo component, whose readiness the pair&#39;s photo announces on its own schedule, the client keeps the still photo until the stream appears. The standalone video pipeline counts three jobs (metadata, thumbnail, transcode), so its readiness follows all three; the component&#39;s counter counts its transcode alone. 
  ///
  /// Parameters:
  /// * [id] - Identifier of the asset.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [Uint8List] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<Uint8List>> playAsset({ 
    required String id,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets/{id}/play'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
    final _options = Options(
      method: r'GET',
      responseType: ResponseType.bytes,
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

    Uint8List? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : rawResponse as Uint8List;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<Uint8List>(
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

  /// Restore one trashed asset.
  /// Clears the trash mark of one asset: the row becomes live again and re-enters the timeline (its album memberships were never removed -- a trashed asset is filtered out of the live views, not evicted from its albums). The component video row of a Live Photo is restored with its photo.  The operation follows set semantics: restoring an asset that is already live succeeds (204, nothing to do) -- a client that races another restore, or asks twice, converges instead of erroring; the row is already in the desired state. Only an id that is unknown, or that belongs to another account while the caller is not its owner, answers 404 -- byte-identical for both cases, so asset ids cannot be enumerated. An asset whose row the expiry job already removed answers 404 as well: a purge is not reversible.  The retention countdown of the trash list is the only signal that a restore is still possible; after the row is gone, the content itself can only return through a re-upload (which the upload idempotency answers as a fresh asset). 
  ///
  /// Parameters:
  /// * [id] - Identifier of the trashed asset.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> restoreAsset({ 
    required String id,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/trash/{id}/restore'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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

    return _response;
  }

  /// Upload one photo, optionally with its Live Photo video.
  /// The backup upload endpoint: one photo per request, streamed into content-addressed storage while it is read, with the Live Photo pairing video riding the same request as a second file part.  The request body is &#x60;multipart/form-data&#x60; with these parts:  - &#x60;deviceAssetId&#x60; (text, required) -- the stable identity the   uploading device assigned to the file; together with the   account it scopes the device idempotency of backups. - &#x60;fileName&#x60; (text, required) -- the original client-side file   name of the photo, recorded as the asset&#39;s   &#x60;originalFilename&#x60;. - &#x60;deviceCreatedAt&#x60; (text, optional) -- the capture time the   device reports, RFC 3339 with offset. It seeds the asset&#39;s   &#x60;takenAt&#x60; until the metadata pipeline refines it from EXIF. - &#x60;deviceTimeZone&#x60; (text, optional) -- the IANA time zone name   of the capture location; it feeds the metadata pipeline&#39;s   timezone fallback chain and is not validated at upload. - &#x60;photo&#x60; (file, required) -- the photo bytes. - &#x60;liveVideo&#x60; (file, optional) -- the pairing video of a Live   Photo; present only together with &#x60;photo&#x60;.  Part order is part of the contract: all text fields precede every file part, and the &#x60;liveVideo&#x60; part follows the &#x60;photo&#x60; part. The server streams file parts to disk without buffering them, so every field must be known and validated before the first file byte is accepted -- a file part arriving before the required fields, a text field after a file part, or a &#x60;liveVideo&#x60; without a preceding &#x60;photo&#x60; answers 400 &#x60;VALIDATION_FAILED&#x60;. Each text field value is capped at 1024 bytes; &#x60;deviceAssetId&#x60;, &#x60;fileName&#x60; and &#x60;deviceTimeZone&#x60; carry the length bounds below. The storage extension and the media kind derive from the *file part&#39;s own* file name, and the payload must be consistent with it: a renamed file of another type answers 400 &#x60;VALIDATION_FAILED&#x60;.  Uploads are idempotent through the owner-scoped checksum: re-uploading identical bytes never creates a second asset. The outcome is one of three states, discriminated by the &#x60;state&#x60; field of the response body:  - 201 &#x60;created&#x60;: no asset with this checksum existed; a new   asset was created (a Live Photo created two rows, the   response describes the photo). - 200 &#x60;duplicate&#x60;: an identical asset of the same owner is   live; that asset is returned. - 200 &#x60;trashed&#x60;: an identical asset of the same owner sits in   the trash; that asset is returned (with &#x60;isTrashed: true&#x60;)   so the client can offer the user restore or save-as-new.  Each file is capped at the instance&#39;s per-file upload limit (2 GiB by default); a file over the limit aborts mid-stream with 413 &#x60;ASSET_TOO_LARGE&#x60;. This endpoint takes multipart bodies, not JSON, and is deliberately exempt from the API-wide JSON body cap; the per-file upload limit governs instead. 
  ///
  /// Parameters:
  /// * [deviceAssetId] - Stable device-side identity of the uploaded file.
  /// * [fileName] - Original client-side file name of the photo.
  /// * [photo] - The photo bytes; their file part name carries the extension.
  /// * [deviceCreatedAt] - Capture time reported by the device (RFC 3339 with offset). Optional; when absent the server records the upload time until the metadata pipeline refines it. 
  /// * [deviceTimeZone] - IANA time zone name of the capture location, for the metadata pipeline's timezone fallback chain. Not validated at upload time. 
  /// * [liveVideo] - The Live Photo pairing video; present only together with `photo`. The server stores it as a linked component row that never appears in the timeline, and queues its transcode so the pair's motion is playable in the browser. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AssetResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AssetResponse>> uploadAsset({ 
    required String deviceAssetId,
    required String fileName,
    required MultipartFile photo,
    DateTime? deviceCreatedAt,
    String? deviceTimeZone,
    MultipartFile? liveVideo,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/assets';
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
      contentType: 'multipart/form-data',
      validateStatus: validateStatus,
    );

    dynamic _bodyData;

    try {
      _bodyData = FormData.fromMap(<String, dynamic>{
        r'deviceAssetId': encodeFormParameter(_serializers, deviceAssetId, const FullType(String)),
        r'fileName': encodeFormParameter(_serializers, fileName, const FullType(String)),
        if (deviceCreatedAt != null) r'deviceCreatedAt': encodeFormParameter(_serializers, deviceCreatedAt, const FullType(DateTime)),
        if (deviceTimeZone != null) r'deviceTimeZone': encodeFormParameter(_serializers, deviceTimeZone, const FullType(String)),
        r'photo': photo,
        if (liveVideo != null) r'liveVideo': liveVideo,
      });

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

    AssetResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AssetResponse),
      ) as AssetResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AssetResponse>(
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
