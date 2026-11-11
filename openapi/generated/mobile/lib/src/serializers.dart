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

import 'package:lumemor_api/src/model/add_tag_assets_response.dart';
import 'package:lumemor_api/src/model/admin_create_user_request.dart';
import 'package:lumemor_api/src/model/admin_user_list_response.dart';
import 'package:lumemor_api/src/model/admin_user_patch.dart';
import 'package:lumemor_api/src/model/admin_user_response.dart';
import 'package:lumemor_api/src/model/album_audience_request.dart';
import 'package:lumemor_api/src/model/album_audience_response.dart';
import 'package:lumemor_api/src/model/album_audience_skip.dart';
import 'package:lumemor_api/src/model/album_create_request.dart';
import 'package:lumemor_api/src/model/album_create_response.dart';
import 'package:lumemor_api/src/model/album_detail.dart';
import 'package:lumemor_api/src/model/album_facet_count.dart';
import 'package:lumemor_api/src/model/album_item.dart';
import 'package:lumemor_api/src/model/album_item_asset.dart';
import 'package:lumemor_api/src/model/album_items_request.dart';
import 'package:lumemor_api/src/model/album_owner_ref.dart';
import 'package:lumemor_api/src/model/album_patch.dart';
import 'package:lumemor_api/src/model/album_path_completion.dart';
import 'package:lumemor_api/src/model/album_path_completions_response.dart';
import 'package:lumemor_api/src/model/album_rescan_response.dart';
import 'package:lumemor_api/src/model/album_share_entry.dart';
import 'package:lumemor_api/src/model/album_summary.dart';
import 'package:lumemor_api/src/model/album_timeline_preference_request.dart';
import 'package:lumemor_api/src/model/album_timeline_preference_response.dart';
import 'package:lumemor_api/src/model/album_wall_response.dart';
import 'package:lumemor_api/src/model/asset_album_ref.dart';
import 'package:lumemor_api/src/model/asset_detail.dart';
import 'package:lumemor_api/src/model/asset_face_summary.dart';
import 'package:lumemor_api/src/model/asset_patch.dart';
import 'package:lumemor_api/src/model/asset_place_ref.dart';
import 'package:lumemor_api/src/model/asset_response.dart';
import 'package:lumemor_api/src/model/asset_tag_summary.dart';
import 'package:lumemor_api/src/model/batch_request.dart';
import 'package:lumemor_api/src/model/batch_response.dart';
import 'package:lumemor_api/src/model/bearer_session.dart';
import 'package:lumemor_api/src/model/camera_facet_count.dart';
import 'package:lumemor_api/src/model/cluster_detail.dart';
import 'package:lumemor_api/src/model/create_admin_request.dart';
import 'package:lumemor_api/src/model/device_session.dart';
import 'package:lumemor_api/src/model/download_format.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/exif_summary.dart';
import 'package:lumemor_api/src/model/face_detail.dart';
import 'package:lumemor_api/src/model/face_detail_asset.dart';
import 'package:lumemor_api/src/model/job_run_kind.dart';
import 'package:lumemor_api/src/model/job_run_list_response.dart';
import 'package:lumemor_api/src/model/job_run_response.dart';
import 'package:lumemor_api/src/model/job_run_state.dart';
import 'package:lumemor_api/src/model/kind_facet_count.dart';
import 'package:lumemor_api/src/model/label_person_request.dart';
import 'package:lumemor_api/src/model/login_request.dart';
import 'package:lumemor_api/src/model/media_kind.dart';
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
import 'package:lumemor_api/src/model/password_change_request.dart';
import 'package:lumemor_api/src/model/person_facet_count.dart';
import 'package:lumemor_api/src/model/person_label.dart';
import 'package:lumemor_api/src/model/person_label_response.dart';
import 'package:lumemor_api/src/model/person_summary.dart';
import 'package:lumemor_api/src/model/place_facet_count.dart';
import 'package:lumemor_api/src/model/place_summary.dart';
import 'package:lumemor_api/src/model/places_rerun_response.dart';
import 'package:lumemor_api/src/model/remove_album_items_response.dart';
import 'package:lumemor_api/src/model/remove_tag_assets_response.dart';
import 'package:lumemor_api/src/model/search_facets.dart';
import 'package:lumemor_api/src/model/search_response.dart';
import 'package:lumemor_api/src/model/server_info.dart';
import 'package:lumemor_api/src/model/session_info.dart';
import 'package:lumemor_api/src/model/set_person_hidden_request.dart';
import 'package:lumemor_api/src/model/storage_settings_response.dart';
import 'package:lumemor_api/src/model/storage_settings_update_request.dart';
import 'package:lumemor_api/src/model/tag_assets_request.dart';
import 'package:lumemor_api/src/model/tag_facet_count.dart';
import 'package:lumemor_api/src/model/tag_name_request.dart';
import 'package:lumemor_api/src/model/tag_summary.dart';
import 'package:lumemor_api/src/model/timeline_item.dart';
import 'package:lumemor_api/src/model/timeline_response.dart';
import 'package:lumemor_api/src/model/transcode_preset.dart';
import 'package:lumemor_api/src/model/transcoding_settings_response.dart';
import 'package:lumemor_api/src/model/transcoding_settings_update_request.dart';
import 'package:lumemor_api/src/model/trash_item.dart';
import 'package:lumemor_api/src/model/trash_response.dart';
import 'package:lumemor_api/src/model/trash_settings_response.dart';
import 'package:lumemor_api/src/model/trash_settings_update_request.dart';
import 'package:lumemor_api/src/model/unnamed_cluster_summary.dart';
import 'package:lumemor_api/src/model/unnamed_clusters_response.dart';
import 'package:lumemor_api/src/model/user.dart';
import 'package:lumemor_api/src/model/user_list_response.dart';
import 'package:lumemor_api/src/model/user_locale.dart';
import 'package:lumemor_api/src/model/user_role.dart';
import 'package:lumemor_api/src/model/user_self_patch.dart';
import 'package:lumemor_api/src/model/user_self_response.dart';
import 'package:lumemor_api/src/model/user_summary.dart';
import 'package:lumemor_api/src/model/user_theme.dart';
import 'package:lumemor_api/src/model/year_facet_count.dart';

part 'serializers.g.dart';

@SerializersFor([
  AddTagAssetsResponse,
  AdminCreateUserRequest,
  AdminUserListResponse,
  AdminUserPatch,
  AdminUserResponse,
  AlbumAudienceRequest,
  AlbumAudienceResponse,
  AlbumAudienceSkip,
  AlbumCreateRequest,
  AlbumCreateResponse,
  AlbumDetail,
  AlbumFacetCount,
  AlbumItem,
  AlbumItemAsset,
  AlbumItemsRequest,
  AlbumOwnerRef,
  AlbumPatch,
  AlbumPathCompletion,
  AlbumPathCompletionsResponse,
  AlbumRescanResponse,
  AlbumShareEntry,
  AlbumSummary,$AlbumSummary,
  AlbumTimelinePreferenceRequest,
  AlbumTimelinePreferenceResponse,
  AlbumWallResponse,
  AssetAlbumRef,
  AssetDetail,
  AssetFaceSummary,
  AssetPatch,
  AssetPlaceRef,
  AssetResponse,
  AssetTagSummary,
  BatchRequest,
  BatchResponse,
  BearerSession,
  CameraFacetCount,
  ClusterDetail,
  CreateAdminRequest,
  DeviceSession,
  DownloadFormat,
  Error,
  ExifSummary,
  FaceDetail,
  FaceDetailAsset,
  JobRunKind,
  JobRunListResponse,
  JobRunResponse,
  JobRunState,
  KindFacetCount,
  LabelPersonRequest,
  LoginRequest,
  MediaKind,
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
  PasswordChangeRequest,
  PersonFacetCount,
  PersonLabel,
  PersonLabelResponse,
  PersonSummary,
  PlaceFacetCount,
  PlaceSummary,
  PlacesRerunResponse,
  RemoveAlbumItemsResponse,
  RemoveTagAssetsResponse,
  SearchFacets,
  SearchResponse,
  ServerInfo,
  SessionInfo,
  SetPersonHiddenRequest,
  StorageSettingsResponse,
  StorageSettingsUpdateRequest,
  TagAssetsRequest,
  TagFacetCount,
  TagNameRequest,
  TagSummary,
  TimelineItem,
  TimelineResponse,
  TranscodePreset,
  TranscodingSettingsResponse,
  TranscodingSettingsUpdateRequest,
  TrashItem,
  TrashResponse,
  TrashSettingsResponse,
  TrashSettingsUpdateRequest,
  UnnamedClusterSummary,
  UnnamedClustersResponse,
  User,
  UserListResponse,
  UserLocale,
  UserRole,
  UserSelfPatch,
  UserSelfResponse,
  UserSummary,
  UserTheme,
  YearFacetCount,
])
Serializers serializers = (_$serializers.toBuilder()
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(TimelineItem)]),
        () => ListBuilder<TimelineItem>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(PlaceSummary)]),
        () => ListBuilder<PlaceSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(UnnamedClusterSummary)]),
        () => ListBuilder<UnnamedClusterSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(PlaceFacetCount)]),
        () => ListBuilder<PlaceFacetCount>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(PersonSummary)]),
        () => ListBuilder<PersonSummary>(),
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
        const FullType(BuiltList, [FullType(PersonFacetCount)]),
        () => ListBuilder<PersonFacetCount>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(SessionInfo)]),
        () => ListBuilder<SessionInfo>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(DeviceSession)]),
        () => ListBuilder<DeviceSession>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(YearFacetCount)]),
        () => ListBuilder<YearFacetCount>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(TagFacetCount)]),
        () => ListBuilder<TagFacetCount>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(JobRunResponse)]),
        () => ListBuilder<JobRunResponse>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AssetAlbumRef)]),
        () => ListBuilder<AssetAlbumRef>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AssetFaceSummary)]),
        () => ListBuilder<AssetFaceSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AlbumFacetCount)]),
        () => ListBuilder<AlbumFacetCount>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(CameraFacetCount)]),
        () => ListBuilder<CameraFacetCount>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AlbumSummary)]),
        () => ListBuilder<AlbumSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(KindFacetCount)]),
        () => ListBuilder<KindFacetCount>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(TrashItem)]),
        () => ListBuilder<TrashItem>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(MediaKind)]),
        () => ListBuilder<MediaKind>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(ModelResponse)]),
        () => ListBuilder<ModelResponse>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AlbumAudienceSkip)]),
        () => ListBuilder<AlbumAudienceSkip>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AssetTagSummary)]),
        () => ListBuilder<AssetTagSummary>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(int)]),
        () => ListBuilder<int>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltMap, [FullType(String), FullType.nullable(JsonObject)]),
        () => MapBuilder<String, JsonObject?>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(TagSummary)]),
        () => ListBuilder<TagSummary>(),
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
        const FullType(BuiltList, [FullType(String)]),
        () => ListBuilder<String>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(AlbumPathCompletion)]),
        () => ListBuilder<AlbumPathCompletion>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(FaceDetail)]),
        () => ListBuilder<FaceDetail>(),
      )
      ..add(AlbumSummary.serializer)
      ..add(const OneOfSerializer())
      ..add(const AnyOfSerializer())
      ..add(const DateSerializer())
      ..add(Iso8601DateTimeSerializer())
    ).build();

Serializers standardSerializers =
    (serializers.toBuilder()..addPlugin(StandardJsonPlugin())).build();
