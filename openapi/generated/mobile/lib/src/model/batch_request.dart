//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'batch_request.g.dart';

/// One batch action over a list of asset ids. The action applies to every id that resolves to a caller-owned asset; the rest (unknown or another account's, indistinguishable) are ignored and reported in the response's `missing` list -- a batch never fails over an individual id. `ids` carries between 1 and 1000 entries; repeated ids collapse to one application. 
///
/// Properties:
/// * [action] - The set operation to apply. `favorite` / `unfavorite` write or remove the caller's per-user favorites rows; `hide` / `unhide` flip the owner-level visibility switch (a hidden asset leaves every live view until unhidden); `trash` / `restore` run the trash lifecycle of the trash endpoints, the Live Photo pair moving as one asset. 
/// * [ids] - The asset ids to apply the action to: at least 1, at most 1000. 
@BuiltValue()
abstract class BatchRequest implements Built<BatchRequest, BatchRequestBuilder> {
  /// The set operation to apply. `favorite` / `unfavorite` write or remove the caller's per-user favorites rows; `hide` / `unhide` flip the owner-level visibility switch (a hidden asset leaves every live view until unhidden); `trash` / `restore` run the trash lifecycle of the trash endpoints, the Live Photo pair moving as one asset. 
  @BuiltValueField(wireName: r'action')
  BatchRequestActionEnum get action;
  // enum actionEnum {  favorite,  unfavorite,  hide,  unhide,  trash,  restore,  };

  /// The asset ids to apply the action to: at least 1, at most 1000. 
  @BuiltValueField(wireName: r'ids')
  BuiltList<String> get ids;

  BatchRequest._();

  factory BatchRequest([void updates(BatchRequestBuilder b)]) = _$BatchRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(BatchRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<BatchRequest> get serializer => _$BatchRequestSerializer();
}

class _$BatchRequestSerializer implements PrimitiveSerializer<BatchRequest> {
  @override
  final Iterable<Type> types = const [BatchRequest, _$BatchRequest];

  @override
  final String wireName = r'BatchRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    BatchRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'action';
    yield serializers.serialize(
      object.action,
      specifiedType: const FullType(BatchRequestActionEnum),
    );
    yield r'ids';
    yield serializers.serialize(
      object.ids,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    BatchRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required BatchRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'action':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BatchRequestActionEnum),
          ) as BatchRequestActionEnum;
          result.action = valueDes;
          break;
        case r'ids':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.ids.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  BatchRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = BatchRequestBuilder();
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


/// The set operation to apply. `favorite` / `unfavorite` write or remove the caller's per-user favorites rows; `hide` / `unhide` flip the owner-level visibility switch (a hidden asset leaves every live view until unhidden); `trash` / `restore` run the trash lifecycle of the trash endpoints, the Live Photo pair moving as one asset. 
class BatchRequestActionEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'favorite')
  static const BatchRequestActionEnum favorite = _$batchRequestActionEnum_favorite;
  @BuiltValueEnumConst(wireName: r'unfavorite')
  static const BatchRequestActionEnum unfavorite = _$batchRequestActionEnum_unfavorite;
  @BuiltValueEnumConst(wireName: r'hide')
  static const BatchRequestActionEnum hide_ = _$batchRequestActionEnum_hide_;
  @BuiltValueEnumConst(wireName: r'unhide')
  static const BatchRequestActionEnum unhide = _$batchRequestActionEnum_unhide;
  @BuiltValueEnumConst(wireName: r'trash')
  static const BatchRequestActionEnum trash = _$batchRequestActionEnum_trash;
  @BuiltValueEnumConst(wireName: r'restore')
  static const BatchRequestActionEnum restore = _$batchRequestActionEnum_restore;

  static Serializer<BatchRequestActionEnum> get serializer => _$batchRequestActionEnumSerializer;

  const BatchRequestActionEnum._(String name): super(name);

  static BuiltSet<BatchRequestActionEnum> get values => _$batchRequestActionEnumValues;
  static BatchRequestActionEnum valueOf(String name) => _$batchRequestActionEnumValueOf(name);
}

