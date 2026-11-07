//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/transcode_preset.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'transcoding_settings_response.g.dart';

/// The transcoding settings of `GET /admin/settings/transcoding` and the successful `PUT`: the instance's video transcoding profile. 
///
/// Properties:
/// * [preset] 
/// * [concurrency] - How many video re-encodes may run at once: 1 (low load), 2 (the recommended default) or 4 (high load). 
/// * [thumbnailFrame] - Whether the video thumbnail job captures a poster frame (the poster IS a video's thumbnail). Off skips the frame capture; the video's pipeline still completes. 
/// * [keepOriginal] - The re-encode failure policy: on, a failed re-encode completes the job quietly and the original stays in place and downloadable; off, the job fails and retries. 
@BuiltValue()
abstract class TranscodingSettingsResponse implements Built<TranscodingSettingsResponse, TranscodingSettingsResponseBuilder> {
  @BuiltValueField(wireName: r'preset')
  TranscodePreset get preset;
  // enum presetEnum {  auto,  1080p,  720p,  480p,  off,  };

  /// How many video re-encodes may run at once: 1 (low load), 2 (the recommended default) or 4 (high load). 
  @BuiltValueField(wireName: r'concurrency')
  TranscodingSettingsResponseConcurrencyEnum get concurrency;
  // enum concurrencyEnum {  1,  2,  4,  };

  /// Whether the video thumbnail job captures a poster frame (the poster IS a video's thumbnail). Off skips the frame capture; the video's pipeline still completes. 
  @BuiltValueField(wireName: r'thumbnailFrame')
  bool get thumbnailFrame;

  /// The re-encode failure policy: on, a failed re-encode completes the job quietly and the original stays in place and downloadable; off, the job fails and retries. 
  @BuiltValueField(wireName: r'keepOriginal')
  bool get keepOriginal;

  TranscodingSettingsResponse._();

  factory TranscodingSettingsResponse([void updates(TranscodingSettingsResponseBuilder b)]) = _$TranscodingSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TranscodingSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TranscodingSettingsResponse> get serializer => _$TranscodingSettingsResponseSerializer();
}

class _$TranscodingSettingsResponseSerializer implements PrimitiveSerializer<TranscodingSettingsResponse> {
  @override
  final Iterable<Type> types = const [TranscodingSettingsResponse, _$TranscodingSettingsResponse];

  @override
  final String wireName = r'TranscodingSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TranscodingSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'preset';
    yield serializers.serialize(
      object.preset,
      specifiedType: const FullType(TranscodePreset),
    );
    yield r'concurrency';
    yield serializers.serialize(
      object.concurrency,
      specifiedType: const FullType(TranscodingSettingsResponseConcurrencyEnum),
    );
    yield r'thumbnailFrame';
    yield serializers.serialize(
      object.thumbnailFrame,
      specifiedType: const FullType(bool),
    );
    yield r'keepOriginal';
    yield serializers.serialize(
      object.keepOriginal,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TranscodingSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TranscodingSettingsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'preset':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(TranscodePreset),
          ) as TranscodePreset;
          result.preset = valueDes;
          break;
        case r'concurrency':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(TranscodingSettingsResponseConcurrencyEnum),
          ) as TranscodingSettingsResponseConcurrencyEnum;
          result.concurrency = valueDes;
          break;
        case r'thumbnailFrame':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.thumbnailFrame = valueDes;
          break;
        case r'keepOriginal':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.keepOriginal = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TranscodingSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TranscodingSettingsResponseBuilder();
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


/// How many video re-encodes may run at once: 1 (low load), 2 (the recommended default) or 4 (high load). 
class TranscodingSettingsResponseConcurrencyEnum extends EnumClass {

  @BuiltValueEnumConst(wireNumber: 1)
  static const TranscodingSettingsResponseConcurrencyEnum number1 = _$transcodingSettingsResponseConcurrencyEnum_number1;
  @BuiltValueEnumConst(wireNumber: 2)
  static const TranscodingSettingsResponseConcurrencyEnum number2 = _$transcodingSettingsResponseConcurrencyEnum_number2;
  @BuiltValueEnumConst(wireNumber: 4)
  static const TranscodingSettingsResponseConcurrencyEnum number4 = _$transcodingSettingsResponseConcurrencyEnum_number4;

  static Serializer<TranscodingSettingsResponseConcurrencyEnum> get serializer => _$transcodingSettingsResponseConcurrencyEnumSerializer;

  const TranscodingSettingsResponseConcurrencyEnum._(String name): super(name);

  static BuiltSet<TranscodingSettingsResponseConcurrencyEnum> get values => _$transcodingSettingsResponseConcurrencyEnumValues;
  static TranscodingSettingsResponseConcurrencyEnum valueOf(String name) => _$transcodingSettingsResponseConcurrencyEnumValueOf(name);
}

