//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'create_admin_request.g.dart';

/// Payload of the first-run wizard (`POST /server/setup/admin`).
///
/// Properties:
/// * [name] - Optional display name; defaults to the email address.
/// * [email] - Login address of the administrator.
/// * [password] - Account password; stored only as an Argon2id hash.
@BuiltValue()
abstract class CreateAdminRequest implements Built<CreateAdminRequest, CreateAdminRequestBuilder> {
  /// Optional display name; defaults to the email address.
  @BuiltValueField(wireName: r'name')
  String? get name;

  /// Login address of the administrator.
  @BuiltValueField(wireName: r'email')
  String get email;

  /// Account password; stored only as an Argon2id hash.
  @BuiltValueField(wireName: r'password')
  String get password;

  CreateAdminRequest._();

  factory CreateAdminRequest([void updates(CreateAdminRequestBuilder b)]) = _$CreateAdminRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(CreateAdminRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<CreateAdminRequest> get serializer => _$CreateAdminRequestSerializer();
}

class _$CreateAdminRequestSerializer implements PrimitiveSerializer<CreateAdminRequest> {
  @override
  final Iterable<Type> types = const [CreateAdminRequest, _$CreateAdminRequest];

  @override
  final String wireName = r'CreateAdminRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    CreateAdminRequest object, {
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
  }

  @override
  Object serialize(
    Serializers serializers,
    CreateAdminRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required CreateAdminRequestBuilder result,
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
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  CreateAdminRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = CreateAdminRequestBuilder();
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


