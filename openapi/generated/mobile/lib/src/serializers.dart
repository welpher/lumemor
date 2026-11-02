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

import 'package:lumemor_api/src/model/asset_detail.dart';
import 'package:lumemor_api/src/model/asset_patch.dart';
import 'package:lumemor_api/src/model/asset_response.dart';
import 'package:lumemor_api/src/model/batch_request.dart';
import 'package:lumemor_api/src/model/batch_response.dart';
import 'package:lumemor_api/src/model/bearer_session.dart';
import 'package:lumemor_api/src/model/create_admin_request.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/exif_summary.dart';
import 'package:lumemor_api/src/model/login_request.dart';
import 'package:lumemor_api/src/model/pairing_consume_request.dart';
import 'package:lumemor_api/src/model/pairing_token_response.dart';
import 'package:lumemor_api/src/model/server_info.dart';
import 'package:lumemor_api/src/model/session_info.dart';
import 'package:lumemor_api/src/model/timeline_item.dart';
import 'package:lumemor_api/src/model/timeline_response.dart';
import 'package:lumemor_api/src/model/trash_item.dart';
import 'package:lumemor_api/src/model/trash_response.dart';
import 'package:lumemor_api/src/model/user.dart';

part 'serializers.g.dart';

@SerializersFor([
  AssetDetail,
  AssetPatch,
  AssetResponse,
  BatchRequest,
  BatchResponse,
  BearerSession,
  CreateAdminRequest,
  Error,
  ExifSummary,
  LoginRequest,
  PairingConsumeRequest,
  PairingTokenResponse,
  ServerInfo,
  SessionInfo,
  TimelineItem,
  TimelineResponse,
  TrashItem,
  TrashResponse,
  User,
])
Serializers serializers = (_$serializers.toBuilder()
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(TimelineItem)]),
        () => ListBuilder<TimelineItem>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(SessionInfo)]),
        () => ListBuilder<SessionInfo>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(TrashItem)]),
        () => ListBuilder<TrashItem>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltMap, [FullType(String), FullType.nullable(JsonObject)]),
        () => MapBuilder<String, JsonObject?>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltMap, [FullType(String), FullType(bool)]),
        () => MapBuilder<String, bool>(),
      )
      ..addBuilderFactory(
        const FullType(BuiltList, [FullType(String)]),
        () => ListBuilder<String>(),
      )
      ..add(const OneOfSerializer())
      ..add(const AnyOfSerializer())
      ..add(const DateSerializer())
      ..add(Iso8601DateTimeSerializer())
    ).build();

Serializers standardSerializers =
    (serializers.toBuilder()..addPlugin(StandardJsonPlugin())).build();
