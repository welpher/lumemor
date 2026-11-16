//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/job_run_kind.dart';
import 'package:lumemor_api/src/model/job_run_state.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'job_run_response.g.dart';

/// One job run as the task panel and the model page render it. The progress fields carry the run's own work's units: pinned bytes for the model download (with `failed` counting failed models), asset counts for the reruns (with `failed` counting the assets whose rerun failed). 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the run's `job_runs` row.
/// * [kind] 
/// * [state] 
/// * [subject] - The run's subject label (E41): the directory album a rescan run walks, by title, so the panel's row names what it did (\"Album rescan · Family archive\"). Every other kind has no subject -- the instance-wide reruns never name one -- and answers null; a null subject renders the kind's own label alone. 
/// * [total] - The run's whole work, in the run's own units: the download group's pinned bytes, or the rerun scope's asset count. 
/// * [done] - The work completed so far, in the same units as `total`. 
/// * [failed] - The failed work count: failed models for the download, assets whose rerun failed for a rerun. 
/// * [startedAt] - When the run's driver first picked it up (RFC 3339 UTC). Null while the run sits `queued`. 
/// * [finishedAt] - When the run closed (RFC 3339 UTC). Null until then. 
@BuiltValue()
abstract class JobRunResponse implements Built<JobRunResponse, JobRunResponseBuilder> {
  /// UUIDv7 identifier of the run's `job_runs` row.
  @BuiltValueField(wireName: r'id')
  String get id;

  @BuiltValueField(wireName: r'kind')
  JobRunKind get kind;
  // enum kindEnum {  model_download,  face_detect_rerun,  face_recognize_rerun,  clip_embed_rerun,  ocr_rerun,  places_rerun,  album_rescan,  rebuild_derivatives,  };

  @BuiltValueField(wireName: r'state')
  JobRunState get state;
  // enum stateEnum {  queued,  running,  succeeded,  failed,  cancelled,  };

  /// The run's subject label (E41): the directory album a rescan run walks, by title, so the panel's row names what it did (\"Album rescan · Family archive\"). Every other kind has no subject -- the instance-wide reruns never name one -- and answers null; a null subject renders the kind's own label alone. 
  @BuiltValueField(wireName: r'subject')
  String? get subject;

  /// The run's whole work, in the run's own units: the download group's pinned bytes, or the rerun scope's asset count. 
  @BuiltValueField(wireName: r'total')
  int get total;

  /// The work completed so far, in the same units as `total`. 
  @BuiltValueField(wireName: r'done')
  int get done;

  /// The failed work count: failed models for the download, assets whose rerun failed for a rerun. 
  @BuiltValueField(wireName: r'failed')
  int get failed;

  /// When the run's driver first picked it up (RFC 3339 UTC). Null while the run sits `queued`. 
  @BuiltValueField(wireName: r'startedAt')
  DateTime? get startedAt;

  /// When the run closed (RFC 3339 UTC). Null until then. 
  @BuiltValueField(wireName: r'finishedAt')
  DateTime? get finishedAt;

  JobRunResponse._();

  factory JobRunResponse([void updates(JobRunResponseBuilder b)]) = _$JobRunResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(JobRunResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<JobRunResponse> get serializer => _$JobRunResponseSerializer();
}

class _$JobRunResponseSerializer implements PrimitiveSerializer<JobRunResponse> {
  @override
  final Iterable<Type> types = const [JobRunResponse, _$JobRunResponse];

  @override
  final String wireName = r'JobRunResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    JobRunResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'kind';
    yield serializers.serialize(
      object.kind,
      specifiedType: const FullType(JobRunKind),
    );
    yield r'state';
    yield serializers.serialize(
      object.state,
      specifiedType: const FullType(JobRunState),
    );
    yield r'subject';
    yield object.subject == null ? null : serializers.serialize(
      object.subject,
      specifiedType: const FullType.nullable(String),
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
    JobRunResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required JobRunResponseBuilder result,
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
        case r'kind':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(JobRunKind),
          ) as JobRunKind;
          result.kind = valueDes;
          break;
        case r'state':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(JobRunState),
          ) as JobRunState;
          result.state = valueDes;
          break;
        case r'subject':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.subject = valueDes;
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
  JobRunResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = JobRunResponseBuilder();
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


