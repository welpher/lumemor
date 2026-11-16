//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'trash_batch_response.g.dart';

/// The set-semantics report of one trash batch action. `affected` and `missing` partition the request's distinct ids: every id is either applied or was not in the caller's trash, none is both, and none is silently dropped. 
///
/// Properties:
/// * [affected] - How many distinct request ids were in the caller's trash at the time of the request and were processed. The trash batch's guard is trash-scoped, so an id already live or already purged lands in `missing` instead of counting here (the deliberate difference from the assets batch, whose state-agnostic guard counts an already-in-state row as affected): the trash view's rows are the caller's whole world, and a row that left it between render and click is a missing row, not a no-op. 
/// * [missing] - The distinct request ids that were not in the caller's trash at the time of the request -- unknown ids, other accounts' assets (indistinguishable by design), rows already restored to the live views and rows the expiry job already purged -- in the order of their first appearance in the request. Empty when every id was still in the trash. 
@BuiltValue()
abstract class TrashBatchResponse implements Built<TrashBatchResponse, TrashBatchResponseBuilder> {
  /// How many distinct request ids were in the caller's trash at the time of the request and were processed. The trash batch's guard is trash-scoped, so an id already live or already purged lands in `missing` instead of counting here (the deliberate difference from the assets batch, whose state-agnostic guard counts an already-in-state row as affected): the trash view's rows are the caller's whole world, and a row that left it between render and click is a missing row, not a no-op. 
  @BuiltValueField(wireName: r'affected')
  int get affected;

  /// The distinct request ids that were not in the caller's trash at the time of the request -- unknown ids, other accounts' assets (indistinguishable by design), rows already restored to the live views and rows the expiry job already purged -- in the order of their first appearance in the request. Empty when every id was still in the trash. 
  @BuiltValueField(wireName: r'missing')
  BuiltList<String> get missing;

  TrashBatchResponse._();

  factory TrashBatchResponse([void updates(TrashBatchResponseBuilder b)]) = _$TrashBatchResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TrashBatchResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TrashBatchResponse> get serializer => _$TrashBatchResponseSerializer();
}

class _$TrashBatchResponseSerializer implements PrimitiveSerializer<TrashBatchResponse> {
  @override
  final Iterable<Type> types = const [TrashBatchResponse, _$TrashBatchResponse];

  @override
  final String wireName = r'TrashBatchResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TrashBatchResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'affected';
    yield serializers.serialize(
      object.affected,
      specifiedType: const FullType(int),
    );
    yield r'missing';
    yield serializers.serialize(
      object.missing,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TrashBatchResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TrashBatchResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'affected':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.affected = valueDes;
          break;
        case r'missing':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.missing.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TrashBatchResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TrashBatchResponseBuilder();
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


