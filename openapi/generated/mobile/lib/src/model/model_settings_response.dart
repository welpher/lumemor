//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_settings_response.g.dart';

/// The model store's policy of `GET /admin/settings/models` and the successful `PUT`. 
///
/// Properties:
/// * [autoDownload] - Whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own. It never fills an unconfigured slot: the first-run download stays the administrator's own act. 
@BuiltValue()
abstract class ModelSettingsResponse implements Built<ModelSettingsResponse, ModelSettingsResponseBuilder> {
  /// Whether an AI job whose slot is configured but whose weights are missing may start the model group's download on its own. It never fills an unconfigured slot: the first-run download stays the administrator's own act. 
  @BuiltValueField(wireName: r'autoDownload')
  bool get autoDownload;

  ModelSettingsResponse._();

  factory ModelSettingsResponse([void updates(ModelSettingsResponseBuilder b)]) = _$ModelSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelSettingsResponse> get serializer => _$ModelSettingsResponseSerializer();
}

class _$ModelSettingsResponseSerializer implements PrimitiveSerializer<ModelSettingsResponse> {
  @override
  final Iterable<Type> types = const [ModelSettingsResponse, _$ModelSettingsResponse];

  @override
  final String wireName = r'ModelSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelSettingsResponse object, {
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
    ModelSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelSettingsResponseBuilder result,
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
  ModelSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelSettingsResponseBuilder();
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


