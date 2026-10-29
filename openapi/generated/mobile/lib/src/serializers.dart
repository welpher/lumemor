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

import 'package:lumemor_api/src/model/create_admin_request.dart';
import 'package:lumemor_api/src/model/error.dart';
import 'package:lumemor_api/src/model/server_info.dart';
import 'package:lumemor_api/src/model/user.dart';

part 'serializers.g.dart';

@SerializersFor([
  CreateAdminRequest,
  Error,
  ServerInfo,
  User,
])
Serializers serializers = (_$serializers.toBuilder()
      ..addBuilderFactory(
        const FullType(BuiltMap, [FullType(String), FullType(bool)]),
        () => MapBuilder<String, bool>(),
      )
      ..add(const OneOfSerializer())
      ..add(const AnyOfSerializer())
      ..add(const DateSerializer())
      ..add(Iso8601DateTimeSerializer())
    ).build();

Serializers standardSerializers =
    (serializers.toBuilder()..addPlugin(StandardJsonPlugin())).build();
