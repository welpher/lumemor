//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'server_info.g.dart';

/// Instance capability report served by `GET /server/info`.
///
/// Properties:
/// * [version] - Server version (semantic versioning).
/// * [initialized] - True once the instance has an administrator account. Clients use it to decide between the first-run wizard and the login screen. 
/// * [features] - Instance feature switches (model slots, search modes, pairing). Unknown keys must be ignored by clients. 
@BuiltValue()
abstract class ServerInfo implements Built<ServerInfo, ServerInfoBuilder> {
  /// Server version (semantic versioning).
  @BuiltValueField(wireName: r'version')
  String get version;

  /// True once the instance has an administrator account. Clients use it to decide between the first-run wizard and the login screen. 
  @BuiltValueField(wireName: r'initialized')
  bool get initialized;

  /// Instance feature switches (model slots, search modes, pairing). Unknown keys must be ignored by clients. 
  @BuiltValueField(wireName: r'features')
  BuiltMap<String, bool> get features;

  ServerInfo._();

  factory ServerInfo([void updates(ServerInfoBuilder b)]) = _$ServerInfo;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ServerInfoBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ServerInfo> get serializer => _$ServerInfoSerializer();
}

class _$ServerInfoSerializer implements PrimitiveSerializer<ServerInfo> {
  @override
  final Iterable<Type> types = const [ServerInfo, _$ServerInfo];

  @override
  final String wireName = r'ServerInfo';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ServerInfo object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'version';
    yield serializers.serialize(
      object.version,
      specifiedType: const FullType(String),
    );
    yield r'initialized';
    yield serializers.serialize(
      object.initialized,
      specifiedType: const FullType(bool),
    );
    yield r'features';
    yield serializers.serialize(
      object.features,
      specifiedType: const FullType(BuiltMap, [FullType(String), FullType(bool)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ServerInfo object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ServerInfoBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'version':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.version = valueDes;
          break;
        case r'initialized':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.initialized = valueDes;
          break;
        case r'features':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltMap, [FullType(String), FullType(bool)]),
          ) as BuiltMap<String, bool>;
          result.features.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ServerInfo deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ServerInfoBuilder();
    final serializedList = (serialized as Iterable<Object?>).toList();
    final unhandled = <Object?>[];
    _deserializeProperties(
      serializers,
      serialized,
      specifiedType: specifiedType,
      serializedList: serializedList,
      unhandled: unhandled,
      result: result,
    );
    return result.build();
  }
}


