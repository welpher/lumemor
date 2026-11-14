//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'slot_state_response_progress.g.dart';

/// The in-flight download run's `{done, total}` pinned-byte counts, null when no run is in flight. 
///
/// Properties:
/// * [done] 
/// * [total] 
@BuiltValue()
abstract class SlotStateResponseProgress implements Built<SlotStateResponseProgress, SlotStateResponseProgressBuilder> {
  @BuiltValueField(wireName: r'done')
  int get done;

  @BuiltValueField(wireName: r'total')
  int get total;

  SlotStateResponseProgress._();

  factory SlotStateResponseProgress([void updates(SlotStateResponseProgressBuilder b)]) = _$SlotStateResponseProgress;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(SlotStateResponseProgressBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<SlotStateResponseProgress> get serializer => _$SlotStateResponseProgressSerializer();
}

class _$SlotStateResponseProgressSerializer implements PrimitiveSerializer<SlotStateResponseProgress> {
  @override
  final Iterable<Type> types = const [SlotStateResponseProgress, _$SlotStateResponseProgress];

  @override
  final String wireName = r'SlotStateResponseProgress';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    SlotStateResponseProgress object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'done';
    yield serializers.serialize(
      object.done,
      specifiedType: const FullType(int),
    );
    yield r'total';
    yield serializers.serialize(
      object.total,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    SlotStateResponseProgress object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required SlotStateResponseProgressBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'done':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.done = valueDes;
          break;
        case r'total':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.total = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  SlotStateResponseProgress deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = SlotStateResponseProgressBuilder();
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


