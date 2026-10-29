//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/asset_response.dart';
import 'package:lumemor_api/src/model/error.dart';

class AssetsApi {

  final Dio _dio;

  final Serializers _serializers;

  const AssetsApi(this._dio, this._serializers);

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
