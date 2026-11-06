//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_import

import 'package:one_of_serializer/any_of_serializer.dart';
import 'package:one_of_serializer/one_of_serializer.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/json_object.dart';
import 'package:built_value/serializer.dart';
import 'package:built_value/standard_json_plugin.dart';
import 'package:built_value/iso_8601_date_time_serializer.dart';
import 'package:lumemor_api/src/date_serializer.dart';
import 'package:lumemor_api/src/model/date.dart';

import 'package:lumemor_api/src/model/add_album_items_response.dart';
import 'package:lumemor_api/src/model/admin_create_user_request.dart';
import 'package:lumemor_api/src/model/admin_user_list_response.dart';
import 'package:lumemor_api/src/model/admin_user_patch.dart';
import 'package:lumemor_api/src/model/admin_user_response.dart';
import 'package:lumemor_api/src/model/album_create_request.dart';
import 'package:lumemor_api/src/model/album_detail.dart';
import 'package:lumemor_api/src/model/album_item.dart';
import 'package:lumemor_api/src/model/album_item_asset.dart';
import 'package:lumemor_api/src/model/album_items_request.dart';
import 'package:lumemor_api/src/model/album_owner_ref.dart';
import 'package:lumemor_api/src/model/album_patch.dart';
import 'package:lumemor_api/src/model/album_share_entry.dart';
import 'package:lumemor_api/src/model/album_share_grant_request.dart';
import 'package:lumemor_api/src/model/album_share_response.dart';
import 'package:lumemor_api/src/model/album_summary.dart';
import 'package:lumemor_api/src/model/album_timeline_preference_request.dart';
import 'package:lumemor_api/src/model/album_timeline_preference_response.dart';
import 'package:lumemor_api/src/model/album_wall_response.dart';
import 'package:lumemor_api/src/model/asset_detail.dart';
import 'package:lumemor_api/src/model/asset_face_summary.dart';
import 'package:lumemor_api/src/model/asset_patch.dart';
import 'package:lumemor_api/src/model/asset_response.dart';
import 'package:lumemor_api/src/model/batch_request.dart';
import 'package:lumemor_api/src/model/batch_response.dart';
import 'package:lumemor_api/src/model/bearer_session.dart';
import 'package:lumemor_api/src/model/cluster_detail.dart';
import 'package:lumemor_api/src/model/create_admin_request.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/exif_summary.dart';
import 'package:lumemor_api/src/model/face_detail.dart';
import 'package:lumemor_api/src/model/face_detail_asset.dart';
import 'package:lumemor_api/src/model/job_run_kind.dart';
import 'package:lumemor_api/src/model/job_run_list_response.dart';
import 'package:lumemor_api/src/model/job_run_response.dart';
import 'package:lumemor_api/src/model/job_run_state.dart';
import 'package:lumemor_api/src/model/label_person_request.dart';
import 'package:lumemor_api/src/model/login_request.dart';
import 'package:lumemor_api/src/model/merge_people_request.dart';
import 'package:lumemor_api/src/model/merge_people_response.dart';
import 'package:lumemor_api/src/model/model_activate_response.dart';
import 'package:lumemor_api/src/model/model_dimension_change.dart';
import 'package:lumemor_api/src/model/model_download_response.dart';
import 'package:lumemor_api/src/model/model_list_response.dart';
import 'package:lumemor_api/src/model/model_ref.dart';
import 'package:lumemor_api/src/model/model_rerun_response.dart';
import 'package:lumemor_api/src/model/model_response.dart';
import 'package:lumemor_api/src/model/model_response_input_size.dart';
import 'package:lumemor_api/src/model/model_scan_response.dart';
import 'package:lumemor_api/src/model/model_slot.dart';
import 'package:lumemor_api/src/model/model_source.dart';
import 'package:lumemor_api/src/model/move_face_request.dart';
import 'package:lumemor_api/src/model/move_face_response.dart';
import 'package:lumemor_api/src/model/pairing_consume_request.dart';
import 'package:lumemor_api/src/model/pairing_token_response.dart';
import 'package:lumemor_api/src/model/person_label.dart';
import 'package:lumemor_api/src/model/person_label_response.dart';
import 'package:lumemor_api/src/model/person_summary.dart';
import 'package:lumemor_api/src/model/place_summary.dart';
import 'package:lumemor_api/src/model/places_rerun_response.dart';
import 'package:lumemor_api/src/model/remove_album_items_response.dart';
import 'package:lumemor_api/src/model/server_info.dart';
import 'package:lumemor_api/src/model/session_info.dart';
import 'package:lumemor_api/src/model/set_person_hidden_request.dart';
import 'package:lumemor_api/src/model/timeline_item.dart';
import 'package:lumemor_api/src/model/timeline_response.dart';
import 'package:lumemor_api/src/model/trash_item.dart';
import 'package:lumemor_api/src/model/trash_response.dart';
import 'package:lumemor_api/src/model/unnamed_cluster_summary.dart';
import 'package:lumemor_api/src/model/unnamed_clusters_response.dart';
import 'package:lumemor_api/src/model/user.dart';
import 'package:lumemor_api/src/model/user_list_response.dart';
import 'package:lumemor_api/src/model/user_role.dart';
import 'package:lumemor_api/src/model/user_summary.dart';

part 'serializers.g.dart';

@SerializersFor([
  AddAlbumItemsResponse,
  AdminCreateUserRequest,
  AdminUserListResponse,
  AdminUserPatch,
  AdminUserResponse,
  AlbumCreateRequest,
  AlbumDetail,
  AlbumItem,
  AlbumItemAsset,
  AlbumItemsRequest,
  AlbumOwnerRef,
  AlbumPatch,
  AlbumShareEntry,
  AlbumShareGrantRequest,
  AlbumShareResponse,
  AlbumSummary,
  AlbumTimelinePreferenceRequest,
  AlbumTimelinePreferenceResponse,
  AlbumWallResponse,
  AssetDetail,
  AssetFaceSummary,
  AssetPatch,
  AssetResponse,
  BatchRequest,
  BatchResponse,
  BearerSession,
  ClusterDetail,
  CreateAdminRequest,
  Error,
  ExifSummary,
  FaceDetail,
  FaceDetailAsset,
  JobRunKind,
  JobRunListResponse,
  JobRunResponse,
  JobRunState,
  LabelPersonRequest,
  LoginRequest,
  MergePeopleRequest,
  MergePeopleResponse,
  ModelActivateResponse,
  ModelDimensionChange,
  ModelDownloadResponse,
  ModelListResponse,
  ModelRef,
  ModelRerunResponse,
  ModelResponse,
  ModelResponseInputSize,
  ModelScanResponse,
  ModelSlot,
  ModelSource,
  MoveFaceRequest,
  MoveFaceResponse,
  PairingConsumeRequest,
  PairingTokenResponse,
  PersonLabel,
  PersonLabelResponse,
  PersonSummary,
  PlaceSummary,
  PlacesRerunResponse,
  RemoveAlbumItemsResponse,
  ServerInfo,
  SessionInfo,
  SetPersonHiddenRequest,
  TimelineItem,
  TimelineResponse,
  TrashItem,
  TrashResponse,
  UnnamedClusterSummary,
  UnnamedClustersResponse,
  User,
  UserListResponse,
  UserRole,
  UserSummary,
])
Serializers serializers = (_$serializers.toBuilder()
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AssetFaceSummary)]),
        () => ListBuilder<AssetFaceSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(TimelineItem)]),
        () => ListBuilder<TimelineItem>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(PlaceSummary)]),
        () => ListBuilder<PlaceSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AlbumSummary)]),
        () => ListBuilder<AlbumSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(TrashItem)]),
        () => ListBuilder<TrashItem>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(UnnamedClusterSummary)]),
        () => ListBuilder<UnnamedClusterSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(PersonSummary)]),
        () => ListBuilder<PersonSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(ModelResponse)]),
        () => ListBuilder<ModelResponse>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AlbumShareEntry)]),
        () => ListBuilder<AlbumShareEntry>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AdminUserResponse)]),
        () => ListBuilder<AdminUserResponse>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AlbumItem)]),
        () => ListBuilder<AlbumItem>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(SessionInfo)]),
        () => ListBuilder<SessionInfo>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltMap, [FullType(String), FullType.nullable(JsonObject)]),
        () => MapBuilder<String, JsonObject?>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(UserSummary)]),
        () => ListBuilder<UserSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltMap, [FullType(String), FullType(bool)]),
        () => MapBuilder<String, bool>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(JobRunResponse)]),
        () => ListBuilder<JobRunResponse>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(String)]),
        () => ListBuilder<String>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(FaceDetail)]),
        () => ListBuilder<FaceDetail>(),
      )
      ..add(const OneOfSerializer())
      ..add(const AnyOfSerializer())
      ..add(const DateSerializer())
      ..add(Iso8601DateTimeSerializer())
    ).build();

Serializers standardSerializers =
    (serializers.toBuilder()..addPlugin(StandardJsonPlugin())).build();
