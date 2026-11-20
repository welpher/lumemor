//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/pipeline_counts.dart';
import 'package:lumemor_api/src/model/latest_job_runs.dart';
import 'package:lumemor_api/src/model/paused_jobs_state.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'jobs_summary_response.g.dart';

/// The fixed task catalog's state in one read: the live queue counts per pipeline kind, the run budget's pause switches and the latest run of each batch kind. The catalog never lists runs -- the history listing owns that. 
///
/// Properties:
/// * [pipeline] 
/// * [paused] 
/// * [runs] 
@BuiltValue()
abstract class JobsSummaryResponse implements Built<JobsSummaryResponse, JobsSummaryResponseBuilder> {
  @BuiltValueField(wireName: r'pipeline')
  PipelineCounts get pipeline;

  @BuiltValueField(wireName: r'paused')
  PausedJobsState get paused;

  @BuiltValueField(wireName: r'runs')
  LatestJobRuns get runs;

  JobsSummaryResponse._();

  factory JobsSummaryResponse([void updates(JobsSummaryResponseBuilder b)]) = _$JobsSummaryResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(JobsSummaryResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<JobsSummaryResponse> get serializer => _$JobsSummaryResponseSerializer();
}

class _$JobsSummaryResponseSerializer implements PrimitiveSerializer<JobsSummaryResponse> {
  @override
  final Iterable<Type> types = const [JobsSummaryResponse, _$JobsSummaryResponse];

  @override
  final String wireName = r'JobsSummaryResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    JobsSummaryResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'pipeline';
    yield serializers.serialize(
      object.pipeline,
      specifiedType: const FullType(PipelineCounts),
    );
    yield r'paused';
    yield serializers.serialize(
      object.paused,
      specifiedType: const FullType(PausedJobsState),
    );
    yield r'runs';
    yield serializers.serialize(
      object.runs,
      specifiedType: const FullType(LatestJobRuns),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    JobsSummaryResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required JobsSummaryResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'pipeline':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PipelineCounts),
          ) as PipelineCounts;
          result.pipeline.replace(valueDes);
          break;
        case r'paused':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PausedJobsState),
          ) as PausedJobsState;
          result.paused.replace(valueDes);
          break;
        case r'runs':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(LatestJobRuns),
          ) as LatestJobRuns;
          result.runs.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  JobsSummaryResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = JobsSummaryResponseBuilder();
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


