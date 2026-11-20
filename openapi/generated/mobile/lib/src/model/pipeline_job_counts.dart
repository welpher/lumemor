//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'pipeline_job_counts.g.dart';

/// One pipeline kind's queue counts. The three numbers partition the kind's unfinished work: a job waits, a worker runs it, or the queue discarded it after its attempts ran out. 
///
/// Properties:
/// * [waiting] - Jobs waiting for a worker: available, scheduled (a retry backoff or a pause's snooze), or retryable. 
/// * [active] - Jobs a worker is running now.
/// * [failed] - Jobs the queue discarded after their attempts ran out -- the failed work the row's danger line counts. 
@BuiltValue()
abstract class PipelineJobCounts implements Built<PipelineJobCounts, PipelineJobCountsBuilder> {
  /// Jobs waiting for a worker: available, scheduled (a retry backoff or a pause's snooze), or retryable. 
  @BuiltValueField(wireName: r'waiting')
  int get waiting;

  /// Jobs a worker is running now.
  @BuiltValueField(wireName: r'active')
  int get active;

  /// Jobs the queue discarded after their attempts ran out -- the failed work the row's danger line counts. 
  @BuiltValueField(wireName: r'failed')
  int get failed;

  PipelineJobCounts._();

  factory PipelineJobCounts([void updates(PipelineJobCountsBuilder b)]) = _$PipelineJobCounts;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PipelineJobCountsBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PipelineJobCounts> get serializer => _$PipelineJobCountsSerializer();
}

class _$PipelineJobCountsSerializer implements PrimitiveSerializer<PipelineJobCounts> {
  @override
  final Iterable<Type> types = const [PipelineJobCounts, _$PipelineJobCounts];

  @override
  final String wireName = r'PipelineJobCounts';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PipelineJobCounts object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'waiting';
    yield serializers.serialize(
      object.waiting,
      specifiedType: const FullType(int),
    );
    yield r'active';
    yield serializers.serialize(
      object.active,
      specifiedType: const FullType(int),
    );
    yield r'failed';
    yield serializers.serialize(
      object.failed,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PipelineJobCounts object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PipelineJobCountsBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'waiting':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.waiting = valueDes;
          break;
        case r'active':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.active = valueDes;
          break;
        case r'failed':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.failed = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PipelineJobCounts deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PipelineJobCountsBuilder();
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


