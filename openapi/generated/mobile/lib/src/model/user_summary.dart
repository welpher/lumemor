//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/user_role.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'user_summary.g.dart';

/// One account of the minimal member list: exactly the identity fields a pick list renders -- the display name (with the email as the fallback for an account that set none), the role and the disabled switch; no timestamps, and never a password hash. 
///
/// Properties:
/// * [id] - UUIDv7 identifier.
/// * [email] - Login address: the identity's unique key, the fallback a pick list renders when the display name is empty, and the disambiguator between two accounts that share a name. 
/// * [name] - The account's display name, exactly as stored. The empty string when the account set none -- the client then renders `email` instead. 
/// * [role] 
/// * [isDisabled] - The disabled switch: a disabled account cannot sign in and its sessions carry no authority until re-enabled. 
@BuiltValue()
abstract class UserSummary implements Built<UserSummary, UserSummaryBuilder> {
  /// UUIDv7 identifier.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// Login address: the identity's unique key, the fallback a pick list renders when the display name is empty, and the disambiguator between two accounts that share a name. 
  @BuiltValueField(wireName: r'email')
  String get email;

  /// The account's display name, exactly as stored. The empty string when the account set none -- the client then renders `email` instead. 
  @BuiltValueField(wireName: r'name')
  String get name;

  @BuiltValueField(wireName: r'role')
  UserRole get role;
  // enum roleEnum {  admin,  user,  };

  /// The disabled switch: a disabled account cannot sign in and its sessions carry no authority until re-enabled. 
  @BuiltValueField(wireName: r'isDisabled')
  bool get isDisabled;

  UserSummary._();

  factory UserSummary([void updates(UserSummaryBuilder b)]) = _$UserSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UserSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UserSummary> get serializer => _$UserSummarySerializer();
}

class _$UserSummarySerializer implements PrimitiveSerializer<UserSummary> {
  @override
  final Iterable<Type> types = const [UserSummary, _$UserSummary];

  @override
  final String wireName = r'UserSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UserSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'email';
    yield serializers.serialize(
      object.email,
      specifiedType: const FullType(String),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
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
  }

  @override
  Object serialize(
    Serializers serializers,
    UserSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UserSummaryBuilder result,
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
        case r'email':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.email = valueDes;
          break;
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
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
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UserSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UserSummaryBuilder();
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


