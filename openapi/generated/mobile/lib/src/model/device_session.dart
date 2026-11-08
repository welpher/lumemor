//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'device_session.g.dart';

/// One of the caller's active sessions as the device list renders it: the row's identity and rendering fields, the `isCurrent` marker derived against the request's own credential, and the session's creation-time address. 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the session.
/// * [kind] - Where the session credential lives: `web` sessions ride the `lumemor_session` cookie, `mobile` sessions ride a bearer token. 
/// * [deviceName] - Human-readable device label: the browser's user agent captured at login for web sessions, the name supplied during pairing for mobile ones. 
/// * [lastSeenAt] - When the session was last used (RFC 3339 UTC).
/// * [ipAddress] - The address the session was created from (the login's or pairing exchange's socket peer), captured at creation time. Omitted when the row predates the column or no address was captured; an audit hint, not a geolocation claim. 
/// * [isCurrent] - Whether this row is the session the request itself authenticates with -- the device the caller is managing from. Exactly one entry of a list carries `true`, and that entry cannot be revoked through `DELETE /users/me/sessions/{id}` (the sign-out flow is its exit). 
@BuiltValue()
abstract class DeviceSession implements Built<DeviceSession, DeviceSessionBuilder> {
  /// UUIDv7 identifier of the session.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// Where the session credential lives: `web` sessions ride the `lumemor_session` cookie, `mobile` sessions ride a bearer token. 
  @BuiltValueField(wireName: r'kind')
  DeviceSessionKindEnum get kind;
  // enum kindEnum {  web,  mobile,  };

  /// Human-readable device label: the browser's user agent captured at login for web sessions, the name supplied during pairing for mobile ones. 
  @BuiltValueField(wireName: r'deviceName')
  String? get deviceName;

  /// When the session was last used (RFC 3339 UTC).
  @BuiltValueField(wireName: r'lastSeenAt')
  DateTime get lastSeenAt;

  /// The address the session was created from (the login's or pairing exchange's socket peer), captured at creation time. Omitted when the row predates the column or no address was captured; an audit hint, not a geolocation claim. 
  @BuiltValueField(wireName: r'ipAddress')
  String? get ipAddress;

  /// Whether this row is the session the request itself authenticates with -- the device the caller is managing from. Exactly one entry of a list carries `true`, and that entry cannot be revoked through `DELETE /users/me/sessions/{id}` (the sign-out flow is its exit). 
  @BuiltValueField(wireName: r'isCurrent')
  bool get isCurrent;

  DeviceSession._();

  factory DeviceSession([void updates(DeviceSessionBuilder b)]) = _$DeviceSession;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(DeviceSessionBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<DeviceSession> get serializer => _$DeviceSessionSerializer();
}

class _$DeviceSessionSerializer implements PrimitiveSerializer<DeviceSession> {
  @override
  final Iterable<Type> types = const [DeviceSession, _$DeviceSession];

  @override
  final String wireName = r'DeviceSession';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    DeviceSession object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'kind';
    yield serializers.serialize(
      object.kind,
      specifiedType: const FullType(DeviceSessionKindEnum),
    );
    if (object.deviceName != null) {
      yield r'deviceName';
      yield serializers.serialize(
        object.deviceName,
        specifiedType: const FullType(String),
      );
    }
    yield r'lastSeenAt';
    yield serializers.serialize(
      object.lastSeenAt,
      specifiedType: const FullType(DateTime),
    );
    if (object.ipAddress != null) {
      yield r'ipAddress';
      yield serializers.serialize(
        object.ipAddress,
        specifiedType: const FullType(String),
      );
    }
    yield r'isCurrent';
    yield serializers.serialize(
      object.isCurrent,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    DeviceSession object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required DeviceSessionBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'id':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.id = valueDes;
          break;
        case r'kind':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DeviceSessionKindEnum),
          ) as DeviceSessionKindEnum;
          result.kind = valueDes;
          break;
        case r'deviceName':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.deviceName = valueDes;
          break;
        case r'lastSeenAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.lastSeenAt = valueDes;
          break;
        case r'ipAddress':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.ipAddress = valueDes;
          break;
        case r'isCurrent':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isCurrent = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  DeviceSession deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = DeviceSessionBuilder();
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


/// Where the session credential lives: `web` sessions ride the `lumemor_session` cookie, `mobile` sessions ride a bearer token. 
class DeviceSessionKindEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'web')
  static const DeviceSessionKindEnum web = _$deviceSessionKindEnum_web;
  @BuiltValueEnumConst(wireName: r'mobile')
  static const DeviceSessionKindEnum mobile = _$deviceSessionKindEnum_mobile;

  static Serializer<DeviceSessionKindEnum> get serializer => _$deviceSessionKindEnumSerializer;

  const DeviceSessionKindEnum._(String name): super(name);

  static BuiltSet<DeviceSessionKindEnum> get values => _$deviceSessionKindEnumValues;
  static DeviceSessionKindEnum valueOf(String name) => _$deviceSessionKindEnumValueOf(name);
}

