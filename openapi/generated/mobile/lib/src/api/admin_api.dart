//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

import 'dart:async';

import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:dio/dio.dart';

import 'package:built_collection/built_collection.dart';
import 'package:built_value/json_object.dart';
import 'package:lumemor_api/src/api_util.dart';
import 'package:lumemor_api/src/model/admin_create_user_request.dart';
import 'package:lumemor_api/src/model/admin_user_delete_request.dart';
import 'package:lumemor_api/src/model/admin_user_list_response.dart';
import 'package:lumemor_api/src/model/admin_user_patch.dart';
import 'package:lumemor_api/src/model/admin_user_response.dart';
import 'package:lumemor_api/src/model/delete_model200_response.dart';
import 'package:lumemor_api/src/model/derivatives_summary_response.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/image_settings_response.dart';
import 'package:lumemor_api/src/model/image_settings_update_request.dart';
import 'package:lumemor_api/src/model/job_run_list_response.dart';
import 'package:lumemor_api/src/model/model_activate_response.dart';
import 'package:lumemor_api/src/model/model_download_response.dart';
import 'package:lumemor_api/src/model/model_list_response.dart';
import 'package:lumemor_api/src/model/model_probe_accepted_response.dart';
import 'package:lumemor_api/src/model/model_probe_response.dart';
import 'package:lumemor_api/src/model/model_rerun_response.dart';
import 'package:lumemor_api/src/model/model_scan_response.dart';
import 'package:lumemor_api/src/model/model_settings_response.dart';
import 'package:lumemor_api/src/model/model_settings_update_request.dart';
import 'package:lumemor_api/src/model/model_slot.dart';
import 'package:lumemor_api/src/model/model_source_switch_request.dart';
import 'package:lumemor_api/src/model/model_source_switch_response.dart';
import 'package:lumemor_api/src/model/places_rerun_response.dart';
import 'package:lumemor_api/src/model/rebuild_derivatives_request.dart';
import 'package:lumemor_api/src/model/rebuild_derivatives_response.dart';
import 'package:lumemor_api/src/model/run_budget_settings_response.dart';
import 'package:lumemor_api/src/model/run_budget_settings_update_request.dart';
import 'package:lumemor_api/src/model/storage_settings_response.dart';
import 'package:lumemor_api/src/model/storage_settings_update_request.dart';
import 'package:lumemor_api/src/model/tiles_settings_response.dart';
import 'package:lumemor_api/src/model/tiles_settings_update_request.dart';
import 'package:lumemor_api/src/model/transcoding_settings_response.dart';
import 'package:lumemor_api/src/model/transcoding_settings_update_request.dart';
import 'package:lumemor_api/src/model/trash_settings_response.dart';
import 'package:lumemor_api/src/model/trash_settings_update_request.dart';
import 'package:lumemor_api/src/model/update_check_response.dart';
import 'package:lumemor_api/src/model/update_check_settings_response.dart';
import 'package:lumemor_api/src/model/update_check_settings_update_request.dart';

class AdminApi {

  final Dio _dio;

  final Serializers _serializers;

  const AdminApi(this._dio, this._serializers);

  /// Activate the model (the model switch).
  /// The activation switch of the admin surface: the model becomes the active one of its slot. The request body is an empty JSON object -- the path&#39;s model identifier is the whole request.  A same-slot switch runs in one transaction that only flips the activation flags: the previous model&#39;s products stay in the database untouched (multi-generation embeddings and faces coexist by their model columns), queries that filter on the active model stop seeing them, and switching back revives them with zero recompute -- but the newly activated model does nothing until its products are rerun.  Cross-dimension switches of the clip-embed slot are the one exception, because a vector column cannot hold two dimensions: activating a clip model whose dimension differs from the &#x60;clip_embeddings&#x60; column&#39;s current one (read from the database catalog, never assumed) clears the table and re-types the column to &#x60;vector(N)&#x60; inside the SAME transaction as the flag flip -- the wipe, the new dimension and the activation land together or not at all, and a failure rolls everything back. The response then reports the switch&#39;s facts (&#x60;dimensionChanged&#x60; with the from and to dimensions, &#x60;wipedEmbeddings&#x60; with the number of deleted rows) so the admin UI can demand the explicit confirmation a destructive switch deserves. The rerun that fills the new model&#39;s embeddings stays manual: nothing is recomputed here.  The model&#39;s files are verified against their pinned sha256 digests before anything flips: a model whose files are missing or drifted answers 400 MODEL_INVALID and activates nothing. 
  ///
  /// Parameters:
  /// * [id] - The identifier of the model to activate (the models table's slug id, as the listing returns it). 
  /// * [requestBody] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelActivateResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelActivateResponse>> activateModel({ 
    required String id,
    required BuiltMap<String, JsonObject?> requestBody,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/models/{id}'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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
      const _type = FullType(BuiltMap, [FullType(String), FullType(JsonObject)]);
      _bodyData = _serializers.serialize(requestBody, specifiedType: _type);

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

    ModelActivateResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelActivateResponse),
      ) as ModelActivateResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelActivateResponse>(
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

  /// Cancel one in-flight job run.
  /// The task panel&#39;s cancel (E41): asks the named run to end at its next chunk boundary. The request itself flags the run only -- the run&#39;s driver checks the flag BETWEEN chunks (a walk page, a rescan batch, one model of a download group) and closes the run &#x60;cancelled&#x60; there, so:  - The committed portion stays: every chunk already finished keeps its work on disk and its count in the run&#39;s &#x60;done&#x60;. The response makes no promise about work that has not committed yet. - An in-flight asset-level job (one decode, one inference, one encode) runs to its own end -- the boundary is between chunks, never mid-asset. - The run&#39;s state flips to &#x60;cancelled&#x60; when the driver closes it, and the close rides the task panel&#39;s existing events (&#x60;job.finished&#x60;) -- the listing converges through the same stream it always converges through. A QUEUED run cancels at once (its driver checks the flag before the first chunk). - A run that already finished -- succeeded, failed or cancelled -- answers 409 &#x60;JOB_NOT_CANCELLABLE&#x60;: nothing is in flight to cancel, and the row&#39;s own state is the honest answer. An unknown run id answers 404 &#x60;JOB_RUN_NOT_FOUND&#x60;.  The cancel takes no request body and answers no body: the 204 is the flag&#39;s whole receipt. 
  ///
  /// Parameters:
  /// * [runId] - The `job_runs` row identifier of the run to cancel.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> cancelJobRun({ 
    required String runId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/jobs/{runId}/cancel'.replaceAll('{' r'runId' '}', encodeQueryParameter(_serializers, runId, const FullType(String)).toString());
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

  /// Create one account.
  /// The account creation of the admin surface, the management counterpart of the first-run wizard: an administrator provisions another account -- a family member&#39;s login, or a second administrator. The account rules mirror the wizard&#39;s exactly: the email is parsed and lower-cased as the login identity, the password is stored only as an Argon2id hash and never appears in any response, the id is a fresh UUIDv7, and the account&#39;s sync album -- the core backup stream every upload lands in and the timeline aggregates -- is created in the SAME transaction, so the new account can take uploads the moment it exists (a second creation point of the same wiring the wizard uses). The wizard&#39;s one-time guarantee is NOT part of this endpoint; it exists exactly because an administrator already does.  The role defaults to &#x60;user&#x60; when omitted; creating another administrator is the same request with &#x60;role: \&quot;admin\&quot;&#x60;. A duplicate email answers 409 USER_EMAIL_TAKEN -- the address is the account identity, unique instance-wide, and the login lookup depends on that uniqueness. 
  ///
  /// Parameters:
  /// * [adminCreateUserRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AdminUserResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AdminUserResponse>> createUser({ 
    required AdminCreateUserRequest adminCreateUserRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/users';
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
      const _type = FullType(AdminCreateUserRequest);
      _bodyData = _serializers.serialize(adminCreateUserRequest, specifiedType: _type);

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

    AdminUserResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AdminUserResponse),
      ) as AdminUserResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AdminUserResponse>(
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

  /// Delete one registered model.
  /// The removal of a model registration: the row is deleted and its model directory (weights plus manifest copy) is removed from the models dir. Only a model that is NOT its slot&#39;s active one can be deleted -- the active model of a slot is the pipeline&#39;s running configuration, and the removal of it is a switch first (the dialog&#39;s removal flow switches the slot back onto the official entry, then removes).  A rerun or other job still holding the model&#39;s products is unaffected: the products reference the model by id and the registry&#39;s foreign keys keep them resolvable, so a delete of a model whose products still exist answers 200 and leaves the products to their own lifecycle.  The model&#39;s files are removed from disk after the row is deleted; a directory removal failure logs and still answers 200 (the registration is gone; the leftover directory is cleaned by the next scan&#39;s refusal). 
  ///
  /// Parameters:
  /// * [id] - The identifier of the model to delete.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [DeleteModel200Response] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<DeleteModel200Response>> deleteModel({ 
    required String id,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/models/{id}'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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

    DeleteModel200Response? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(DeleteModel200Response),
      ) as DeleteModel200Response;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<DeleteModel200Response>(
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

  /// Delete one account, disposing of its content.
  /// The account disposal of the admin surface: an account leaves the instance, and everything it held -- photos, albums, memories and the labels it gave them -- is settled before the row goes. The account row goes immediately: there is no soft delete and no recovery window for the account itself. The whole settlement is ONE transaction that commits whole or rolls back whole, so a failure leaves the account and its content exactly as they were.  THE DISPOSITION is exactly one of two arms:  - &#x60;transferTo&#x60; names another account that receives everything   the departing account owned: its photos with their albums   and memories, their archived and trash marks intact, and the   tags attached to them. The photos join the receiver&#39;s own   stream, so the family timeline continues without a gap, and   the albums the departing account shared keep their grantees.    COLLIDING CONTENT. The storage&#39;s uniqueness rule is   owner plus content checksum, and it is TOTAL: one account   holds at most one row of a given content whether that row is   live or in the trash. Content the receiver already holds --   in ANY of the four state pairs -- therefore cannot simply   change hands, and the disposal folds it instead:    - the LIVE copy wins a mixed pair, so content that arrives     live stays visible under the receiver&#39;s name, and a copy     the receiver had trashed earlier does not bind it (trash it     again to be rid of it) -- symmetrically, the receiver&#39;s     live copy survives a trashed departing one;   - a tie keeps the RECEIVER&#39;s own row: both live keeps its     copy, both trashed keeps it under the LATER of the two     deletion marks (the content stays recoverable as long as     either copy had left);   - the losing copy&#39;s album memberships, tags, favorites and     any album cover it pinned move onto the survivor, then the     losing row goes. A losing Live Photo takes its component     video row with it (the pair is one asset), and every losing     photo announces &#x60;asset.deleted&#x60;, so a client holding the id     lets it go.  - &#x60;dispose: \&quot;trash\&quot;&#x60; walks the departing account&#39;s photos into   the CALLING administrator&#39;s trash, where the instance&#39;s   retention window governs them -- restorable until it runs   out, permanently removed after. The departing account&#39;s own   albums and memories go with the account (a container whose   owner is gone has no reader), so the photos survive as   individual trash items; a directory album&#39;s source file is   moved into the server&#39;s trash area on this arm exactly like   any other trash move, and the rest of the family&#39;s source   files are never touched. The collision rule above applies   here too, onto the calling administrator: content they   already hold in any state folds rather than collides, and   their own live copy is never trashed by someone else&#39;s   disposal.  What dies with the account in BOTH arms: its sessions, API keys and pairing tokens (it can never authenticate again), its favorites, its per-album timeline preferences, the private names it gave shared face clusters (a cluster belongs to the family; the name a person gave it is theirs), and every album share granted TO it. A share the account granted on its own album rides the content: the album changes hands and its grantees keep their access. Authorship another member&#39;s album carries -- a membership the departing account added -- passes to the surviving owner instead of deleting the membership, so no photo leaves an album it sits in.  THE REFUSALS. The instance&#39;s last administrator cannot be deleted (409 USER_LAST_ADMIN): an instance without an administrator strands this surface, and while one administrator remains that account is necessarily the caller&#39;s own row. An administrator cannot delete their own account at all (400 VALIDATION_FAILED naming the id) -- the disposal hands the content to a survivor, and the caller is the one performing the handover. &#x60;transferTo&#x60; naming the departing account itself answers 409 USER_TRANSFER_TO_SELF, and an unknown receiver is a payload error (400 VALIDATION_FAILED naming the field). Account ids are enumerable through this surface&#39;s own listing, so an unknown path id answers 404 USER_NOT_FOUND without an anti-enumeration pairing.  The 204 carries no body: the account is gone, and the caller&#39;s own disposal target (the receiver&#39;s stream or the caller&#39;s trash) is where its content answers next. 
  ///
  /// Parameters:
  /// * [id] - Identifier of the account to delete.
  /// * [adminUserDeleteRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future]
  /// Throws [DioException] if API call or serialization fails
  Future<Response<void>> deleteUser({ 
    required String id,
    required AdminUserDeleteRequest adminUserDeleteRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/users/{id}'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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
      const _type = FullType(AdminUserDeleteRequest);
      _bodyData = _serializers.serialize(adminUserDeleteRequest, specifiedType: _type);

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

    return _response;
  }

  /// Start the default model group&#39;s download.
  /// The model download trigger of the admin surface: the default group&#39;s models -- pinned by the manifest with upstream URLs and sha256 digests -- are downloaded, verified and registered, and each model activates into its slot when the slot is still empty. The request body is deliberately empty: the default group is the only group v1 downloads.  The endpoint is idempotent. A run that is still in flight (state &#x60;queued&#x60; or &#x60;running&#x60;) answers 200 with its identifier and nothing new is started; so does the most recent finished run -- a repeated click neither re-downloads a landed group nor piles up jobs, and the recovery of a failed run belongs to the job&#39;s own retries. Only a first call -- no run of this kind at all -- creates one and answers 202.  The run&#39;s progress is observable two ways: the &#x60;job_runs&#x60; row of the returned identifier (kind &#x60;model_download&#x60;, &#x60;total&#x60; and &#x60;done&#x60; carrying the group&#39;s pinned byte counts) and the &#x60;job.started&#x60; / &#x60;job.progress&#x60; / &#x60;job.finished&#x60; events on the activity stream, the progress events throttled to at most one per second. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelDownloadResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelDownloadResponse>> downloadModels({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/models/download';
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

    ModelDownloadResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelDownloadResponse),
      ) as ModelDownloadResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelDownloadResponse>(
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

  /// Read the derivative library&#39;s size and rebuild estimate.
  /// The image card&#39;s cost line: how many image assets the library carries and how long a full derivative rebuild would take, as the instance itself estimates it.  THE ESTIMATE IS DECLARED, NOT MEASURED: the server multiplies the image count by one second per asset (one decode plus the tile and view encodes at the shipped settings), divides by the concurrency the run-budget dial currently maps, and rounds up to whole minutes, with one minute the floor. The answer is the plan-of-record number the dialog quotes back before the rebuild starts, never a promise: NAS-class storage, the encoding format and the dial all move it.  The count covers IMAGE assets only: the rebuild re-derives the three serving sizes of images; video assets carry a poster frame the rebuild does not touch. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [DerivativesSummaryResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<DerivativesSummaryResponse>> getDerivativesSummary({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/derivatives/summary';
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

    DerivativesSummaryResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(DerivativesSummaryResponse),
      ) as DerivativesSummaryResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<DerivativesSummaryResponse>(
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

  /// Read the derivative image settings.
  /// The image card&#39;s document: the sizes and encoding every NEW derivation runs at. Five knobs, the honest set --  - &#x60;tileSize&#x60;: the timeline tile&#39;s bounding box, 250 (the   shipped default) or 320 (sharper on high-density screens). - &#x60;previewSize&#x60;: the viewer&#39;s inline preview long edge, 1080   (the shipped default), 1440 or 2048. - &#x60;format&#x60;: the serving copies&#39; encoding, &#x60;avif&#x60; (smaller,   slower to encode) or &#x60;webp&#x60; (faster, a little larger). On a   weak machine AVIF makes first browsing slower; the card says   so beside the knob. - &#x60;fullSize&#x60;: whether the thumbnail job also derives the   full-resolution serving copy (shipped OFF: the viewer reads   the preview, which is enough to look at and cheaper to   store). - &#x60;rawEmbedded&#x60;: whether a RAW source derives from its   embedded preview (shipped ON).  THE COST STATEMENT IS THE CARD&#39;S, NOT THIS DOCUMENT&#39;S: saving these values recomputes nothing. Derivatives already on disk stay exactly as they are; only assets derived AFTER the save (new uploads, and the explicit rebuild run) use the new values. A rebuild mid-flight keeps the parameters it started with.  The color space (P3), progressive encoding and the per-size quality are deliberately NOT knobs: a family instance does not need twenty dials.  A corrupt settings row answers the defaults with the corruption logged server-side. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ImageSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ImageSettingsResponse>> getImageSettings({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/image';
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

    ImageSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ImageSettingsResponse),
      ) as ImageSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ImageSettingsResponse>(
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

  /// Read one model source probe&#39;s result.
  /// The probe the switch dialog polls. A source probe -- the download-to-staging of a URL, or the accept of an uploaded file, then the sha256 check when one was given, the sandbox load of the ONNX graph and the slot contract&#39;s shape and dimension checks -- runs in the server&#39;s background after the probe POST answered 202; this read answers the probe&#39;s current state, whatever it is: running (with its download progress), ok (with the facts the dialog renders), error (with the reason that names the exact check that failed) or warn (the checks passed, the host is not on the download sources list, and the save may still proceed).  A probe is transient: its record lives for a few minutes after it settles, then it is gone -- an unknown or expired probeId answers 404 and the dialog simply probes again. 
  ///
  /// Parameters:
  /// * [probeId] - The probe identifier the POST answered.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelProbeResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelProbeResponse>> getModelProbe({ 
    required String probeId,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/models/probes/{probeId}'.replaceAll('{' r'probeId' '}', encodeQueryParameter(_serializers, probeId, const FullType(String)).toString());
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

    ModelProbeResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelProbeResponse),
      ) as ModelProbeResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelProbeResponse>(
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

  /// Read the model download settings.
  /// The model store&#39;s policy switch, download sources and pipeline params: whether an AI job whose slot is configured but whose weights are missing may start the model group&#39;s download on its own, where the bytes may travel, and the two steering params the ML jobs read at their own dispatch (see &#x60;ModelParams&#x60;).  WHAT AUTO-FILLS: only a slot that already has an active model whose files are gone from the store -- the case of a store wiped below a configured instance. A slot that was never configured stays empty; the first-run download and the settings face&#39;s download button remain the administrator&#39;s active path, and an unconfigured slot keeps the honest no-AI degraded mode.  A corrupt settings row answers the defaults (the policy on, the shipped hosts, the pipeline&#39;s own params) with the corruption logged server-side. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelSettingsResponse>> getModelSettings({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/models';
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

    ModelSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelSettingsResponse),
      ) as ModelSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelSettingsResponse>(
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

  /// Read the background work&#39;s budget.
  /// The task panel&#39;s run-settings document: the three dials that decide WHEN the background work spends the machine&#39;s capacity --  - &#x60;concurrencyPreset&#x60;: the dial&#39;s three steps, &#x60;light&#x60;   (one job at a time), &#x60;standard&#x60; (half the cores) and &#x60;full&#x60;   (all but one core). The panel never asks for a NUMBER: a   NAS owner does not know what to type, so the dial speaks in   steps and the mapping is the honest explanation line. - &#x60;nightlyEnabled&#x60; with &#x60;nightlyStart&#x60;/&#x60;nightlyEnd&#x60;: the night   window (shipped 02:00-06:00, off). While it is on, the HEAVY   work -- derivative rebuilds, video transcodes, model   recomputation -- runs only inside it; a window that ends   stops the work at the asset it just finished and the next   window resumes from there. A window that spans midnight   (start after end) is a legal shape. Uploads and browsing   are NEVER held by the window -- they are foreground work,   not background jobs. - &#x60;pausedJobs&#x60;: the five per-kind pause switches   (thumbnails, metadata, transcode, machine learning, place   resolution). A paused kind&#39;s NEW jobs sit held; the one   already running finishes. Uploads of a paused kind enqueue   without being blocked -- the request never waits, the job   does.  Both the concurrency preset and the pauses take effect immediately: every job reads this document when it starts, so a save changes what the NEXT job sees, and a held job wakes within the hold&#39;s own granularity to re-read it.  A corrupt settings row answers the defaults with the corruption logged server-side. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [RunBudgetSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<RunBudgetSettingsResponse>> getRunBudgetSettings({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/run-budget';
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

    RunBudgetSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(RunBudgetSettingsResponse),
      ) as RunBudgetSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<RunBudgetSettingsResponse>(
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

  /// Read the storage template setting.
  /// The storage settings read of the admin surface: the instance&#39;s upload path template, the sample path it renders, and the built-in preset list the settings section&#39;s dropdown fills from.  The template is the path grammar every NEW upload renders its file placement from: &#x60;originals/&lt;ownerUserId&gt;/&lt;rendered&gt;.&lt;ext&gt;&#x60;, with the per-user root pinning one directory tree per account. The grammar is variable substitution plus &#x60;{{#if album}}...{{else}}...{{/if}}&#x60; conditionals over the file (&#x60;{{filename}}&#x60;, &#x60;{{ext}}&#x60;, &#x60;{{filetype}}&#x60;, &#x60;{{filetypefull}}&#x60;), date (&#x60;{{y}} {{yy}} {{M}} {{MM}} {{MMM}} {{MMMM}} {{d}} {{dd}} {{W}} {{WW}} {{H}} {{HH}} {{h}} {{hh}} {{m}} {{mm}} {{s}} {{ss}} {{SSS}}&#x60;), album (&#x60;{{album}}&#x60;, &#x60;{{album-startDate-*}}&#x60;, &#x60;{{album-endDate-*}}&#x60;), camera (&#x60;{{make}} {{model}} {{lensModel}}&#x60;) and id (&#x60;{{assetId}}&#x60;, &#x60;{{assetIdShort}}&#x60;) families. Compilation is strict -- an unknown variable or a malformed conditional is an error, never a silent blank.  THE RENDERING CONTRACT (what changing the template does and never does): the path is rendered exactly once, at ingest, from the taken-at fallback chain (the file&#39;s EXIF header time, else the client-reported &#x60;deviceCreatedAt&#x60;, else the upload time) and the values known at that moment. Later metadata refinement NEVER re-paths an asset -- a refined taken-at, an album membership joined after upload, none of it moves a file. Existing assets keep the paths they were ingested under; there is deliberately NO migration of stored files when the template changes (moving files after ingest is the architecture photo libraries have lost data to; lumemor never moves them). New uploads follow the new template from the moment it is saved. Album variables render empty for direct uploads: the only album an upload joins at ingest is the system sync stream, which is not a naming source.  The sample path is the current template rendered over a fixed sample asset (an image named &#x60;IMG_1234.jpg&#x60;, taken &#x60;2023:07:14 10:15:30.123 +05:30&#x60;, camera &#x60;Acme Alpha 5&#x60; with the &#x60;24-70mm f/2.8&#x60; lens, no album, under the sample owner &#x60;0192f8c1-0000-7000-8000-000000000001&#x60;) with the extension appended -- exactly the shape a real upload of that file would land at under this template. The response never fails over a corrupt setting row: the default template answers instead, and the server log carries the corruption. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [StorageSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<StorageSettingsResponse>> getStorageSettings({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/storage';
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

    StorageSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(StorageSettingsResponse),
      ) as StorageSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<StorageSettingsResponse>(
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

  /// Read the map tile source setting.
  /// The map&#39;s tile sources, as the administrator configured them: the light style every theme loads as its fallback, and the optional dark style the dark theme loads instead. Either is the empty string when the instance has none.  THE SHAPE: each URL names a document the browser&#39;s map library can build a base layer from -- a style document (the vector case: sources, layers, sprites and glyphs) or a TileJSON descriptor (the raster case: the tile templates and their attribution). The map detects which of the two it received and builds the base layer accordingly, so each field covers both the raster and the vector-and-style forms of the self-hosting ecosystem.  THE DARK STYLE IS OPTIONAL: an empty &#x60;darkTilejsonUrl&#x60; means the dark theme reuses the light source, so a single source keeps a single look in both themes. A configured one is loaded by the dark theme only; the light theme always loads &#x60;tilejsonUrl&#x60;. A row written before the field existed reads as the empty string.  THE EMPTY VALUE IS THE SHIPPED DEFAULT and a real state: no source ships with the product (a public tile service&#39;s usage policy forbids carrying the traffic of every install), so a fresh instance draws its map with the tile layer absent and says so. Loading the document needs the host to send the usual cross-origin headers; a host that does not lands on the same stated absence.  A corrupt settings row answers the empty strings with the corruption logged server-side. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TilesSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TilesSettingsResponse>> getTilesSettings({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/tiles';
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

    TilesSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TilesSettingsResponse),
      ) as TilesSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TilesSettingsResponse>(
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

  /// Read the video transcoding settings.
  /// The transcoding settings read of the admin surface: the instance&#39;s video transcoding profile -- the preset the transcode job re-encodes new video uploads at, the concurrency bound of simultaneous re-encodes, the video-thumbnail frame capture switch, and the keep-original-on-failure policy. A fresh instance (or a corrupt row) answers the documented defaults. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TranscodingSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TranscodingSettingsResponse>> getTranscodingSettings({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/transcoding';
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

    TranscodingSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TranscodingSettingsResponse),
      ) as TranscodingSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TranscodingSettingsResponse>(
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

  /// Read the trash retention setting.
  /// The trash settings read of the admin surface: the instance&#39;s trash retention in whole days -- the window a soft-deleted asset stays recoverable before the hourly expiry job purges it (rows, thumbnails, derivatives and orphaned files together). A fresh instance (or a corrupt row) answers the default 30; the read never fails over a row the save endpoint&#39;s validation should have kept honest. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TrashSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TrashSettingsResponse>> getTrashSettings({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/trash';
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

    TrashSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TrashSettingsResponse),
      ) as TrashSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TrashSettingsResponse>(
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

  /// Read the instance&#39;s release check state.
  /// The instance&#39;s release check, read from the cached answer -- this request never touches the network. The server runs the probe itself on the configured cadence (a release feed URL, the interval in hours) and this face reports what the last probe concluded.  THE PROBE: one plain HTTPS request for a public document -- no photo, no EXIF, no embedding, no instance identifier; the feed sees the instance&#39;s address and nothing else. The check can be switched off (a settings switch); the endpoint address must be a public one: an address that resolves to a loopback, private, link-local or otherwise internal range is refused, and redirects are never followed. A refused or failed probe is not an error state: &#x60;status&#x60; turns &#x60;unknown&#x60; and nothing else changes -- the next interval retries.  THE COMPARISON takes tags only: an optional &#x60;v&#x60; prefix, then the numeric version fields. &#x60;up_to_date&#x60; and &#x60;update_available&#x60; are the two claims a successful probe can make; &#x60;latestVersion&#x60; carries the feed&#39;s tag verbatim. The server never downloads or installs an update -- this surface only reports that a newer release exists.  &#x60;disabled&#x60; means the check does not run: the administrator switched it off, or the deployment disabled it at boot. The last answer is kept while disabled, so switching the check back on resumes from the same state. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [UpdateCheckResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<UpdateCheckResponse>> getUpdateCheck({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/server/update-check';
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

    UpdateCheckResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(UpdateCheckResponse),
      ) as UpdateCheckResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<UpdateCheckResponse>(
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

  /// Read the release check settings.
  /// The release check&#39;s three knobs, as the settings face renders them: the switch, the release feed URL and the cadence in hours.  &#x60;endpoint&#x60; is the URL the server fetches on every check. The document behind it must be a JSON object whose newest tag rides &#x60;tag&#x60; (a release feed that names it &#x60;tag_name&#x60;, as release APIs commonly do, is read the same way). The value is handed back verbatim; the default is the product&#39;s own release feed.  &#x60;intervalHours&#x60; is the cadence: the server probes at most once per interval, and the answer is cached until the next one -- a failed probe waits the same interval instead of hammering a feed that is not there.  A corrupt settings row answers the documented defaults (enabled, the default feed, 24 hours) with the corruption logged server-side, the same discipline the other instance settings keep. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [UpdateCheckSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<UpdateCheckSettingsResponse>> getUpdateCheckSettings({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/update-check';
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

    UpdateCheckSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(UpdateCheckSettingsResponse),
      ) as UpdateCheckSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<UpdateCheckSettingsResponse>(
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

  /// List the job runs of the task panel.
  /// The task panel listing of the admin surface: every job run the instance ever recorded, newest first -- model downloads and ML reruns alike. Each row carries the run&#39;s progress as its own work&#39;s units: the model download&#39;s &#x60;total&#x60; and &#x60;done&#x60; count pinned bytes and &#x60;failed&#x60; counts failed models; a rerun&#39;s count the assets of its scope and the assets whose rerun failed. &#x60;startedAt&#x60; is null while a run sits &#x60;queued&#x60;, &#x60;finishedAt&#x60; is null until it closes.  &#x60;limit&#x60; bounds the page size: absent means the default of 50, a value over 200 is answered with the maximum of 200, and a value below 1 answers 400 &#x60;VALIDATION_FAILED&#x60;. 
  ///
  /// Parameters:
  /// * [limit] - Page size: the number of job runs one response carries at most. Defaults to 50; values over 200 are answered with the maximum of 200, values below 1 answer 400. 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [JobRunListResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<JobRunListResponse>> listJobs({ 
    int? limit = 50,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/jobs';
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

    JobRunListResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(JobRunListResponse),
      ) as JobRunListResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<JobRunListResponse>(
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

  /// List the registered models and the download run&#39;s state.
  /// The model management listing of the admin surface: every registered model of the instance with its facts, its activation state and the presence of its files, plus the most recent model download run as the summary the model page&#39;s progress bar renders.  Every model row of the &#x60;models&#x60; table appears, grouped by slot then id -- including a model whose weights went missing, which lists with &#x60;filesPresent: false&#x60; so the administrator can see and clean it up. &#x60;filesPresent&#x60; is a presence probe only (the on-disk manifest is readable and every file it pins exists); the sha256 verification itself is the activation&#39;s and the session loader&#39;s business, never the listing&#39;s. &#x60;license&#x60;, &#x60;notice&#x60; and &#x60;inputSize&#x60; come from the model&#39;s on-disk manifest, &#x60;dim&#x60; and &#x60;params&#x60; from the registered row (which the download and the scan keep in step with that manifest); &#x60;dim&#x60; and &#x60;inputSize&#x60; are null when the model declares none, &#x60;license&#x60; is empty and &#x60;notice&#x60; null when the manifest is unreadable; &#x60;isActive&#x60; comes from the table.  &#x60;download&#x60; is the most recent &#x60;model_download&#x60; run of whatever state -- the run the download endpoint&#39;s idempotence would return -- or null before the first download was ever started.  &#x60;slots&#x60; is the per-slot state view the page&#39;s cards render: one row per pipeline slot, each carrying one of the four states the cards speak (&#x60;active&#x60;, &#x60;downloading&#x60;, &#x60;not_downloaded&#x60;, &#x60;failed&#x60;), the slot&#39;s active model, its source facts (official or custom, the host, the digest, the download time) and the download run&#39;s progress while a run is in flight. &#x60;official&#x60; is the manifest&#39;s catalog -- the validated list the switch dialog&#39;s official group renders, every entry with its download state (&#x60;downloaded&#x60; is the presence probe, the digest check stays the activation&#39;s). 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelListResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelListResponse>> listModels({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/models';
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

    ModelListResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelListResponse),
      ) as ModelListResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelListResponse>(
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

  /// List the instance&#39;s accounts.
  /// The user management listing of the admin surface: every account of the instance with its role and its disabled switch, oldest first. The listing selects only the safe columns -- a password hash never appears in any admin response.  The list is small by construction (a family instance holds a handful of accounts) and grows only through this surface&#39;s own create endpoint, so it answers every account in one response: there is no pagination, no filtering and no ordering parameter. Account rows disappear only through this surface&#39;s own delete endpoint, whose disposal moves the account&#39;s content to a surviving owner or into the trash before the row goes, so a refetch after any write reconciles the list completely.  The endpoint is admin-only: the session middleware authenticates the caller like every authed route, and a caller whose role is not &#x60;admin&#x60; answers 403 AUTH_FORBIDDEN -- a code distinct from the 401 family, because the caller is signed in, just not entitled to this surface. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AdminUserListResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AdminUserListResponse>> listUsers({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/users';
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

    AdminUserListResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AdminUserListResponse),
      ) as AdminUserListResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AdminUserListResponse>(
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

  /// Probe a model source for one slot.
  /// The first half of a source import: download (or read) the candidate model, check it against the slot&#39;s contract, and stage it -- without touching the slot&#39;s active model. The answer is 202 with a probeId; the probe itself runs in the server&#39;s background (a URL download legitimately takes minutes) and is read through the probe GET until it settles.  The body is &#x60;multipart/form-data&#x60; with these parts:  - &#x60;kind&#x60; (text, required) -- &#x60;url&#x60; or &#x60;file&#x60;. - &#x60;url&#x60; (text, required for kind &#x60;url&#x60;) -- the model URL; the   download lands in the slot&#39;s staging area, its bytes are   never trusted before the digest check. - &#x60;sha256&#x60; (text, optional) -- a digest the bytes must match;   without it the probe still computes and reports the digest,   and the save marks the model an unverified source. - &#x60;file&#x60; (file, required for kind &#x60;file&#x60;) -- the ONNX bytes   (up to 2 GiB; the sniff reads the content, never the file   name).  The checks, in order, every failure naming its own check: the digest when one was given; the graph loads (a file that is not a readable ONNX model is refused); the input shape and the output dimension against the slot&#39;s contract -- a mismatch is refused naming both numbers. A host that is not on the download sources list does NOT refuse: the probe answers &#x60;warn&#x60; (the checks passed, the host is undeclared, the save is still the administrator&#39;s own act and its use is logged).  The probe does not register anything: the save endpoint takes the probeId, and a probe whose checks failed can never be saved. 
  ///
  /// Parameters:
  /// * [slot] - The slot the candidate model would fill.
  /// * [kind] - Which source the candidate comes from.
  /// * [url] - The candidate model's URL, for kind `url`.
  /// * [sha256] - The digest the bytes must match, when the caller pins one (64 lowercase hex chars). 
  /// * [file] - The candidate model's bytes, for kind `file`.
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelProbeAcceptedResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelProbeAcceptedResponse>> probeModelSource({ 
    required ModelSlot slot,
    required String kind,
    String? url,
    String? sha256,
    MultipartFile? file,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/models/{slot}/probe'.replaceAll('{' r'slot' '}', encodeQueryParameter(_serializers, slot, const FullType(ModelSlot)).toString());
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
        r'kind': encodeFormParameter(_serializers, kind, const FullType(String)),
        if (url != null) r'url': encodeFormParameter(_serializers, url, const FullType(String)),
        if (sha256 != null) r'sha256': encodeFormParameter(_serializers, sha256, const FullType(String)),
        if (file != null) r'file': file,
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

    ModelProbeAcceptedResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelProbeAcceptedResponse),
      ) as ModelProbeAcceptedResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelProbeAcceptedResponse>(
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

  /// Start a derivative rebuild run.
  /// The image card&#39;s explicit rebuild: one run that re-derives the serving copies of every image asset at the settings the instance currently carries.  THE COST IS NEVER SILENT: the image card states what the settings change (nothing is recomputed until this run is asked for), and the two scopes are the honest pair --  - &#x60;missing&#x60; derives only the assets whose current-format   serving files are absent: a library that has never been   rebuilt, or one that gained the full-size copy, pays only   for what it lacks (minutes on a family library). - &#x60;all&#x60; re-derives every image asset at the current settings   and removes the other encoding format&#39;s stale files: the CPU   belongs to the rebuild for the estimate&#39;s duration, and   browsing is slower while it runs.  THE RUN IS A JOB: it lists on the task panel with the four states, its progress counts the image assets it visited, and the pause and window rules that govern thumbnail-class work hold it the same way -- a night window that closes stops it at the asset it just finished, and the next window resumes from there, never from zero.  The endpoint is idempotent the way the place re-resolution is: a rebuild run still in flight (&#x60;queued&#x60; or &#x60;running&#x60;) answers 200 with its identifier and starts nothing; a finished run never blocks a new one.  SETTINGS CHANGES NEVER TRIGGER THIS: saving the image settings writes the document only -- the serving copies already on disk stay as they are until an explicit rebuild re-derives them. 
  ///
  /// Parameters:
  /// * [rebuildDerivativesRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [RebuildDerivativesResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<RebuildDerivativesResponse>> rebuildDerivatives({ 
    required RebuildDerivativesRequest rebuildDerivativesRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/derivatives/rebuild';
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
      const _type = FullType(RebuildDerivativesRequest);
      _bodyData = _serializers.serialize(rebuildDerivativesRequest, specifiedType: _type);

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

    RebuildDerivativesResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(RebuildDerivativesResponse),
      ) as RebuildDerivativesResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<RebuildDerivativesResponse>(
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

  /// Start a full place re-resolution run.
  /// The places-data-update path of the admin surface: one run that re-resolves the place of EVERY asset whose exif row carries GPS coordinates, against the baked place data the server currently runs on. The run that refills resolutions after the baked GeoNames data changed -- the startup import of a rebuilt image replaces the place tables and, through the schema&#39;s ON DELETE SET NULL, releases every stale resolution; this run is the deliberate step that refills them. (The metadata job resolves every NEW upload on its own; this run only matters for re-resolving what already exists.)  The scope is every GPS-bearing asset row whatever its lifecycle state -- place resolution is metadata, not visibility. &#x60;total&#x60; and &#x60;done&#x60; of the run&#39;s &#x60;job_runs&#x60; row count those assets and &#x60;failed&#x60; the ones whose re-resolution write failed.  The endpoint is idempotent the way the model rerun is: a place re-resolution run still in flight (&#x60;queued&#x60; or &#x60;running&#x60;) answers 200 with its identifier and starts nothing; a finished (or failed) run never blocks a new one.  The run is API-only for now: no admin page button is wired to it yet (the job panel&#39;s listing shows its progress like any other run). 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [PlacesRerunResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<PlacesRerunResponse>> reresolvePlaces({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/places/reresolve';
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

    PlacesRerunResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(PlacesRerunResponse),
      ) as PlacesRerunResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<PlacesRerunResponse>(
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

  /// Start a manual rerun of the active model&#39;s slot.
  /// The rerun trigger of the admin surface: one run that redoes the slot&#39;s ML work over the whole library -- the run that fills the newly activated model&#39;s products, the manual step every model switch deliberately leaves to the administrator (a same-dimension switch keeps the old products usable immediately; a cross-dimension switch leaves the table empty until this run refills it).  Only the ACTIVE model of a slot can be rerun: products attach to the active model, so a rerun of a dormant model would fill rows no active-model query could see. The run&#39;s scope is every image asset that finished its pipeline and is not in the trash (ML work belongs after asset readiness, and video assets stay out of ML so far); &#x60;total&#x60; and &#x60;done&#x60; of the run&#39;s row count those assets and &#x60;failed&#x60; the ones whose rerun failed -- progress in asset counts, not bytes (the model download&#39;s row counts pinned bytes and failed models; each run&#39;s units are its own work&#39;s).  The endpoint is idempotent the way the download endpoint is: a rerun of the same slot still in flight (&#x60;queued&#x60; or &#x60;running&#x60;) answers 200 with its identifier and starts nothing. Unlike the download, a FINISHED run never blocks a new one -- the rerun is manual, so a second click after a completed (or failed) run starts a fresh run.  The per-asset ML workers of the slots arrive later; until a slot&#39;s dispatch is wired, a rerun of that slot still creates the run but the run fails (state &#x60;failed&#x60;, zero progress) instead of pretending to have done work it cannot do. 
  ///
  /// Parameters:
  /// * [id] - The identifier of the active model whose slot is rerun (the models table's slug id, as the listing returns it). 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelRerunResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelRerunResponse>> rerunModel({ 
    required String id,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/models/{id}/rerun'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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

    ModelRerunResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelRerunResponse),
      ) as ModelRerunResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelRerunResponse>(
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

  /// Run the release check now.
  /// The manual check of the settings face: the server probes the configured release feed now (the same bounded, refused- internal-address, redirect-refusing request the scheduled check makes) and answers the fresh state -- the same document &#x60;GET&#x60; serves.  A failed probe answers 200 with &#x60;status: unknown&#x60;, exactly as the scheduled one would: a check that cannot reach its feed is the ordinary case of an offline instance, not a request error, and it carries no message. A check that is switched off (or disabled by the deployment) answers 200 with &#x60;disabled&#x60; and probes nothing. The probe is bounded server-side and well under the request timeout; a slow feed lands on &#x60;unknown&#x60; like any other failure. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [UpdateCheckResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<UpdateCheckResponse>> runUpdateCheck({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/server/update-check';
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

    UpdateCheckResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(UpdateCheckResponse),
      ) as UpdateCheckResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<UpdateCheckResponse>(
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

  /// Register the custom models found in the models dir.
  /// The custom model registration of the admin surface: every directory under the instance&#39;s models dir that carries a &#x60;manifest.json&#x60; is verified against the digests the manifest pins and registered in the models table with source &#x60;custom&#x60;. Directories without a manifest are skipped -- the models dir also holds download staging, not every entry is a model.  Scanning registers, it never activates: an administrator reviews the registered models and switches them on through the activation endpoint. Re-scanning converges: the registration is an upsert on the descriptive fields and leaves every row&#39;s activation state alone. A directory whose files drifted from their pinned digests fails the whole scan with 400 MODEL_INVALID and registers nothing. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelScanResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelScanResponse>> scanModels({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/models/scan';
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

    ModelScanResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelScanResponse),
      ) as ModelScanResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelScanResponse>(
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

  /// Stop the in-flight derivative rebuild run.
  /// The task panel&#39;s stop for a rebuild in flight: the run&#39;s row is marked as asked-to-stop, and the driver -- which checks the flag between assets -- closes the run at the asset it just finished, never mid-asset. The work already done stays on disk: a stopped rebuild has paid exactly for the assets it re-derived, and the next rebuild&#39;s &#x60;missing&#x60; scope skips them.  The run closes &#x60;cancelled&#x60; -- the task panel renders the cancelled state, not a broken run: the admin asked for this ending (E41 shared the flag-closed ending with the task panel&#39;s per-run cancel, and the two speak one state).  A stop with no rebuild in flight (the run finished between the panel&#39;s listing and the click, or none ever started) answers 409 &#x60;REBUILD_NOT_RUNNING&#x60;: the flag has nothing to land on. 
  ///
  /// Parameters:
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [RebuildDerivativesResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<RebuildDerivativesResponse>> stopDerivativesRebuild({ 
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/derivatives/rebuild/stop';
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

    RebuildDerivativesResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(RebuildDerivativesResponse),
      ) as RebuildDerivativesResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<RebuildDerivativesResponse>(
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

  /// Switch a slot onto a probed source.
  /// The second half of a source import, and the official list&#39;s switch: the slot is switched onto the source the request names, under the slot&#39;s lock. The answer is 202 -- the switch itself is seconds, the recalculation its semantics demand is an async run the task panel owns.  The body depends on the kind:  - &#x60;builtin&#x60; (the official list): &#x60;modelId&#x60; names the manifest   entry; nothing was probed because the entry is the   validated list. A cross-dimension switch of the embed slot   requires &#x60;confirmWipe: true&#x60; -- the confirmation is the   dialog&#39;s contract, and the response reports the wipe&#39;s facts   exactly as the activation switch does. - &#x60;url&#x60; or &#x60;file&#x60; (a custom source): &#x60;probeId&#x60; names a probe   that finished OK -- the server never reuses an unverified   result, a missing or failed probe answers 400. The model is   registered from the staged bytes (same sha256 reuse: bytes   already on disk are never stored twice), activated, and the   slot&#39;s recalculation run is enqueued.  The slot lock: a switch stops being possible while one is in flight (409) -- the same-dimension switch is seconds, but the recalculation it started is the slot&#39;s business, and two overlapping switches would race its products. The switch refuses to lose the named persons: a face slot&#39;s recalculation inherits the cluster names by overlap, so the confirm copy states what is kept; nothing here deletes a user label.  A failed switch changes nothing: the slot keeps its previous model and its products. 
  ///
  /// Parameters:
  /// * [slot] - The slot being switched.
  /// * [modelSourceSwitchRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelSourceSwitchResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelSourceSwitchResponse>> switchModelSource({ 
    required ModelSlot slot,
    required ModelSourceSwitchRequest modelSourceSwitchRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/models/{slot}/source'.replaceAll('{' r'slot' '}', encodeQueryParameter(_serializers, slot, const FullType(ModelSlot)).toString());
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
      const _type = FullType(ModelSourceSwitchRequest);
      _bodyData = _serializers.serialize(modelSourceSwitchRequest, specifiedType: _type);

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

    ModelSourceSwitchResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelSourceSwitchResponse),
      ) as ModelSourceSwitchResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelSourceSwitchResponse>(
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

  /// Save the derivative image settings.
  /// The settings save. Every field is contract-required (an omitted one would decode as its zero value and silently change the posture), and each value is checked against its enum: the tile sizes 250 and 320, the preview sizes 1080, 1440 and 2048, the encodings &#x60;avif&#x60; and &#x60;webp&#x60;.  The save writes the document and nothing else: no derivation starts, no existing serving file is touched, and a rebuild already running keeps its own starting parameters. The new values bind at the next derivation -- a new upload&#39;s thumbnail job, or the explicit rebuild run the image card starts.  The save takes effect without a restart: the thumbnail job reads the settings at run time, so an upload that arrives after the save derives at them. 
  ///
  /// Parameters:
  /// * [imageSettingsUpdateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ImageSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ImageSettingsResponse>> updateImageSettings({ 
    required ImageSettingsUpdateRequest imageSettingsUpdateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/image';
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
      const _type = FullType(ImageSettingsUpdateRequest);
      _bodyData = _serializers.serialize(imageSettingsUpdateRequest, specifiedType: _type);

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

    ImageSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ImageSettingsResponse),
      ) as ImageSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ImageSettingsResponse>(
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

  /// Save the model download settings and download sources.
  /// The settings save. &#x60;autoDownload&#x60;, &#x60;allowedHosts&#x60;, &#x60;mirrorBaseUrl&#x60; and &#x60;params&#x60; are required (an omitted field would decode as its zero value, silently changing the policy): the caller saves the whole object it read.  &#x60;allowedHosts&#x60; replaces the stored list wholesale; every entry must be a hostname, optionally with a leading &#x60;*.&#x60; wildcard label and an optional port. &#x60;mirrorBaseUrl&#x60;, when non-empty, must be a host (or a full http/https URL whose host is kept); its host joins &#x60;allowedHosts&#x60; automatically, so saving a mirror never leaves it refusing its own first hop. &#x60;params&#x60; carries the two pipeline steering values, each validated against the closed 0-1 domain -- a value outside it answers 400 &#x60;VALIDATION_FAILED&#x60; naming the field, and nothing is written.  The values take effect on the next dispatch (the download&#39;s next run; the ML jobs&#39; next per-asset enqueue) -- no run is started or cancelled by the save itself, and work already in flight runs to completion on the values it read. 
  ///
  /// Parameters:
  /// * [modelSettingsUpdateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [ModelSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<ModelSettingsResponse>> updateModelSettings({ 
    required ModelSettingsUpdateRequest modelSettingsUpdateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/models';
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
      const _type = FullType(ModelSettingsUpdateRequest);
      _bodyData = _serializers.serialize(modelSettingsUpdateRequest, specifiedType: _type);

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

    ModelSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(ModelSettingsResponse),
      ) as ModelSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<ModelSettingsResponse>(
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

  /// Save the background work&#39;s budget.
  /// The budget save. Every field is contract-required (an omitted one would decode as its zero value and silently change the posture), and each value is checked: the concurrency preset against its three-step enum, the two times against the &#x60;HH:MM&#x60; form, and the five pause switches against booleans.  The two times must differ: an equal pair would define a zero-length window, which reads as a window that never opens -- turning the window OFF is the honest way to say that, so the save refuses the ambiguous shape.  The save takes effect immediately, without a restart: the next job a worker picks up reads the fresh document, and a job already held wakes within its hold granularity and re-reads. The switches never touch a job already running -- the one in flight finishes, which is the card&#39;s own headline.  The window and the pauses NEVER gate an upload or a serving request: they are worker-side facts. The upload path enqueues without consulting them, and the browser&#39;s reads are answered whatever the budget says. 
  ///
  /// Parameters:
  /// * [runBudgetSettingsUpdateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [RunBudgetSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<RunBudgetSettingsResponse>> updateRunBudgetSettings({ 
    required RunBudgetSettingsUpdateRequest runBudgetSettingsUpdateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/run-budget';
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
      const _type = FullType(RunBudgetSettingsUpdateRequest);
      _bodyData = _serializers.serialize(runBudgetSettingsUpdateRequest, specifiedType: _type);

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

    RunBudgetSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(RunBudgetSettingsResponse),
      ) as RunBudgetSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<RunBudgetSettingsResponse>(
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

  /// Validate and save the storage template.
  /// The storage settings save of the admin surface. The template passes a three-stage validation before anything is written:  1. **Compile** -- strict grammar: every &#x60;{{variable}}&#x60; must be a known member of the variable table, every conditional balanced. An unknown variable or a malformed conditional is rejected here. 2. **Sample render** -- the compiled template renders over a fixed set of sample assets (an album-less image, an album-grouped image, a video), exactly as an upload would: a template that renders an empty path (an album-only template over the album-less sample) is rejected here. 3. **Structure** -- the rendered paths must be safe: no &#x60;..&#x60; traversal, no dot segments, bounded depth (10) and bounded length, per-segment and total. The engine sanitizes the user-influenced values (file name, album, camera) and the structural pass guards the literal template text.  A template that passes is saved and answers 200 with the saved value, its sample render and the presets. A template that fails any stage answers 400 &#x60;VALIDATION_FAILED&#x60; naming &#x60;template&#x60; in the details, with the specific reason in the error message; nothing is written. The save takes effect for uploads from the next request on: paths already recorded are never re-derived and never moved (see GET for the full rendering contract). 
  ///
  /// Parameters:
  /// * [storageSettingsUpdateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [StorageSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<StorageSettingsResponse>> updateStorageSettings({ 
    required StorageSettingsUpdateRequest storageSettingsUpdateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/storage';
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
      const _type = FullType(StorageSettingsUpdateRequest);
      _bodyData = _serializers.serialize(storageSettingsUpdateRequest, specifiedType: _type);

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

    StorageSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(StorageSettingsResponse),
      ) as StorageSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<StorageSettingsResponse>(
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

  /// Save the map tile source setting.
  /// The settings save. &#x60;tilejsonUrl&#x60; is required: the empty string clears the light source (the map falls back to its stated absence), any other value must be an absolute &#x60;http&#x60; or &#x60;https&#x60; URL. &#x60;darkTilejsonUrl&#x60; is the optional dark style: omitting it or sending the empty string clears it, which makes the dark theme reuse the light source, and any other value must be an absolute &#x60;http&#x60; or &#x60;https&#x60; URL as well.  The save replaces both fields: it is one document, not two partial writes. The validation is symmetric -- a bad dark URL is refused with the field named, and nothing is written.  The save validates the SHAPE only. The browser is the party that loads the source, so a LAN address is legitimate here (a self-hosted tile server inside the home network) and reachability is not judged at save time. The value takes effect on the map&#39;s next load -- there is no restart, and a map already open keeps the source it loaded. 
  ///
  /// Parameters:
  /// * [tilesSettingsUpdateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TilesSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TilesSettingsResponse>> updateTilesSettings({ 
    required TilesSettingsUpdateRequest tilesSettingsUpdateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/tiles';
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
      const _type = FullType(TilesSettingsUpdateRequest);
      _bodyData = _serializers.serialize(tilesSettingsUpdateRequest, specifiedType: _type);

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

    TilesSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TilesSettingsResponse),
      ) as TilesSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TilesSettingsResponse>(
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

  /// Save the video transcoding settings.
  /// The transcoding settings save of the admin surface.  **The scope pin, documented honestly: the profile affects NEW ingests only.** The transcode job of a video upload reads the profile when it runs, so a saved change governs every video uploaded after it; videos already transcoded keep their existing playable copy and are never re-encoded, and a video whose job is still queued picks up the profile at its run time (the job reads the setting at run time, not enqueue time).  The fields: &#x60;preset&#x60; bounds the re-encode&#39;s vertical resolution (&#x60;auto&#x60; keeps the source resolution, the numeric presets scale down to the bound, &#x60;off&#x60; skips the re-encode entirely -- the original file stays in place and downloadable, and the video&#39;s pipeline still completes); &#x60;concurrency&#x60; bounds how many re-encodes run at once; &#x60;thumbnailFrame&#x60; turns the video poster-frame capture (the video&#39;s thumbnail) on or off; &#x60;keepOriginal&#x60; decides the failure policy -- a failed re-encode either completes the job quietly (the original stays in place and downloadable through the original endpoint) or fails it for retry.  The save validates every field against its enum and answers the saved profile; an out-of-enum value answers 400 VALIDATION_FAILED naming the field, and nothing is written. 
  ///
  /// Parameters:
  /// * [transcodingSettingsUpdateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TranscodingSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TranscodingSettingsResponse>> updateTranscodingSettings({ 
    required TranscodingSettingsUpdateRequest transcodingSettingsUpdateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/transcoding';
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
      const _type = FullType(TranscodingSettingsUpdateRequest);
      _bodyData = _serializers.serialize(transcodingSettingsUpdateRequest, specifiedType: _type);

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

    TranscodingSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TranscodingSettingsResponse),
      ) as TranscodingSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TranscodingSettingsResponse>(
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

  /// Save the trash retention setting.
  /// The trash settings save of the admin surface. The retention is a whole number of days, 0 to 365:  - **0 means immediate permanent delete** -- an asset enters the trash and the NEXT expiry run purges it (the hourly periodic job and the boot run are the runs; nothing is deleted by the save request itself). - The expiry job reads the setting at the START of each run (the boot run and every hourly periodic run alike), so a change takes effect for the next run&#39;s window, not retroactively for rows a past run already considered. - An already-trashed asset&#39;s fate is decided by the run&#39;s own deadline computation: with a SHORTENED retention, rows past the new window leave on the next run; with a LENGTHENED one, every row gains the added days from its own deletion mark.  The save validates the bound and answers the saved value; a value outside 0-365 answers 400 VALIDATION_FAILED naming &#x60;retentionDays&#x60;, and nothing is written. 
  ///
  /// Parameters:
  /// * [trashSettingsUpdateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [TrashSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<TrashSettingsResponse>> updateTrashSettings({ 
    required TrashSettingsUpdateRequest trashSettingsUpdateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/trash';
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
      const _type = FullType(TrashSettingsUpdateRequest);
      _bodyData = _serializers.serialize(trashSettingsUpdateRequest, specifiedType: _type);

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

    TrashSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(TrashSettingsResponse),
      ) as TrashSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<TrashSettingsResponse>(
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

  /// Save the release check settings.
  /// The settings save: all three fields are required (an omitted &#x60;enabled&#x60; would decode as the disabled switch, silently turning the check off), and each is validated before anything is written.  &#x60;endpoint&#x60; must be an &#x60;http&#x60; or &#x60;https&#x60; URL with a host -- the address the server will probe. The probe itself refuses addresses that resolve to loopback, private, link-local or otherwise internal ranges, and never follows a redirect: the setting can be saved pointing at an internal host, and the check simply stays &#x60;unknown&#x60; (the settings save does not probe, so it cannot judge reachability).  A failed validation answers 400 &#x60;VALIDATION_FAILED&#x60; naming the offending field, with nothing written. A saved value takes effect from the next scheduled check on; the manual check runs against the saved value immediately. 
  ///
  /// Parameters:
  /// * [updateCheckSettingsUpdateRequest] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [UpdateCheckSettingsResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<UpdateCheckSettingsResponse>> updateUpdateCheckSettings({ 
    required UpdateCheckSettingsUpdateRequest updateCheckSettingsUpdateRequest,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/settings/update-check';
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
      const _type = FullType(UpdateCheckSettingsUpdateRequest);
      _bodyData = _serializers.serialize(updateCheckSettingsUpdateRequest, specifiedType: _type);

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

    UpdateCheckSettingsResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(UpdateCheckSettingsResponse),
      ) as UpdateCheckSettingsResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<UpdateCheckSettingsResponse>(
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

  /// Toggle one account&#39;s disabled switch.
  /// The enable/disable switch of the admin surface: an administrator locks an account out of the instance (or lets it back in) without deleting the account or any of its data. The switch is enforced by the session middleware on EVERY request, which is what pins its observable semantics:  - Disabling takes effect with the account&#39;s next request: an   existing session of the account answers 401 from then on   (byte-identical with any other authentication failure) and   sign-in is refused the same way. The session rows   themselves are not deleted -- disabling revokes their   authority, not their existence. - Re-enabling restores the account&#39;s unexpired sessions   without a new sign-in; a session that lapsed its own   expiry while the account was disabled is gone for good.  The write is idempotent -- setting the current value again answers 200 with the unchanged account -- and the response is the account after the write, so the client adopts it as the fresh state without a refetch.  An administrator cannot disable their own account: the last admin locking themselves out would strand the admin surface, so the request answers 400 VALIDATION_FAILED naming the id. Nothing stops an admin from disabling ANOTHER administrator; the last enabled admin is the instance&#39;s line of defense.  The id must name an account. Account ids are enumerable through this surface&#39;s own listing, so there is no anti-enumeration pairing here: an unknown id (a stale list on the client) answers 404 USER_NOT_FOUND. 
  ///
  /// Parameters:
  /// * [id] - Identifier of the account.
  /// * [adminUserPatch] 
  /// * [cancelToken] - A [CancelToken] that can be used to cancel the operation
  /// * [headers] - Can be used to add additional headers to the request
  /// * [extras] - Can be used to add flags to the request
  /// * [validateStatus] - A [ValidateStatus] callback that can be used to determine request success based on the HTTP status of the response
  /// * [onSendProgress] - A [ProgressCallback] that can be used to get the send progress
  /// * [onReceiveProgress] - A [ProgressCallback] that can be used to get the receive progress
  ///
  /// Returns a [Future] containing a [Response] with a [AdminUserResponse] as data
  /// Throws [DioException] if API call or serialization fails
  Future<Response<AdminUserResponse>> updateUser({ 
    required String id,
    required AdminUserPatch adminUserPatch,
    CancelToken? cancelToken,
    Map<String, dynamic>? headers,
    Map<String, dynamic>? extra,
    ValidateStatus? validateStatus,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    final _path = r'/admin/users/{id}'.replaceAll('{' r'id' '}', encodeQueryParameter(_serializers, id, const FullType(String)).toString());
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
      const _type = FullType(AdminUserPatch);
      _bodyData = _serializers.serialize(adminUserPatch, specifiedType: _type);

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

    AdminUserResponse? _responseData;

    try {
      final rawResponse = _response.data;
      _responseData = rawResponse == null ? null : _serializers.deserialize(
        rawResponse,
        specifiedType: const FullType(AdminUserResponse),
      ) as AdminUserResponse;

    } catch (error, stackTrace) {
      throw DioException(
        requestOptions: _response.requestOptions,
        response: _response,
        type: DioExceptionType.unknown,
        error: error,
        stackTrace: stackTrace,
      );
    }

    return Response<AdminUserResponse>(
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
