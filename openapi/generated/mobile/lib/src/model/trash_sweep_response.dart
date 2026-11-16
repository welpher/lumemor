//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'trash_sweep_response.g.dart';

/// The report of one whole-trash action (`restore-all`, `empty`): how many rows the sweep actually applied. An empty trash answers 0 -- the endpoints are idempotent, there is no not-found surface (the sweep scans what is in the trash right now, so the 404 of the single-row endpoints never appears). 
///
/// Properties:
/// * [affected] - The number of trashed rows the sweep applied: the rows the restore-all returned to the live views (the events the sweep published), or the rows the empty permanently deleted. The trash is owner-scoped, so the sweep never touches another account's rows. 
@BuiltValue()
abstract class TrashSweepResponse implements Built<TrashSweepResponse, TrashSweepResponseBuilder> {
  /// The number of trashed rows the sweep applied: the rows the restore-all returned to the live views (the events the sweep published), or the rows the empty permanently deleted. The trash is owner-scoped, so the sweep never touches another account's rows. 
  @BuiltValueField(wireName: r'affected')
  int get affected;

  TrashSweepResponse._();

  factory TrashSweepResponse([void updates(TrashSweepResponseBuilder b)]) = _$TrashSweepResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TrashSweepResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TrashSweepResponse> get serializer => _$TrashSweepResponseSerializer();
}

class _$TrashSweepResponseSerializer implements PrimitiveSerializer<TrashSweepResponse> {
  @override
  final Iterable<Type> types = const [TrashSweepResponse, _$TrashSweepResponse];

  @override
  final String wireName = r'TrashSweepResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TrashSweepResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'affected';
    yield serializers.serialize(
      object.affected,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TrashSweepResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TrashSweepResponseBuilder result,
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
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TrashSweepResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TrashSweepResponseBuilder();
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


