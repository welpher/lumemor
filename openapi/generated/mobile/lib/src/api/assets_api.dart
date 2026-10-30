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
import 'package:lumemor_api/src/model/asset_response.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/timeline_response.dart';

class AssetsApi {

  final Dio _dio;

  final Serializers _serializers;

  const AssetsApi(this._dio, this._serializers);

  /// Fetch the detail of one owned asset.
  /// The asset detail: the identity and storage fields of the upload response plus the refinements the pipeline produced afterwards -- dimensions, duration, blurhash, the EXIF hot fields, and readiness -- which the asset detail view renders.  Access is owner-only: an id that belongs to another account answers the same 404 &#x60;ASSET_NOT_FOUND&#x60; as an id that does not exist, byte for byte, so asset ids cannot be enumerated. Album sharing and ACLs widen access for shared content in a later milestone (M2c, with the album domain); until then only the owner&#39;s own assets answer.  A trashed asset answers 200 with &#x60;isTrashed: true&#x60; -- the trash view reads the detail the same way. The component video row of a Live Photo is readable by its id (the future playback endpoint resolves it through &#x60;livePhotoVideoId&#x60;) but never appears in the timeline. 
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
  /// Streams the stored original bytes as a download. It is available immediately after the upload: the original is the uploaded content-addressed file itself, not a pipeline product.  The download is always an attachment: &#x60;Content-Disposition: attachment; filename*&#x3D;UTF-8&#39;&#39;...&#x60; with the original file name escaped per RFC 5987 -- every byte outside the attribute character set is percent-encoded, which rules out header injection and in-browser rendering alike (SVG is not in the upload whitelist, so the download-only rule covers every stored format). &#x60;Content-Type&#x60; is the asset&#39;s stored MIME type. The response carries the same &#x60;ETag&#x60; and &#x60;Cache-Control&#x60; validators as the thumbnail (the original bytes are content-addressed, hence immutable).  Byte ranges follow the Go standard library&#39;s &#x60;ServeContent&#x60; semantics: a satisfiable single range answers 206 with &#x60;Content-Range&#x60;; multiple ranges answer a &#x60;multipart/byteranges&#x60; body; an unsatisfiable range answers 416 with the total size in &#x60;Content-Range&#x60;; a malformed Range header is ignored and the full body answers 200.  The same owner-only download gate as every asset endpoint: foreign and unknown ids answer byte-identical 404 &#x60;ASSET_NOT_FOUND&#x60;. 
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
  /// The static serving copy of the asset:  - image assets: the AVIF thumbnail of the requested size --   &#x60;tile&#x60; (fits the 250px timeline box), &#x60;view&#x60; (long edge   capped at 1080px), &#x60;original&#x60; (the full-resolution   re-encode) -- served as &#x60;image/avif&#x60;; - video assets: the poster frame, whatever &#x60;size&#x60; says -- the   poster (the first frame, scaled into the tile box) is the   video&#39;s only serving copy until transcodes exist -- served   as &#x60;image/jpeg&#x60;.  The response is a private immutable static asset: &#x60;Cache-Control: private, max-age&#x3D;31536000, immutable&#x60;, and a strong &#x60;ETag&#x60; built from the first 16 hex characters of the asset&#39;s checksum (quoted). A matching &#x60;If-None-Match&#x60; answers 304 with the same validators and no body. Conditional requests and byte ranges otherwise follow the Go standard library&#39;s &#x60;ServeContent&#x60; semantics.  The asset exists but its derived file is absent -- the thumbnail job has not run yet, or failed and is retrying -- answers 404 &#x60;ASSET_NOT_READY&#x60;: the client renders the blurhash or a placeholder and waits for the &#x60;asset.ready&#x60; event instead of treating the asset as gone. &#x60;ASSET_NOT_FOUND&#x60;, by contrast, means the id is unknown or not the caller&#39;s; the owner-only anti-enumeration of the detail endpoint applies here too, with byte-identical 404 bodies for foreign and unknown ids.  &#x60;size&#x60; must be one of the three values below; anything else (including a missing &#x60;size&#x60;) answers 400 &#x60;VALIDATION_FAILED&#x60;. 
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

  /// The aggregated timeline, one keyset page at a time.
  /// The aggregated timeline of the caller: the union of the albums that feed it -- the caller&#39;s sync album (the core backup stream, never excludable), the caller&#39;s own directory albums, and every album shared to the caller regardless of type (virtual albums included) -- minus the albums the caller turned off with the per-user \&quot;keep out of the timeline\&quot; preference (a missing preference row means kept in). The caller&#39;s own virtual albums never feed their own timeline: in-stream curation would be pure duplication. One asset visible through several albums appears once.  Every item is live by construction: hidden assets, assets in the trash, and the component video rows of Live Photos never appear.  Pages are keyset-paginated with an opaque cursor: the response carries &#x60;nextCursor&#x60; exactly when another page exists, and the client passes it back as &#x60;cursor&#x60; to continue after the last item of the page. Cursors are stable for the ordering key (&#x60;takenAt&#x60; descending, &#x60;id&#x60; as the tiebreak); an asset uploaded with an older &#x60;takenAt&#x60; than the cursor appears only in a fresh pagination run, never in the current one. The first page omits the &#x60;cursor&#x60; parameter.  &#x60;limit&#x60; bounds the page size: absent means the default of 100, a value over 200 is answered with the maximum of 200, and a value below 1 answers 400 &#x60;VALIDATION_FAILED&#x60;.  &#x60;albumId&#x60; narrows the response to that album&#39;s own timeline instead of the aggregate: the album&#39;s items in the same ordering and filtering. The caller must be able to see the album (own album of any type, or an album shared to the caller); an album that does not exist or is not visible answers the empty list, not an error, so album ids cannot be probed for existence through this endpoint. 
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

  /// Upload one photo, optionally with its Live Photo video.
  /// The backup upload endpoint: one photo per request, streamed into content-addressed storage while it is read, with the Live Photo pairing video riding the same request as a second file part.  The request body is &#x60;multipart/form-data&#x60; with these parts:  - &#x60;deviceAssetId&#x60; (text, required) -- the stable identity the   uploading device assigned to the file; together with the   account it scopes the device idempotency of backups. - &#x60;fileName&#x60; (text, required) -- the original client-side file   name of the photo, recorded as the asset&#39;s   &#x60;originalFilename&#x60;. - &#x60;deviceCreatedAt&#x60; (text, optional) -- the capture time the   device reports, RFC 3339 with offset. It seeds the asset&#39;s   &#x60;takenAt&#x60; until the metadata pipeline refines it from EXIF. - &#x60;deviceTimeZone&#x60; (text, optional) -- the IANA time zone name   of the capture location; it feeds the metadata pipeline&#39;s   timezone fallback chain and is not validated at upload. - &#x60;photo&#x60; (file, required) -- the photo bytes. - &#x60;liveVideo&#x60; (file, optional) -- the pairing video of a Live   Photo; present only together with &#x60;photo&#x60;.  Part order is part of the contract: all text fields precede every file part, and the &#x60;liveVideo&#x60; part follows the &#x60;photo&#x60; part. The server streams file parts to disk without buffering them, so every field must be known and validated before the first file byte is accepted -- a file part arriving before the required fields, a text field after a file part, or a &#x60;liveVideo&#x60; without a preceding &#x60;photo&#x60; answers 400 &#x60;VALIDATION_FAILED&#x60;. Each text field value is capped at 1024 bytes; &#x60;deviceAssetId&#x60;, &#x60;fileName&#x60; and &#x60;deviceTimeZone&#x60; carry the length bounds below. The storage extension and the media kind derive from the *file part&#39;s own* file name, and the payload must be consistent with it: a renamed file of another type answers 400 &#x60;VALIDATION_FAILED&#x60;.  Uploads are idempotent through the owner-scoped checksum: re-uploading identical bytes never creates a second asset. The outcome is one of three states, discriminated by the &#x60;state&#x60; field of the response body:  - 201 &#x60;created&#x60;: no asset with this checksum existed; a new   asset was created (a Live Photo created two rows, the   response describes the photo). - 200 &#x60;duplicate&#x60;: an identical asset of the same owner is   live; that asset is returned. - 200 &#x60;trashed&#x60;: an identical asset of the same owner sits in   the trash; that asset is returned (with &#x60;isTrashed: true&#x60;)   so the client can offer the user restore or save-as-new.  Each file is capped at the instance&#39;s per-file upload limit (2 GiB by default); a file over the limit aborts mid-stream with 413 &#x60;ASSET_TOO_LARGE&#x60;. This endpoint takes multipart bodies, not JSON, and is deliberately exempt from the API-wide JSON body cap; the per-file upload limit governs instead. 
  ///
  /// Parameters:
  /// * [deviceAssetId] - Stable device-side identity of the uploaded file.
  /// * [fileName] - Original client-side file name of the photo.
  /// * [photo] - The photo bytes; their file part name carries the extension.
  /// * [deviceCreatedAt] - Capture time reported by the device (RFC 3339 with offset). Optional; when absent the server records the upload time until the metadata pipeline refines it. 
  /// * [deviceTimeZone] - IANA time zone name of the capture location, for the metadata pipeline's timezone fallback chain. Not validated at upload time. 
  /// * [liveVideo] - The Live Photo pairing video; present only together with `photo`. The server stores it as a linked component row that never appears in the timeline. 
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
