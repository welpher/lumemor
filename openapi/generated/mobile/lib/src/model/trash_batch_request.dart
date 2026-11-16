//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'trash_batch_request.g.dart';

/// One batch action over a list of ids IN THE CALLER'S TRASH. Both actions are destructive-of-place: `restore` returns the rows to the live views, `delete` removes them permanently -- the client confirms the delete arm first. The id resolution mirrors the assets batch's set semantics with a trash-scoped guard: an id that is not in the caller's trash at the time of the request (unknown, another account's, already live or already purged) is ignored and reported in `missing`; a batch never fails over an individual id. `ids` carries between 1 and 1000 entries; repeated ids collapse to one application. 
///
/// Properties:
/// * [action] - The action to apply to every id in the caller's trash. `restore` clears the trash mark (the Live Photo pair moving as one asset, the external row's file moving back to its recorded original location); `delete` removes the rows and their files permanently (the shared-blob rule applies, the component video rows go with their photos). 
/// * [ids] - The asset ids to apply the action to: at least 1, at most 1000. 
@BuiltValue()
abstract class TrashBatchRequest implements Built<TrashBatchRequest, TrashBatchRequestBuilder> {
  /// The action to apply to every id in the caller's trash. `restore` clears the trash mark (the Live Photo pair moving as one asset, the external row's file moving back to its recorded original location); `delete` removes the rows and their files permanently (the shared-blob rule applies, the component video rows go with their photos). 
  @BuiltValueField(wireName: r'action')
  TrashBatchRequestActionEnum get action;
  // enum actionEnum {  restore,  delete,  };

  /// The asset ids to apply the action to: at least 1, at most 1000. 
  @BuiltValueField(wireName: r'ids')
  BuiltList<String> get ids;

  TrashBatchRequest._();

  factory TrashBatchRequest([void updates(TrashBatchRequestBuilder b)]) = _$TrashBatchRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TrashBatchRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TrashBatchRequest> get serializer => _$TrashBatchRequestSerializer();
}

class _$TrashBatchRequestSerializer implements PrimitiveSerializer<TrashBatchRequest> {
  @override
  final Iterable<Type> types = const [TrashBatchRequest, _$TrashBatchRequest];

  @override
  final String wireName = r'TrashBatchRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TrashBatchRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'action';
    yield serializers.serialize(
      object.action,
      specifiedType: const FullType(TrashBatchRequestActionEnum),
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
    TrashBatchRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TrashBatchRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'action':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(TrashBatchRequestActionEnum),
          ) as TrashBatchRequestActionEnum;
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
  TrashBatchRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TrashBatchRequestBuilder();
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


/// The action to apply to every id in the caller's trash. `restore` clears the trash mark (the Live Photo pair moving as one asset, the external row's file moving back to its recorded original location); `delete` removes the rows and their files permanently (the shared-blob rule applies, the component video rows go with their photos). 
class TrashBatchRequestActionEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'restore')
  static const TrashBatchRequestActionEnum restore = _$trashBatchRequestActionEnum_restore;
  @BuiltValueEnumConst(wireName: r'delete')
  static const TrashBatchRequestActionEnum delete = _$trashBatchRequestActionEnum_delete;

  static Serializer<TrashBatchRequestActionEnum> get serializer => _$trashBatchRequestActionEnumSerializer;

  const TrashBatchRequestActionEnum._(String name): super(name);

  static BuiltSet<TrashBatchRequestActionEnum> get values => _$trashBatchRequestActionEnumValues;
  static TrashBatchRequestActionEnum valueOf(String name) => _$trashBatchRequestActionEnumValueOf(name);
}

