//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/job_run_state.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'job_run_summary.g.dart';

/// One batch run's latest-state summary: enough for the catalog's \"latest\" line, never the run's whole story (the history listing carries that). 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the run's `job_runs` row.
/// * [state] 
/// * [total] - The run's whole work in its own units: albums for the all-albums rescan, assets for the rebuild and the place re-resolution, pinned bytes for the model download. 
/// * [done] - The work completed so far, in the same units.
/// * [failed] - The failed work count, in the same units.
/// * [startedAt] - When the run's driver first picked it up (RFC 3339 UTC). Null while the run sits `queued`. 
/// * [finishedAt] - When the run closed (RFC 3339 UTC). Null until then.
@BuiltValue()
abstract class JobRunSummary implements Built<JobRunSummary, JobRunSummaryBuilder> {
  /// UUIDv7 identifier of the run's `job_runs` row.
  @BuiltValueField(wireName: r'id')
  String get id;

  @BuiltValueField(wireName: r'state')
  JobRunState get state;
  // enum stateEnum {  queued,  running,  succeeded,  failed,  cancelled,  };

  /// The run's whole work in its own units: albums for the all-albums rescan, assets for the rebuild and the place re-resolution, pinned bytes for the model download. 
  @BuiltValueField(wireName: r'total')
  int get total;

  /// The work completed so far, in the same units.
  @BuiltValueField(wireName: r'done')
  int get done;

  /// The failed work count, in the same units.
  @BuiltValueField(wireName: r'failed')
  int get failed;

  /// When the run's driver first picked it up (RFC 3339 UTC). Null while the run sits `queued`. 
  @BuiltValueField(wireName: r'startedAt')
  DateTime? get startedAt;

  /// When the run closed (RFC 3339 UTC). Null until then.
  @BuiltValueField(wireName: r'finishedAt')
  DateTime? get finishedAt;

  JobRunSummary._();

  factory JobRunSummary([void updates(JobRunSummaryBuilder b)]) = _$JobRunSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(JobRunSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<JobRunSummary> get serializer => _$JobRunSummarySerializer();
}

class _$JobRunSummarySerializer implements PrimitiveSerializer<JobRunSummary> {
  @override
  final Iterable<Type> types = const [JobRunSummary, _$JobRunSummary];

  @override
  final String wireName = r'JobRunSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    JobRunSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'state';
    yield serializers.serialize(
      object.state,
      specifiedType: const FullType(JobRunState),
    );
    yield r'total';
    yield serializers.serialize(
      object.total,
      specifiedType: const FullType(int),
    );
    yield r'done';
    yield serializers.serialize(
      object.done,
      specifiedType: const FullType(int),
    );
    yield r'failed';
    yield serializers.serialize(
      object.failed,
      specifiedType: const FullType(int),
    );
    yield r'startedAt';
    yield object.startedAt == null ? null : serializers.serialize(
      object.startedAt,
      specifiedType: const FullType.nullable(DateTime),
    );
    yield r'finishedAt';
    yield object.finishedAt == null ? null : serializers.serialize(
      object.finishedAt,
      specifiedType: const FullType.nullable(DateTime),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    JobRunSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required JobRunSummaryBuilder result,
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
        case r'state':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(JobRunState),
          ) as JobRunState;
          result.state = valueDes;
          break;
        case r'total':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.total = valueDes;
          break;
        case r'done':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.done = valueDes;
          break;
        case r'failed':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.failed = valueDes;
          break;
        case r'startedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(DateTime),
          ) as DateTime?;
          if (valueDes == null) continue;
          result.startedAt = valueDes;
          break;
        case r'finishedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(DateTime),
          ) as DateTime?;
          if (valueDes == null) continue;
          result.finishedAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  JobRunSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = JobRunSummaryBuilder();
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


