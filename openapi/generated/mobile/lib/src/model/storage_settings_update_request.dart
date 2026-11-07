//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'storage_settings_update_request.g.dart';

/// Payload of `PUT /admin/settings/storage`.
///
/// Properties:
/// * [template] - The template to validate and save. Must compile strictly, render over the sample asset set, and produce safe structure; 1 to 512 runes. 
@BuiltValue()
abstract class StorageSettingsUpdateRequest implements Built<StorageSettingsUpdateRequest, StorageSettingsUpdateRequestBuilder> {
  /// The template to validate and save. Must compile strictly, render over the sample asset set, and produce safe structure; 1 to 512 runes. 
  @BuiltValueField(wireName: r'template')
  String get template;

  StorageSettingsUpdateRequest._();

  factory StorageSettingsUpdateRequest([void updates(StorageSettingsUpdateRequestBuilder b)]) = _$StorageSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(StorageSettingsUpdateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<StorageSettingsUpdateRequest> get serializer => _$StorageSettingsUpdateRequestSerializer();
}

class _$StorageSettingsUpdateRequestSerializer implements PrimitiveSerializer<StorageSettingsUpdateRequest> {
  @override
  final Iterable<Type> types = const [StorageSettingsUpdateRequest, _$StorageSettingsUpdateRequest];

  @override
  final String wireName = r'StorageSettingsUpdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    StorageSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'template';
    yield serializers.serialize(
      object.template,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    StorageSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required StorageSettingsUpdateRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'template':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.template = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  StorageSettingsUpdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = StorageSettingsUpdateRequestBuilder();
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


