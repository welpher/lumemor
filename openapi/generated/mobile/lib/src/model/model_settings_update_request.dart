//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_settings_update_request.g.dart';

/// Payload of `PUT /admin/settings/models`.
///
/// Properties:
/// * [autoDownload] - The policy to save.
@BuiltValue()
abstract class ModelSettingsUpdateRequest implements Built<ModelSettingsUpdateRequest, ModelSettingsUpdateRequestBuilder> {
  /// The policy to save.
  @BuiltValueField(wireName: r'autoDownload')
  bool get autoDownload;

  ModelSettingsUpdateRequest._();

  factory ModelSettingsUpdateRequest([void updates(ModelSettingsUpdateRequestBuilder b)]) = _$ModelSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelSettingsUpdateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelSettingsUpdateRequest> get serializer => _$ModelSettingsUpdateRequestSerializer();
}

class _$ModelSettingsUpdateRequestSerializer implements PrimitiveSerializer<ModelSettingsUpdateRequest> {
  @override
  final Iterable<Type> types = const [ModelSettingsUpdateRequest, _$ModelSettingsUpdateRequest];

  @override
  final String wireName = r'ModelSettingsUpdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'autoDownload';
    yield serializers.serialize(
      object.autoDownload,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelSettingsUpdateRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'autoDownload':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.autoDownload = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelSettingsUpdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelSettingsUpdateRequestBuilder();
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


