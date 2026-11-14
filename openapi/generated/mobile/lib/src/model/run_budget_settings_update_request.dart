//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/concurrency_preset.dart';
import 'package:lumemor_api/src/model/paused_jobs_state.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'run_budget_settings_update_request.g.dart';

/// The budget save. Every field is contract-required: the PUT replaces the document wholesale, so an omitted field would silently reset it. The two times must differ. 
///
/// Properties:
/// * [concurrencyPreset] 
/// * [nightlyEnabled] 
/// * [nightlyStart] 
/// * [nightlyEnd] 
/// * [pausedJobs] 
@BuiltValue()
abstract class RunBudgetSettingsUpdateRequest implements Built<RunBudgetSettingsUpdateRequest, RunBudgetSettingsUpdateRequestBuilder> {
  @BuiltValueField(wireName: r'concurrencyPreset')
  ConcurrencyPreset get concurrencyPreset;
  // enum concurrencyPresetEnum {  light,  standard,  full,  };

  @BuiltValueField(wireName: r'nightlyEnabled')
  bool get nightlyEnabled;

  @BuiltValueField(wireName: r'nightlyStart')
  String get nightlyStart;

  @BuiltValueField(wireName: r'nightlyEnd')
  String get nightlyEnd;

  @BuiltValueField(wireName: r'pausedJobs')
  PausedJobsState get pausedJobs;

  RunBudgetSettingsUpdateRequest._();

  factory RunBudgetSettingsUpdateRequest([void updates(RunBudgetSettingsUpdateRequestBuilder b)]) = _$RunBudgetSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(RunBudgetSettingsUpdateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<RunBudgetSettingsUpdateRequest> get serializer => _$RunBudgetSettingsUpdateRequestSerializer();
}

class _$RunBudgetSettingsUpdateRequestSerializer implements PrimitiveSerializer<RunBudgetSettingsUpdateRequest> {
  @override
  final Iterable<Type> types = const [RunBudgetSettingsUpdateRequest, _$RunBudgetSettingsUpdateRequest];

  @override
  final String wireName = r'RunBudgetSettingsUpdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    RunBudgetSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'concurrencyPreset';
    yield serializers.serialize(
      object.concurrencyPreset,
      specifiedType: const FullType(ConcurrencyPreset),
    );
    yield r'nightlyEnabled';
    yield serializers.serialize(
      object.nightlyEnabled,
      specifiedType: const FullType(bool),
    );
    yield r'nightlyStart';
    yield serializers.serialize(
      object.nightlyStart,
      specifiedType: const FullType(String),
    );
    yield r'nightlyEnd';
    yield serializers.serialize(
      object.nightlyEnd,
      specifiedType: const FullType(String),
    );
    yield r'pausedJobs';
    yield serializers.serialize(
      object.pausedJobs,
      specifiedType: const FullType(PausedJobsState),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    RunBudgetSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required RunBudgetSettingsUpdateRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'concurrencyPreset':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ConcurrencyPreset),
          ) as ConcurrencyPreset;
          result.concurrencyPreset = valueDes;
          break;
        case r'nightlyEnabled':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.nightlyEnabled = valueDes;
          break;
        case r'nightlyStart':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.nightlyStart = valueDes;
          break;
        case r'nightlyEnd':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.nightlyEnd = valueDes;
          break;
        case r'pausedJobs':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(PausedJobsState),
          ) as PausedJobsState;
          result.pausedJobs.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  RunBudgetSettingsUpdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = RunBudgetSettingsUpdateRequestBuilder();
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


