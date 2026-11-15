//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'admin_user_delete_request.g.dart';

/// Payload of `DELETE /admin/users/{id}`: the disposition of the departing account's photos. Exactly one of the two fields is present -- `transferTo` hands them to another member, and `dispose` walks them into the caller's trash. 
///
/// Properties:
/// * [transferTo] - Identifier of the account that receives every photo, album and memory the departing account owned. Must name another existing account: the departing account itself answers 409 USER_TRANSFER_TO_SELF, an unknown id 400 VALIDATION_FAILED. 
/// * [dispose] - The alternative to `transferTo`: the departing account's photos enter the calling administrator's trash, where the instance's retention window governs their permanent removal. 
@BuiltValue()
abstract class AdminUserDeleteRequest implements Built<AdminUserDeleteRequest, AdminUserDeleteRequestBuilder> {
  /// Identifier of the account that receives every photo, album and memory the departing account owned. Must name another existing account: the departing account itself answers 409 USER_TRANSFER_TO_SELF, an unknown id 400 VALIDATION_FAILED. 
  @BuiltValueField(wireName: r'transferTo')
  String? get transferTo;

  /// The alternative to `transferTo`: the departing account's photos enter the calling administrator's trash, where the instance's retention window governs their permanent removal. 
  @BuiltValueField(wireName: r'dispose')
  AdminUserDeleteRequestDisposeEnum? get dispose;
  // enum disposeEnum {  trash,  };

  AdminUserDeleteRequest._();

  factory AdminUserDeleteRequest([void updates(AdminUserDeleteRequestBuilder b)]) = _$AdminUserDeleteRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AdminUserDeleteRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AdminUserDeleteRequest> get serializer => _$AdminUserDeleteRequestSerializer();
}

class _$AdminUserDeleteRequestSerializer implements PrimitiveSerializer<AdminUserDeleteRequest> {
  @override
  final Iterable<Type> types = const [AdminUserDeleteRequest, _$AdminUserDeleteRequest];

  @override
  final String wireName = r'AdminUserDeleteRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AdminUserDeleteRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.transferTo != null) {
      yield r'transferTo';
      yield serializers.serialize(
        object.transferTo,
        specifiedType: const FullType(String),
      );
    }
    if (object.dispose != null) {
      yield r'dispose';
      yield serializers.serialize(
        object.dispose,
        specifiedType: const FullType(AdminUserDeleteRequestDisposeEnum),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    AdminUserDeleteRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AdminUserDeleteRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'transferTo':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.transferTo = valueDes;
          break;
        case r'dispose':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(AdminUserDeleteRequestDisposeEnum),
          ) as AdminUserDeleteRequestDisposeEnum?;
          if (valueDes == null) continue;
          result.dispose = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AdminUserDeleteRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AdminUserDeleteRequestBuilder();
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


/// The alternative to `transferTo`: the departing account's photos enter the calling administrator's trash, where the instance's retention window governs their permanent removal. 
class AdminUserDeleteRequestDisposeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'trash')
  static const AdminUserDeleteRequestDisposeEnum UserDisposeTrash = _$adminUserDeleteRequestDisposeEnum_UserDisposeTrash;

  static Serializer<AdminUserDeleteRequestDisposeEnum> get serializer => _$adminUserDeleteRequestDisposeEnumSerializer;

  const AdminUserDeleteRequestDisposeEnum._(String name): super(name);

  static BuiltSet<AdminUserDeleteRequestDisposeEnum> get values => _$adminUserDeleteRequestDisposeEnumValues;
  static AdminUserDeleteRequestDisposeEnum valueOf(String name) => _$adminUserDeleteRequestDisposeEnumValueOf(name);
}

