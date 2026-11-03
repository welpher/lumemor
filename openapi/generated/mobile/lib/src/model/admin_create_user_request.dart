//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/user_role.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'admin_create_user_request.g.dart';

/// Payload of `POST /admin/users`.
///
/// Properties:
/// * [name] - Optional display name; defaults to the email address.
/// * [email] - Login address of the new account; unique instance-wide.
/// * [password] - Account password; stored only as an Argon2id hash.
/// * [role] 
@BuiltValue()
abstract class AdminCreateUserRequest implements Built<AdminCreateUserRequest, AdminCreateUserRequestBuilder> {
  /// Optional display name; defaults to the email address.
  @BuiltValueField(wireName: r'name')
  String? get name;

  /// Login address of the new account; unique instance-wide.
  @BuiltValueField(wireName: r'email')
  String get email;

  /// Account password; stored only as an Argon2id hash.
  @BuiltValueField(wireName: r'password')
  String get password;

  @BuiltValueField(wireName: r'role')
  UserRole? get role;
  // enum roleEnum {  admin,  user,  };

  AdminCreateUserRequest._();

  factory AdminCreateUserRequest([void updates(AdminCreateUserRequestBuilder b)]) = _$AdminCreateUserRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AdminCreateUserRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AdminCreateUserRequest> get serializer => _$AdminCreateUserRequestSerializer();
}

class _$AdminCreateUserRequestSerializer implements PrimitiveSerializer<AdminCreateUserRequest> {
  @override
  final Iterable<Type> types = const [AdminCreateUserRequest, _$AdminCreateUserRequest];

  @override
  final String wireName = r'AdminCreateUserRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AdminCreateUserRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.name != null) {
      yield r'name';
      yield serializers.serialize(
        object.name,
        specifiedType: const FullType(String),
      );
    }
    yield r'email';
    yield serializers.serialize(
      object.email,
      specifiedType: const FullType(String),
    );
    yield r'password';
    yield serializers.serialize(
      object.password,
      specifiedType: const FullType(String),
    );
    if (object.role != null) {
      yield r'role';
      yield serializers.serialize(
        object.role,
        specifiedType: const FullType(UserRole),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    AdminCreateUserRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AdminCreateUserRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.name = valueDes;
          break;
        case r'email':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.email = valueDes;
          break;
        case r'password':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.password = valueDes;
          break;
        case r'role':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(UserRole),
          ) as UserRole?;
          if (valueDes == null) continue;
          result.role = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AdminCreateUserRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AdminCreateUserRequestBuilder();
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


