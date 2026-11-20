//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/job_run_summary.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'latest_job_runs.g.dart';

/// The most recent `job_runs` row of each batch kind, whatever its state, or null when the instance never started one. The per-album rescan and the per-slot reruns are not here: they belong to their own pages, not the catalog's batch rows. 
///
/// Properties:
/// * [rebuildDerivatives] - The latest derivative rebuild run, or null.
/// * [placesRerun] - The latest place re-resolution run, or null.
/// * [albumRescanAll] - The latest all-albums rescan sweep, or null.
/// * [folderSync] - The latest folder sweep run. The folder sweep is the activity-gated periodic diff and records no run row by design, so this slot answers null on every instance; the catalog's row states the cadence instead of a run. 
/// * [modelDownload] - The latest model download run, or null.
@BuiltValue()
abstract class LatestJobRuns implements Built<LatestJobRuns, LatestJobRunsBuilder> {
  /// The latest derivative rebuild run, or null.
  @BuiltValueField(wireName: r'rebuildDerivatives')
  JobRunSummary? get rebuildDerivatives;

  /// The latest place re-resolution run, or null.
  @BuiltValueField(wireName: r'placesRerun')
  JobRunSummary? get placesRerun;

  /// The latest all-albums rescan sweep, or null.
  @BuiltValueField(wireName: r'albumRescanAll')
  JobRunSummary? get albumRescanAll;

  /// The latest folder sweep run. The folder sweep is the activity-gated periodic diff and records no run row by design, so this slot answers null on every instance; the catalog's row states the cadence instead of a run. 
  @BuiltValueField(wireName: r'folderSync')
  JobRunSummary? get folderSync;

  /// The latest model download run, or null.
  @BuiltValueField(wireName: r'modelDownload')
  JobRunSummary? get modelDownload;

  LatestJobRuns._();

  factory LatestJobRuns([void updates(LatestJobRunsBuilder b)]) = _$LatestJobRuns;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(LatestJobRunsBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<LatestJobRuns> get serializer => _$LatestJobRunsSerializer();
}

class _$LatestJobRunsSerializer implements PrimitiveSerializer<LatestJobRuns> {
  @override
  final Iterable<Type> types = const [LatestJobRuns, _$LatestJobRuns];

  @override
  final String wireName = r'LatestJobRuns';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    LatestJobRuns object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'rebuildDerivatives';
    yield object.rebuildDerivatives == null ? null : serializers.serialize(
      object.rebuildDerivatives,
      specifiedType: const FullType.nullable(JobRunSummary),
    );
    yield r'placesRerun';
    yield object.placesRerun == null ? null : serializers.serialize(
      object.placesRerun,
      specifiedType: const FullType.nullable(JobRunSummary),
    );
    yield r'albumRescanAll';
    yield object.albumRescanAll == null ? null : serializers.serialize(
      object.albumRescanAll,
      specifiedType: const FullType.nullable(JobRunSummary),
    );
    yield r'folderSync';
    yield object.folderSync == null ? null : serializers.serialize(
      object.folderSync,
      specifiedType: const FullType.nullable(JobRunSummary),
    );
    yield r'modelDownload';
    yield object.modelDownload == null ? null : serializers.serialize(
      object.modelDownload,
      specifiedType: const FullType.nullable(JobRunSummary),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    LatestJobRuns object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required LatestJobRunsBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'rebuildDerivatives':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(JobRunSummary),
          ) as JobRunSummary?;
          if (valueDes == null) continue;
          result.rebuildDerivatives.replace(valueDes);
          break;
        case r'placesRerun':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(JobRunSummary),
          ) as JobRunSummary?;
          if (valueDes == null) continue;
          result.placesRerun.replace(valueDes);
          break;
        case r'albumRescanAll':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(JobRunSummary),
          ) as JobRunSummary?;
          if (valueDes == null) continue;
          result.albumRescanAll.replace(valueDes);
          break;
        case r'folderSync':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(JobRunSummary),
          ) as JobRunSummary?;
          if (valueDes == null) continue;
          result.folderSync.replace(valueDes);
          break;
        case r'modelDownload':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(JobRunSummary),
          ) as JobRunSummary?;
          if (valueDes == null) continue;
          result.modelDownload.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  LatestJobRuns deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = LatestJobRunsBuilder();
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


