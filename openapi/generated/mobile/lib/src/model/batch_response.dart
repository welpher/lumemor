//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'batch_response.g.dart';

/// The set-semantics report of one batch action. `affected` and `missing` partition the request's distinct ids: every id is either applied or unresolvable, none is both, and none is silently dropped. 
///
/// Properties:
/// * [affected] - How many distinct request ids resolved to a caller-owned asset and were processed -- INCLUDING ids that were already in the target state (the idempotent no-op of the set semantics: a repeat of a successful batch answers the same count, because the endpoint is a convergence operation, not a state-change counter). NOT a count of rows written: trashing a Live Photo pair moves two rows but one request id, and counts once. 
/// * [missing] - The distinct request ids that did not resolve to a caller-owned asset -- unknown ids and other accounts' assets, indistinguishable by design -- in the order of their first appearance in the request. Empty when every id resolved. This deliberate reporting (instead of the byte-identical 404 of the single-asset endpoints) is the trade of the list surface: the caller handed the ids over themselves, so reporting them back hides nothing. 
@BuiltValue()
abstract class BatchResponse implements Built<BatchResponse, BatchResponseBuilder> {
  /// How many distinct request ids resolved to a caller-owned asset and were processed -- INCLUDING ids that were already in the target state (the idempotent no-op of the set semantics: a repeat of a successful batch answers the same count, because the endpoint is a convergence operation, not a state-change counter). NOT a count of rows written: trashing a Live Photo pair moves two rows but one request id, and counts once. 
  @BuiltValueField(wireName: r'affected')
  int get affected;

  /// The distinct request ids that did not resolve to a caller-owned asset -- unknown ids and other accounts' assets, indistinguishable by design -- in the order of their first appearance in the request. Empty when every id resolved. This deliberate reporting (instead of the byte-identical 404 of the single-asset endpoints) is the trade of the list surface: the caller handed the ids over themselves, so reporting them back hides nothing. 
  @BuiltValueField(wireName: r'missing')
  BuiltList<String> get missing;

  BatchResponse._();

  factory BatchResponse([void updates(BatchResponseBuilder b)]) = _$BatchResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(BatchResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<BatchResponse> get serializer => _$BatchResponseSerializer();
}

class _$BatchResponseSerializer implements PrimitiveSerializer<BatchResponse> {
  @override
  final Iterable<Type> types = const [BatchResponse, _$BatchResponse];

  @override
  final String wireName = r'BatchResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    BatchResponse object, {
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
    BatchResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required BatchResponseBuilder result,
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
  BatchResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = BatchResponseBuilder();
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


