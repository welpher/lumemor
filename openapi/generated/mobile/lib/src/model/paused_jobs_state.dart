//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'paused_jobs_state.g.dart';

/// The five per-kind pause switches. A paused kind's NEW jobs sit held and the one already running finishes; the switches never touch an upload request or a serving read. 
///
/// Properties:
/// * [thumbnail] - Holds new thumbnail derivations -- and, with them, the derivative rebuild (its work is thumbnail-class). 
/// * [metadata] - Holds new metadata jobs.
/// * [transcode] - Holds new video transcodes.
/// * [ml] - Holds new machine learning work: the four per-asset inference jobs and the slot reruns. 
/// * [places] - Holds new place re-resolution runs.
@BuiltValue()
abstract class PausedJobsState implements Built<PausedJobsState, PausedJobsStateBuilder> {
  /// Holds new thumbnail derivations -- and, with them, the derivative rebuild (its work is thumbnail-class). 
  @BuiltValueField(wireName: r'thumbnail')
  bool get thumbnail;

  /// Holds new metadata jobs.
  @BuiltValueField(wireName: r'metadata')
  bool get metadata;

  /// Holds new video transcodes.
  @BuiltValueField(wireName: r'transcode')
  bool get transcode;

  /// Holds new machine learning work: the four per-asset inference jobs and the slot reruns. 
  @BuiltValueField(wireName: r'ml')
  bool get ml;

  /// Holds new place re-resolution runs.
  @BuiltValueField(wireName: r'places')
  bool get places;

  PausedJobsState._();

  factory PausedJobsState([void updates(PausedJobsStateBuilder b)]) = _$PausedJobsState;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PausedJobsStateBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PausedJobsState> get serializer => _$PausedJobsStateSerializer();
}

class _$PausedJobsStateSerializer implements PrimitiveSerializer<PausedJobsState> {
  @override
  final Iterable<Type> types = const [PausedJobsState, _$PausedJobsState];

  @override
  final String wireName = r'PausedJobsState';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PausedJobsState object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'thumbnail';
    yield serializers.serialize(
      object.thumbnail,
      specifiedType: const FullType(bool),
    );
    yield r'metadata';
    yield serializers.serialize(
      object.metadata,
      specifiedType: const FullType(bool),
    );
    yield r'transcode';
    yield serializers.serialize(
      object.transcode,
      specifiedType: const FullType(bool),
    );
    yield r'ml';
    yield serializers.serialize(
      object.ml,
      specifiedType: const FullType(bool),
    );
    yield r'places';
    yield serializers.serialize(
      object.places,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PausedJobsState object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PausedJobsStateBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'thumbnail':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.thumbnail = valueDes;
          break;
        case r'metadata':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.metadata = valueDes;
          break;
        case r'transcode':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.transcode = valueDes;
          break;
        case r'ml':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.ml = valueDes;
          break;
        case r'places':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.places = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PausedJobsState deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PausedJobsStateBuilder();
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


