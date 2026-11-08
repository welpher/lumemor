//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/transcode_preset.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'transcoding_settings_update_request.g.dart';

/// Payload of `PUT /admin/settings/transcoding`.
///
/// Properties:
/// * [preset] 
/// * [concurrency] - The concurrency bound to save: 1, 2 or 4.
/// * [thumbnailFrame] - The poster-frame capture switch to save.
/// * [keepOriginal] - The failure policy to save.
@BuiltValue()
abstract class TranscodingSettingsUpdateRequest implements Built<TranscodingSettingsUpdateRequest, TranscodingSettingsUpdateRequestBuilder> {
  @BuiltValueField(wireName: r'preset')
  TranscodePreset get preset;
  // enum presetEnum {  auto,  1080p,  720p,  480p,  off,  };

  /// The concurrency bound to save: 1, 2 or 4.
  @BuiltValueField(wireName: r'concurrency')
  TranscodingSettingsUpdateRequestConcurrencyEnum get concurrency;
  // enum concurrencyEnum {  1,  2,  4,  };

  /// The poster-frame capture switch to save.
  @BuiltValueField(wireName: r'thumbnailFrame')
  bool get thumbnailFrame;

  /// The failure policy to save.
  @BuiltValueField(wireName: r'keepOriginal')
  bool get keepOriginal;

  TranscodingSettingsUpdateRequest._();

  factory TranscodingSettingsUpdateRequest([void updates(TranscodingSettingsUpdateRequestBuilder b)]) = _$TranscodingSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TranscodingSettingsUpdateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TranscodingSettingsUpdateRequest> get serializer => _$TranscodingSettingsUpdateRequestSerializer();
}

class _$TranscodingSettingsUpdateRequestSerializer implements PrimitiveSerializer<TranscodingSettingsUpdateRequest> {
  @override
  final Iterable<Type> types = const [TranscodingSettingsUpdateRequest, _$TranscodingSettingsUpdateRequest];

  @override
  final String wireName = r'TranscodingSettingsUpdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TranscodingSettingsUpdateRequest object, {
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
      specifiedType: const FullType(TranscodingSettingsUpdateRequestConcurrencyEnum),
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
    TranscodingSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TranscodingSettingsUpdateRequestBuilder result,
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
            specifiedType: const FullType(TranscodingSettingsUpdateRequestConcurrencyEnum),
          ) as TranscodingSettingsUpdateRequestConcurrencyEnum;
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
  TranscodingSettingsUpdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TranscodingSettingsUpdateRequestBuilder();
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


/// The concurrency bound to save: 1, 2 or 4.
class TranscodingSettingsUpdateRequestConcurrencyEnum extends EnumClass {

  @BuiltValueEnumConst(wireNumber: 1)
  static const TranscodingSettingsUpdateRequestConcurrencyEnum number1 = _$transcodingSettingsUpdateRequestConcurrencyEnum_number1;
  @BuiltValueEnumConst(wireNumber: 2)
  static const TranscodingSettingsUpdateRequestConcurrencyEnum number2 = _$transcodingSettingsUpdateRequestConcurrencyEnum_number2;
  @BuiltValueEnumConst(wireNumber: 4)
  static const TranscodingSettingsUpdateRequestConcurrencyEnum number4 = _$transcodingSettingsUpdateRequestConcurrencyEnum_number4;

  static Serializer<TranscodingSettingsUpdateRequestConcurrencyEnum> get serializer => _$transcodingSettingsUpdateRequestConcurrencyEnumSerializer;

  const TranscodingSettingsUpdateRequestConcurrencyEnum._(String name): super(name);

  static BuiltSet<TranscodingSettingsUpdateRequestConcurrencyEnum> get values => _$transcodingSettingsUpdateRequestConcurrencyEnumValues;
  static TranscodingSettingsUpdateRequestConcurrencyEnum valueOf(String name) => _$transcodingSettingsUpdateRequestConcurrencyEnumValueOf(name);
}

