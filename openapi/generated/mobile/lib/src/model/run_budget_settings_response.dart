//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/concurrency_preset.dart';
import 'package:lumemor_api/src/model/paused_jobs_state.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'run_budget_settings_response.g.dart';

/// The background work's budget: the concurrency dial's preset, the night window, and the five per-kind pause switches. Every job reads this document when it starts, so a save takes effect on the next job without a restart. 
///
/// Properties:
/// * [concurrencyPreset] 
/// * [nightlyEnabled] - Whether the night window is on. Off, the heavy work runs whenever there is work. 
/// * [nightlyStart] - The window's opening time, `HH:MM` local to the server. Shipped `02:00`. 
/// * [nightlyEnd] - The window's closing time, `HH:MM` local to the server. Shipped `06:00`; earlier than the start is a window that spans midnight. 
/// * [pausedJobs] 
@BuiltValue()
abstract class RunBudgetSettingsResponse implements Built<RunBudgetSettingsResponse, RunBudgetSettingsResponseBuilder> {
  @BuiltValueField(wireName: r'concurrencyPreset')
  ConcurrencyPreset get concurrencyPreset;
  // enum concurrencyPresetEnum {  light,  standard,  full,  };

  /// Whether the night window is on. Off, the heavy work runs whenever there is work. 
  @BuiltValueField(wireName: r'nightlyEnabled')
  bool get nightlyEnabled;

  /// The window's opening time, `HH:MM` local to the server. Shipped `02:00`. 
  @BuiltValueField(wireName: r'nightlyStart')
  String get nightlyStart;

  /// The window's closing time, `HH:MM` local to the server. Shipped `06:00`; earlier than the start is a window that spans midnight. 
  @BuiltValueField(wireName: r'nightlyEnd')
  String get nightlyEnd;

  @BuiltValueField(wireName: r'pausedJobs')
  PausedJobsState get pausedJobs;

  RunBudgetSettingsResponse._();

  factory RunBudgetSettingsResponse([void updates(RunBudgetSettingsResponseBuilder b)]) = _$RunBudgetSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(RunBudgetSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<RunBudgetSettingsResponse> get serializer => _$RunBudgetSettingsResponseSerializer();
}

class _$RunBudgetSettingsResponseSerializer implements PrimitiveSerializer<RunBudgetSettingsResponse> {
  @override
  final Iterable<Type> types = const [RunBudgetSettingsResponse, _$RunBudgetSettingsResponse];

  @override
  final String wireName = r'RunBudgetSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    RunBudgetSettingsResponse object, {
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
    RunBudgetSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required RunBudgetSettingsResponseBuilder result,
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
  RunBudgetSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = RunBudgetSettingsResponseBuilder();
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


