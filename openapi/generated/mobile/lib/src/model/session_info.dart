//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/user_role.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'session_info.g.dart';

/// One authenticated session of a user. Web sessions authenticate through the `lumemor_session` cookie and mobile sessions through their bearer token; both are the same session records and appear together in `GET /auth/sessions`.  The session carries its user's identity: `userId` and `role` name whose credentials this is, so a client that restores a session (a web app booting with the cookie, a mobile app resuming with its bearer token) knows who is calling without a second endpoint. `POST /auth/login`, `GET /auth/sessions` and `POST /auth/pairing/consume` all answer this schema, so the identity arrives at login time, at restore time and at pairing time alike. The role is the live account row's, read through the session's own user join. 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the session.
/// * [kind] - Where the session credential lives: `web` sessions ride the `lumemor_session` cookie and slide their expiry by 30 days on use; `mobile` sessions ride a bearer token and are revoked explicitly. 
/// * [deviceName] - Human-readable device label of the session. For web sessions it is the browser user agent captured at login; mobile sessions carry the name supplied during pairing. 
/// * [userId] - UUIDv7 identifier of the account the session belongs to. Every entry of `GET /auth/sessions` carries the caller's own id (the list is the caller's sessions); the caller is who the client compares asset and album ownership against. 
/// * [role] 
/// * [createdAt] - When the session was created (RFC 3339 UTC).
/// * [lastSeenAt] - When the session was last used (RFC 3339 UTC).
/// * [expiresAt] - When the session expires (RFC 3339 UTC). Null for bearer sessions, which are revoked explicitly instead of expiring. 
@BuiltValue()
abstract class SessionInfo implements Built<SessionInfo, SessionInfoBuilder> {
  /// UUIDv7 identifier of the session.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// Where the session credential lives: `web` sessions ride the `lumemor_session` cookie and slide their expiry by 30 days on use; `mobile` sessions ride a bearer token and are revoked explicitly. 
  @BuiltValueField(wireName: r'kind')
  SessionInfoKindEnum get kind;
  // enum kindEnum {  web,  mobile,  };

  /// Human-readable device label of the session. For web sessions it is the browser user agent captured at login; mobile sessions carry the name supplied during pairing. 
  @BuiltValueField(wireName: r'deviceName')
  String? get deviceName;

  /// UUIDv7 identifier of the account the session belongs to. Every entry of `GET /auth/sessions` carries the caller's own id (the list is the caller's sessions); the caller is who the client compares asset and album ownership against. 
  @BuiltValueField(wireName: r'userId')
  String get userId;

  @BuiltValueField(wireName: r'role')
  UserRole get role;
  // enum roleEnum {  admin,  user,  };

  /// When the session was created (RFC 3339 UTC).
  @BuiltValueField(wireName: r'createdAt')
  DateTime get createdAt;

  /// When the session was last used (RFC 3339 UTC).
  @BuiltValueField(wireName: r'lastSeenAt')
  DateTime get lastSeenAt;

  /// When the session expires (RFC 3339 UTC). Null for bearer sessions, which are revoked explicitly instead of expiring. 
  @BuiltValueField(wireName: r'expiresAt')
  DateTime? get expiresAt;

  SessionInfo._();

  factory SessionInfo([void updates(SessionInfoBuilder b)]) = _$SessionInfo;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(SessionInfoBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<SessionInfo> get serializer => _$SessionInfoSerializer();
}

class _$SessionInfoSerializer implements PrimitiveSerializer<SessionInfo> {
  @override
  final Iterable<Type> types = const [SessionInfo, _$SessionInfo];

  @override
  final String wireName = r'SessionInfo';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    SessionInfo object, {
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
      specifiedType: const FullType(SessionInfoKindEnum),
    );
    if (object.deviceName != null) {
      yield r'deviceName';
      yield serializers.serialize(
        object.deviceName,
        specifiedType: const FullType(String),
      );
    }
    yield r'userId';
    yield serializers.serialize(
      object.userId,
      specifiedType: const FullType(String),
    );
    yield r'role';
    yield serializers.serialize(
      object.role,
      specifiedType: const FullType(UserRole),
    );
    yield r'createdAt';
    yield serializers.serialize(
      object.createdAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'lastSeenAt';
    yield serializers.serialize(
      object.lastSeenAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'expiresAt';
    yield object.expiresAt == null ? null : serializers.serialize(
      object.expiresAt,
      specifiedType: const FullType.nullable(DateTime),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    SessionInfo object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required SessionInfoBuilder result,
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
            specifiedType: const FullType(SessionInfoKindEnum),
          ) as SessionInfoKindEnum;
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
        case r'userId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.userId = valueDes;
          break;
        case r'role':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(UserRole),
          ) as UserRole;
          result.role = valueDes;
          break;
        case r'createdAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.createdAt = valueDes;
          break;
        case r'lastSeenAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.lastSeenAt = valueDes;
          break;
        case r'expiresAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(DateTime),
          ) as DateTime?;
          if (valueDes == null) continue;
          result.expiresAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  SessionInfo deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = SessionInfoBuilder();
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


/// Where the session credential lives: `web` sessions ride the `lumemor_session` cookie and slide their expiry by 30 days on use; `mobile` sessions ride a bearer token and are revoked explicitly. 
class SessionInfoKindEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'web')
  static const SessionInfoKindEnum web = _$sessionInfoKindEnum_web;
  @BuiltValueEnumConst(wireName: r'mobile')
  static const SessionInfoKindEnum mobile = _$sessionInfoKindEnum_mobile;

  static Serializer<SessionInfoKindEnum> get serializer => _$sessionInfoKindEnumSerializer;

  const SessionInfoKindEnum._(String name): super(name);

  static BuiltSet<SessionInfoKindEnum> get values => _$sessionInfoKindEnumValues;
  static SessionInfoKindEnum valueOf(String name) => _$sessionInfoKindEnumValueOf(name);
}

