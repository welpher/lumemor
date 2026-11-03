//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/job_run_response.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'job_run_list_response.g.dart';

/// The task panel listing: the instance's job runs, newest first, bounded by the `limit` parameter. 
///
/// Properties:
/// * [jobs] - The newest job runs, newest first.
@BuiltValue()
abstract class JobRunListResponse implements Built<JobRunListResponse, JobRunListResponseBuilder> {
  /// The newest job runs, newest first.
  @BuiltValueField(wireName: r'jobs')
  BuiltList<JobRunResponse> get jobs;

  JobRunListResponse._();

  factory JobRunListResponse([void updates(JobRunListResponseBuilder b)]) = _$JobRunListResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(JobRunListResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<JobRunListResponse> get serializer => _$JobRunListResponseSerializer();
}

class _$JobRunListResponseSerializer implements PrimitiveSerializer<JobRunListResponse> {
  @override
  final Iterable<Type> types = const [JobRunListResponse, _$JobRunListResponse];

  @override
  final String wireName = r'JobRunListResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    JobRunListResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'jobs';
    yield serializers.serialize(
      object.jobs,
      specifiedType: const FullType(BuiltList, [FullType(JobRunResponse)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    JobRunListResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required JobRunListResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'jobs':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(JobRunResponse)]),
          ) as BuiltList<JobRunResponse>;
          result.jobs.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  JobRunListResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = JobRunListResponseBuilder();
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


