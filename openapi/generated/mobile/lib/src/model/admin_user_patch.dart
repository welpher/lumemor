//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'admin_user_patch.g.dart';

/// Payload of `PATCH /admin/users/{id}`.
///
/// Properties:
/// * [isDisabled] - The switch to write. `true` locks the account out of its sessions and sign-in; `false` lets it back in. 
@BuiltValue()
abstract class AdminUserPatch implements Built<AdminUserPatch, AdminUserPatchBuilder> {
  /// The switch to write. `true` locks the account out of its sessions and sign-in; `false` lets it back in. 
  @BuiltValueField(wireName: r'isDisabled')
  bool get isDisabled;

  AdminUserPatch._();

  factory AdminUserPatch([void updates(AdminUserPatchBuilder b)]) = _$AdminUserPatch;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AdminUserPatchBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AdminUserPatch> get serializer => _$AdminUserPatchSerializer();
}

class _$AdminUserPatchSerializer implements PrimitiveSerializer<AdminUserPatch> {
  @override
  final Iterable<Type> types = const [AdminUserPatch, _$AdminUserPatch];

  @override
  final String wireName = r'AdminUserPatch';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AdminUserPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'isDisabled';
    yield serializers.serialize(
      object.isDisabled,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AdminUserPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AdminUserPatchBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
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
  AdminUserPatch deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AdminUserPatchBuilder();
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


