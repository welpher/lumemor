//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/user_role.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'admin_user_response.g.dart';

/// One account as the admin surface returns it: the identity and role columns together with the disabled switch this surface toggles. The password hash is never part of an admin response. 
///
/// Properties:
/// * [id] - UUIDv7 identifier.
/// * [name] - Display name.
/// * [email] - Login address.
/// * [role] 
/// * [isDisabled] - The disabled switch. A disabled account loses the authority of its sessions and its sign-in until re-enabled; its data stays untouched. 
/// * [createdAt] - Creation time (RFC 3339 UTC).
@BuiltValue()
abstract class AdminUserResponse implements Built<AdminUserResponse, AdminUserResponseBuilder> {
  /// UUIDv7 identifier.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// Display name.
  @BuiltValueField(wireName: r'name')
  String get name;

  /// Login address.
  @BuiltValueField(wireName: r'email')
  String get email;

  @BuiltValueField(wireName: r'role')
  UserRole get role;
  // enum roleEnum {  admin,  user,  };

  /// The disabled switch. A disabled account loses the authority of its sessions and its sign-in until re-enabled; its data stays untouched. 
  @BuiltValueField(wireName: r'isDisabled')
  bool get isDisabled;

  /// Creation time (RFC 3339 UTC).
  @BuiltValueField(wireName: r'createdAt')
  DateTime get createdAt;

  AdminUserResponse._();

  factory AdminUserResponse([void updates(AdminUserResponseBuilder b)]) = _$AdminUserResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AdminUserResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AdminUserResponse> get serializer => _$AdminUserResponseSerializer();
}

class _$AdminUserResponseSerializer implements PrimitiveSerializer<AdminUserResponse> {
  @override
  final Iterable<Type> types = const [AdminUserResponse, _$AdminUserResponse];

  @override
  final String wireName = r'AdminUserResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AdminUserResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
    yield r'email';
    yield serializers.serialize(
      object.email,
      specifiedType: const FullType(String),
    );
    yield r'role';
    yield serializers.serialize(
      object.role,
      specifiedType: const FullType(UserRole),
    );
    yield r'isDisabled';
    yield serializers.serialize(
      object.isDisabled,
      specifiedType: const FullType(bool),
    );
    yield r'createdAt';
    yield serializers.serialize(
      object.createdAt,
      specifiedType: const FullType(DateTime),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AdminUserResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AdminUserResponseBuilder result,
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
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        case r'email':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.email = valueDes;
          break;
        case r'role':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(UserRole),
          ) as UserRole;
          result.role = valueDes;
          break;
        case r'isDisabled':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isDisabled = valueDes;
          break;
        case r'createdAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.createdAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AdminUserResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AdminUserResponseBuilder();
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


