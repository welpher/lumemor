//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'dart:typed_data';
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/cluster_detail.dart';
import 'package:lumemor_api/src/model/cluster_face_response.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/face_create_request.dart';
import 'package:lumemor_api/src/model/face_created.dart';
import 'package:lumemor_api/src/model/label_person_request.dart';
import 'package:lumemor_api/src/model/merge_people_request.dart';
import 'package:lumemor_api/src/model/merge_people_response.dart';
import 'package:lumemor_api/src/model/move_face_request.dart';
import 'package:lumemor_api/src/model/move_face_response.dart';
import 'package:lumemor_api/src/model/person_avatar_response.dart';
import 'package:lumemor_api/src/model/person_birthdate_response.dart';
import 'package:lumemor_api/src/model/person_label_response.dart';
import 'package:lumemor_api/src/model/person_summary.dart';
import 'package:lumemor_api/src/model/set_person_avatar_request.dart';
import 'package:lumemor_api/src/model/set_person_birthdate_request.dart';
import 'package:lumemor_api/src/model/unnamed_clusters_response.dart';

class PeopleApi {

  final Dio _dio;

  final Serializers _serializers;

  const PeopleApi(this._dio, this._serializers);

  /// Seed a new person from one face.
  /// The from-face person creation: the face leaves whatever cluster it occupies and becomes the sole member AND the representative face of a brand-new cluster. Naming acts on a whole cluster, so a face grouped into the wrong person becomes its own person here -- the caller labels the new cluster afterwards like any other.  The refusal discipline is the face move&#39;s own: the face must sit on an asset the caller OWNS and that is live -- any other state (unknown face, another account&#39;s asset, an asset shared to the caller but not owned, a trashed, archived or Live Photo component asset) answers the same byte-identical 404 &#x60;PERSON_NOT_FOUND&#x60; (anti-enumeration). A face the clustering has not yet assigned to any cluster is VALID here: the creation seeds the new cluster with it and there is simply no source to maintain.  A call on a face that already holds a cluster is a CREATE, never a no-op: the face re-homes onto a new cluster and the vacated source follows the cluster maintenance discipline -- its representative re-homes to the best surviving member, or the cluster dissolves with its label rows cascading when this creation emptied it (the same honest cost a merge pays; a source cluster&#39;s user-given name dies with the cluster).  The whole structural change lands as ONE transaction. The write publishes &#x60;person.face_clustered&#x60; with the face id, the new cluster id and the acting user&#39;s id. 
  ///
  /// Parameters:
  /// * [faceId] - Identifier of the face that seeds the new cluster.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ClusterFaceResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ClusterFaceResponse>> clusterFace({ 
    required String faceId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/faces/{faceId}/cluster'.replaceAll('{' r'faceId' '}', encodeQueryParameter(_serializers, faceId, const FullType(String)).toString());
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

    ClusterFaceResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ClusterFaceResponse),
      ) as ClusterFaceResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ClusterFaceResponse>(
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

  /// Add one hand-drawn face to an asset.
  /// The manual face: the owner draws a box around a face the detector missed and the drawn box becomes a face row of that asset -- the same rows detection writes, minus a detector (the row&#39;s model is null, its score 1.0 -- the caller&#39;s own certainty) and minus an embedding (the recognition never saw it, so the face joins no cluster until the caller assigns one). The box rides in the FaceBbox per-mille units; the width and height halves must be at least 1 (a zero-extent box is nothing drawn).  The refusal discipline is the face writes&#39; own: the face must sit on an asset the caller OWNS and that is live -- any other state (unknown asset, another account&#39;s asset, an asset shared to the caller but not owned, a trashed, archived or Live Photo component asset) answers the same byte-identical 404 &#x60;PERSON_NOT_FOUND&#x60; (anti-enumeration). A bbox value outside the 0..1000 bounds, a missing half, or a body that fails to decode answers 400 &#x60;VALIDATION_FAILED&#x60;.  The write inserts the row and answers it; it publishes NO event: a face with no cluster announces no person change, and the caller&#39;s own session converges through the response (the same silence a fresh detection&#39;s rows keep until recognition names them). The caller&#39;s next step is the assign flow -- &#x60;POST /faces/{faceId}/move&#x60; or the from-face creation -- which is where the drawn face becomes someone. 
  ///
  /// Parameters:
  /// * [faceCreateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [FaceCreated] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<FaceCreated>> createFace({ 
    required FaceCreateRequest faceCreateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/faces';
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
      const _type = FullType(FaceCreateRequest);
      _bodyData = _serializers.serialize(faceCreateRequest, specifiedType: _type);

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

    FaceCreated? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(FaceCreated),
      ) as FaceCreated;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<FaceCreated>(
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

  /// Release one face from its person.
  /// The single-face detach: the face&#39;s cluster assignment goes NULL, returning it to the not-yet-grouped pool it came from (the state a freshly detected face holds before the clustering assigns it). The face row survives -- it still lists among its asset&#39;s faces, now carrying no person -- and any later assign write can re-home it.  The refusal discipline is the face move&#39;s own: the face must sit on an asset the caller OWNS and that is live -- any other state (unknown face, another account&#39;s asset, an asset shared to the caller but not owned, a trashed, archived or Live Photo component asset) answers the same byte-identical 404 &#x60;PERSON_NOT_FOUND&#x60; (anti-enumeration). A face the clustering has not yet assigned to any cluster is part of no person and answers the same 404: there is no membership to release.  The source cluster does NOT dissolve unless this detach emptied it; the vacated cluster re-homes its representative to the best surviving member, and an emptied cluster dissolves WITH its label rows cascading -- the same honest cost a merge pays. The whole structural change lands as ONE transaction. The write publishes &#x60;person.face_detached&#x60; with the face id, the cluster the face left and the acting user&#39;s id. 
  ///
  /// Parameters:
  /// * [faceId] - Identifier of the face to release.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> detachFace({ 
    required String faceId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/faces/{faceId}'.replaceAll('{' r'faceId' '}', encodeQueryParameter(_serializers, faceId, const FullType(String)).toString());
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

  /// Serve one face&#39;s 256x256 crop thumbnail.
  /// The face crop pixels of the people surface: a 256x256 &#x60;image/avif&#x60; crop of one detected face, derived at query time and never stored on disk. The source pixels are the face&#39;s asset&#39;s &#x60;view&#x60; derivative -- the inline serving size of &#x60;GET /assets/{id}/thumbnail&#x60; -- never the original; the face row&#39;s bounding box is in per-mille units of the full image, and the view derivative preserves the original&#39;s framing exactly, so the same per-mille values address the derivative&#39;s own pixels.  The crop window is pinned: a square whose side is 1.25x the larger of the box&#39;s pixel width and height (a 25% margin around the face), centered on the box and clamped to the image bounds (a face larger than the image yields the image&#39;s smaller edge); the window then renders to exactly 256x256 pixels -- upscaled when the window is smaller, a zoom rather than a fit.  The permission gate is the face&#39;s asset: the caller must see a live, non-component asset -- their own, or one shared to them through an album. Every other state answers the SAME byte-identical 404 PERSON_NOT_FOUND -- an unknown face id, a face on another account&#39;s unshared (or a trashed, or archived) asset, and an asset whose view derivative has not been derived yet (the thumbnail job has not run). Face ids cannot be probed for existence; the web renders its skeleton placeholder until the pixels exist, which is why there is deliberately no separate not-ready code here, unlike the asset thumbnail endpoint.  The response is a private immutable derived asset: &#x60;Cache-Control: private, max-age&#x3D;31536000, immutable&#x60;, and a strong &#x60;ETag&#x60; of &#x60;\&quot;face-{faceId}-{updatedAt}\&quot;&#x60;, where updatedAt is the face row&#39;s change signal in unix seconds -- cluster moves bump it, and re-detection replaces the face row outright. A matching &#x60;If-None-Match&#x60; answers 304 with the same validators and no body. Conditional requests and byte ranges otherwise follow the Go standard library&#39;s &#x60;ServeContent&#x60; semantics. 
  ///
  /// Parameters:
  /// * [faceId] - Identifier of the face to crop.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [Uint8List] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<Uint8List>> getFaceThumbnail({ 
    required String faceId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/faces/{faceId}/thumbnail'.replaceAll('{' r'faceId' '}', encodeQueryParameter(_serializers, faceId, const FullType(String)).toString());
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

  /// One face cluster&#39;s detail as the caller sees it.
  /// The detail view of one face cluster: the caller&#39;s own label state, the person entity&#39;s global attributes and the cluster&#39;s faces on assets visible to the caller, strongest score first with the face id as the tiebreak.  &#x60;label&#x60; is null when the caller has not labeled the cluster (the unnamed state). It is the CALLING user&#39;s label or no label at all: another account&#39;s label never shows here (clusters are global, labels are private per user).  &#x60;birthDate&#x60; and &#x60;avatarFaceId&#x60; are the PERSON ENTITY&#39;s attributes, not the caller&#39;s: one birthday per person, every account reading the same value (a family agrees on the person&#39;s birthday), and one avatar face per cluster whatever account picked it. Both are independent of naming -- an unnamed cluster carries them too.  &#x60;photoCount&#x60; is the count the detail bar renders: the number of DISTINCT live assets the cluster has a face on that the caller can see (photos, not faces).  404 has a double meaning by design: the cluster id is unknown, or the cluster exists but none of its faces sits on an asset visible to the caller (the cluster&#39;s assets all belong to other accounts and none is shared to them). Both answer the same &#x60;PERSON_NOT_FOUND&#x60;, byte for byte, so cluster ids cannot be probed for existence -- the same anti-enumeration shape as &#x60;ASSET_NOT_FOUND&#x60;.  Every &#x60;thumbnail&#x60; points at the face crop endpoint &#x60;GET /api/v1/faces/{faceId}/thumbnail&#x60;, which serves the face&#39;s bounding-box crop of the asset&#39;s view-size derivative. 
  ///
  /// Parameters:
  /// * [clusterId] - Identifier of the face cluster.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ClusterDetail] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ClusterDetail>> getPersonCluster({ 
    required String clusterId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people/{clusterId}'.replaceAll('{' r'clusterId' '}', encodeQueryParameter(_serializers, clusterId, const FullType(String)).toString());
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

    ClusterDetail? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ClusterDetail),
      ) as ClusterDetail;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ClusterDetail>(
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

  /// Name one face cluster (the caller&#39;s private label).
  /// Upserts the calling user&#39;s label of one face cluster: the &#x60;user_face_labels&#x60; row keyed (userId, clusterId) -- the per-user write. The cluster must be visible to the caller (at least one of its faces sits on an asset visible to them -- their own, or one shared to them through an album, so a sharee names the people they see); an unknown or invisible cluster id answers the same byte-identical 404 &#x60;PERSON_NOT_FOUND&#x60; as every people read (anti-enumeration -- cluster ids cannot be probed).  The name is trimmed and must span 1-64 characters; anything else answers 400 &#x60;VALIDATION_FAILED&#x60; naming the &#x60;name&#x60; field.  Duplicate names are ALLOWED, server-side and on purpose: no &#x60;NAME_TAKEN&#x60; error exists. Families have real duplicates (two grandparents named Grace), so a de-duplication hint is a web-UI suggestion, never a server rule.  Re-labeling an already labeled cluster renames it in place. The label is the caller&#39;s private state -- another account labeling the same cluster writes its own row and neither sees the other&#39;s -- and it is the ONLY private state of the people domain: the person entity&#39;s birthday and avatar ride the cluster&#39;s own columns, shared by every account.  The write publishes &#x60;person.labeled&#x60; on the activity stream with the acting user&#39;s id in the payload. 
  ///
  /// Parameters:
  /// * [clusterId] - Identifier of the face cluster.
  /// * [labelPersonRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [PersonLabelResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<PersonLabelResponse>> labelPerson({ 
    required String clusterId,
    required LabelPersonRequest labelPersonRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people/{clusterId}/label'.replaceAll('{' r'clusterId' '}', encodeQueryParameter(_serializers, clusterId, const FullType(String)).toString());
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
      const _type = FullType(LabelPersonRequest);
      _bodyData = _serializers.serialize(labelPersonRequest, specifiedType: _type);

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

    PersonLabelResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(PersonLabelResponse),
      ) as PersonLabelResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<PersonLabelResponse>(
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

  /// The caller&#39;s named people.
  /// The named region of the people surface: every face cluster the caller sees that carries one of the caller&#39;s own labels. A cluster is listed here when both hold:  - it contains at least one face on an asset visible to the   caller (an asset the caller can see that is live -- not   trashed, not archived, no Live Photo component row: their   own, or one shared to them through an album), and - the caller has labeled it with a &#x60;user_face_labels&#x60; row of   their own.  The per-user-state semantics of the people domain: face clusters are GLOBAL structures of the instance (the same person&#39;s faces land in one cluster whatever account uploaded the photos), while names are PRIVATE per user -- two accounts label the same cluster independently and neither sees the other&#39;s label. The people domain has exactly two states: named (this list) and unnamed (&#x60;GET /people/unnamed&#x60;). There is no third state: hiding a person retired whole (the per-user switch, its badge and its events are gone; removing the caller&#39;s name -- &#x60;DELETE /people/{clusterId}/label&#x60; -- is the way a person leaves this list).  &#x60;photoCount&#x60; is honest per caller: the number of DISTINCT live assets the cluster has a face on that the caller can see, so the same cluster can show different counts to different accounts. It counts photos, not faces -- two faces of one person in one photo are one photo.  The list is sorted by name, case-insensitively, with the cluster id as the tiebreak. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [BuiltList<PersonSummary>] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<BuiltList<PersonSummary>>> listPeople({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people';
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

    BuiltList<PersonSummary>? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(BuiltList, [FullType(PersonSummary)]),
      ) as BuiltList<PersonSummary>;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<BuiltList<PersonSummary>>(
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

  /// The caller&#39;s unnamed face clusters, one keyset page at a time.
  /// The unnamed region of the people surface: every face cluster that contains at least one face on an asset visible to the caller (an asset the caller can see that is live -- not trashed, not archived, no Live Photo component row: their own, or one shared to them through an album) and carries NO &#x60;user_face_labels&#x60; row of the caller. The moment the caller labels a cluster it moves to &#x60;GET /people&#x60; and out of this list; another account&#39;s label never moves it for this caller (clusters are global, labels are private per user).  Pages are keyset-paginated with an opaque cursor over the ordering &#x60;faceCount&#x60; descending, &#x60;clusterId&#x60; descending as the tiebreak. The continuation key is the (faceCount, clusterId) pair of a page&#39;s last row -- the same row-value predicate pattern the timeline endpoint applies to its (takenAt, id) pair, which keeps pages stable while clusters gain or lose faces. The response carries &#x60;nextCursor&#x60; exactly when another page exists; the first page omits the &#x60;cursor&#x60; parameter.  &#x60;totalCount&#x60; is the caller&#39;s whole unnamed region -- every cluster this listing would walk -- so a page never has to be walked to its end to count the region (the people list&#39;s unnamed entry badge and this page&#39;s own bar count read it).  &#x60;limit&#x60; bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 &#x60;VALIDATION_FAILED&#x60;. 
  ///
  /// Parameters:
  /// * [cursor] - The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
  /// * [limit] - Page size: the number of cluster summaries one response carries at most. Defaults to 60; values below 1 or over 200 answer 400. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [UnnamedClustersResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<UnnamedClustersResponse>> listUnnamedPeople({ 
    String? cursor,
    int? limit = 60,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people/unnamed';
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

    UnnamedClustersResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(UnnamedClustersResponse),
      ) as UnnamedClustersResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<UnnamedClustersResponse>(
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

  /// Fold 1-50 face clusters into one surviving cluster.
  /// One structural merge: every source cluster&#39;s faces re-home onto the target cluster, the target&#39;s representative face recomputes to the strongest-scoring member of the merged set, and every source cluster dissolves -- with its label rows cascading. A single source is the pairwise fold the cluster detail&#39;s \&quot;merge into\&quot; action drives (that person merges into the picked one); more sources are the batch organize path of the unnamed region.  THE HONEST COST, by design: a source cluster&#39;s user-given name DIES with the cluster, whatever account gave it. A person merged away cannot keep the name a caller gave the cluster they merged; the target cluster&#39;s label persists. The web UI names the target before merging when the caller wants the merged person called something.  The whole merge runs as ONE transaction through the cluster maintenance discipline (representative re-pointing plus dissolution of the vacated clusters -- the same primitives the asset deletion sweep applies), never as bare member moves that would strand a stale representative or an empty cluster.  Rules: every source and the target must be visible to the caller (at least one face on an asset the caller OWNS and that is live) -- merging is cluster curation, and curation is the owner&#39;s alone: a sharee who sees a cluster through a shared album reads it and labels it, but a merge from their hand answers the same 404 &#x60;PERSON_NOT_FOUND&#x60;, so the shared structure never re-shapes under another account&#39;s cursor. The target must not be among the sources -- any violation answers the same byte-identical 404 &#x60;PERSON_NOT_FOUND&#x60; (anti-enumeration). An empty source list or more than 50 source ids (duplicates collapse first) answer 400 &#x60;VALIDATION_FAILED&#x60;.  The write publishes one &#x60;person.merged&#x60; event with the surviving cluster id, the dissolved ids comma-joined as &#x60;mergedClusterIds&#x60;, and the acting user&#39;s id. 
  ///
  /// Parameters:
  /// * [mergePeopleRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [MergePeopleResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<MergePeopleResponse>> mergePeople({ 
    required MergePeopleRequest mergePeopleRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people/merge';
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
      const _type = FullType(MergePeopleRequest);
      _bodyData = _serializers.serialize(mergePeopleRequest, specifiedType: _type);

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

    MergePeopleResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(MergePeopleResponse),
      ) as MergePeopleResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<MergePeopleResponse>(
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

  /// Move one face onto an existing face cluster.
  /// One face&#39;s move onto an existing target cluster: the tidy-up primitive behind the viewer&#39;s face chips. The face must sit on an asset the caller OWNS and that is live, and the target must be caller-visible -- any other state (unknown face, another account&#39;s asset, an asset shared to the caller but not owned -- the move is curation, and curation is the owner&#39;s alone -- an invisible or unknown target) answers the same byte-identical 404 &#x60;PERSON_NOT_FOUND&#x60; (anti-enumeration). A face the clustering has not yet assigned to any cluster is a valid source: it simply joins the target (there is no source cluster to maintain).  The move runs as ONE transaction through the cluster maintenance discipline: the target&#39;s representative face recomputes to its strongest member, and the vacated source cluster re-homes its representative to the best surviving member -- or dissolves with its label rows cascading when the move empties it (a source cluster&#39;s user-given name dies with the cluster, the same honest cost as a merge).  Moving a face onto the cluster it already occupies is an accepted no-op. The write publishes &#x60;person.face_moved&#x60; with the face id, the cluster it now occupies and the acting user&#39;s id. 
  ///
  /// Parameters:
  /// * [faceId] - Identifier of the face to move.
  /// * [moveFaceRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [MoveFaceResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<MoveFaceResponse>> moveFace({ 
    required String faceId,
    required MoveFaceRequest moveFaceRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/faces/{faceId}/move'.replaceAll('{' r'faceId' '}', encodeQueryParameter(_serializers, faceId, const FullType(String)).toString());
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
      const _type = FullType(MoveFaceRequest);
      _bodyData = _serializers.serialize(moveFaceRequest, specifiedType: _type);

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

    MoveFaceResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(MoveFaceResponse),
      ) as MoveFaceResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<MoveFaceResponse>(
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

  /// Pick the face one person&#39;s avatar renders.
  /// Writes the PERSON ENTITY&#39;s avatar face: the cluster&#39;s chosen face, read by every account (the person card on &#x60;GET /people&#x60; and the detail&#39;s profile row both render it). Like the birth date it is not the caller&#39;s private state, and it does not depend on naming -- an unnamed cluster carries an avatar too.  The face must belong to the cluster and sit on an asset visible to the caller (the avatar picker offers exactly the cluster&#39;s caller-visible faces); anything else -- a face of another cluster, an invisible face, an unknown cluster -- answers the same byte-identical 404 &#x60;PERSON_NOT_FOUND&#x60; as every people write (anti-enumeration).  The pick is advisory to the clustering, never a structural write: the representative face the clustering maintains for merges and face moves is untouched, and a face that later leaves the cluster (a move, a detach, a purged asset) simply stops counting -- the read falls back to the cluster&#39;s best-scoring caller-visible face.  The write publishes &#x60;person.avatar_updated&#x60; on the activity stream with the acting user&#39;s id in the payload. 
  ///
  /// Parameters:
  /// * [clusterId] - Identifier of the face cluster.
  /// * [setPersonAvatarRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [PersonAvatarResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<PersonAvatarResponse>> setPersonAvatar({ 
    required String clusterId,
    required SetPersonAvatarRequest setPersonAvatarRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people/{clusterId}/avatar'.replaceAll('{' r'clusterId' '}', encodeQueryParameter(_serializers, clusterId, const FullType(String)).toString());
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
      const _type = FullType(SetPersonAvatarRequest);
      _bodyData = _serializers.serialize(setPersonAvatarRequest, specifiedType: _type);

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

    PersonAvatarResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(PersonAvatarResponse),
      ) as PersonAvatarResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<PersonAvatarResponse>(
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

  /// Set or clear one person&#39;s birth date.
  /// Writes the PERSON ENTITY&#39;s birth date: one value per cluster, read by every account (&#x60;GET /people/{clusterId}&#x60; serves it to all of them). It is not the caller&#39;s private state -- a family agrees on the person&#39;s birthday -- and it is independent of naming: an unnamed cluster carries a birth date too, and &#x60;person_birthdate&#x60; (the profile row) renders it as soon as it is set.  The use of the value is honest: the person&#39;s age at the time each photo was taken, and the birthday memories the product will build later. Nothing else reads it.  Any logged-in user who can see the cluster may write it (at least one of the cluster&#39;s faces sits on an asset visible to them); an unknown or invisible cluster id answers the same byte-identical 404 &#x60;PERSON_NOT_FOUND&#x60; as every people read (anti-enumeration -- cluster ids cannot be probed).  &#x60;birthDate&#x60; is the wire&#39;s &#x60;date&#x60; form (&#x60;YYYY-MM-DD&#x60;): a null clears the value. A malformed date answers 400 &#x60;VALIDATION_FAILED&#x60; naming the &#x60;birthDate&#x60; field.  The write publishes &#x60;person.birthdate_updated&#x60; on the activity stream with the acting user&#39;s id in the payload (the person entity&#39;s attributes are shared state, so every open view of the person converges). 
  ///
  /// Parameters:
  /// * [clusterId] - Identifier of the face cluster.
  /// * [setPersonBirthdateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [PersonBirthdateResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<PersonBirthdateResponse>> setPersonBirthdate({ 
    required String clusterId,
    required SetPersonBirthdateRequest setPersonBirthdateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people/{clusterId}/birthdate'.replaceAll('{' r'clusterId' '}', encodeQueryParameter(_serializers, clusterId, const FullType(String)).toString());
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
      const _type = FullType(SetPersonBirthdateRequest);
      _bodyData = _serializers.serialize(setPersonBirthdateRequest, specifiedType: _type);

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

    PersonBirthdateResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(PersonBirthdateResponse),
      ) as PersonBirthdateResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<PersonBirthdateResponse>(
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

  /// Remove the caller&#39;s label of one face cluster.
  /// Deletes the calling user&#39;s &#x60;user_face_labels&#x60; row of one cluster: the person returns to the caller&#39;s unnamed region (&#x60;GET /people/unnamed&#x60;) and out of the caller&#39;s named list. The cluster itself and every other account&#39;s label survive (labels are private per user; only the caller&#39;s own row dies).  The write is idempotent once the cluster is visible: naming no label still answers 204. The cluster must be visible to the caller -- an unknown or invisible cluster id answers the same byte-identical 404 &#x60;PERSON_NOT_FOUND&#x60; as every people read (anti-enumeration).  The write publishes &#x60;person.unlabeled&#x60; on the activity stream with the acting user&#39;s id in the payload. 
  ///
  /// Parameters:
  /// * [clusterId] - Identifier of the face cluster.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> unlabelPerson({ 
    required String clusterId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/people/{clusterId}/label'.replaceAll('{' r'clusterId' '}', encodeQueryParameter(_serializers, clusterId, const FullType(String)).toString());
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

}
