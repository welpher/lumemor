//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'storage_settings_response.g.dart';

/// The storage settings of `GET /admin/settings/storage` and the successful `PUT`: the instance's upload path template, the sample path it renders, and the built-in presets. 
///
/// Properties:
/// * [template] - The template new uploads render their placement from. A fresh instance (or a corrupt row) answers the default `{{y}}/{{MM}}/{{filename}}`. 
/// * [samplePath] - The template rendered over the fixed sample asset with the extension appended, as a stored relative path -- exactly the placement a real upload of the sample file would land at. 
/// * [presets] - The built-in preset templates, in the settings section's dropdown order: date nestings, album layouts, camera groupings, the type grouping, the asset-id fallback and the bare file name. 
@BuiltValue()
abstract class StorageSettingsResponse implements Built<StorageSettingsResponse, StorageSettingsResponseBuilder> {
  /// The template new uploads render their placement from. A fresh instance (or a corrupt row) answers the default `{{y}}/{{MM}}/{{filename}}`. 
  @BuiltValueField(wireName: r'template')
  String get template;

  /// The template rendered over the fixed sample asset with the extension appended, as a stored relative path -- exactly the placement a real upload of the sample file would land at. 
  @BuiltValueField(wireName: r'samplePath')
  String get samplePath;

  /// The built-in preset templates, in the settings section's dropdown order: date nestings, album layouts, camera groupings, the type grouping, the asset-id fallback and the bare file name. 
  @BuiltValueField(wireName: r'presets')
  BuiltList<String> get presets;

  StorageSettingsResponse._();

  factory StorageSettingsResponse([void updates(StorageSettingsResponseBuilder b)]) = _$StorageSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(StorageSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<StorageSettingsResponse> get serializer => _$StorageSettingsResponseSerializer();
}

class _$StorageSettingsResponseSerializer implements PrimitiveSerializer<StorageSettingsResponse> {
  @override
  final Iterable<Type> types = const [StorageSettingsResponse, _$StorageSettingsResponse];

  @override
  final String wireName = r'StorageSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    StorageSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'template';
    yield serializers.serialize(
      object.template,
      specifiedType: const FullType(String),
    );
    yield r'samplePath';
    yield serializers.serialize(
      object.samplePath,
      specifiedType: const FullType(String),
    );
    yield r'presets';
    yield serializers.serialize(
      object.presets,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    StorageSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required StorageSettingsResponseBuilder result,
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
        case r'samplePath':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.samplePath = valueDes;
          break;
        case r'presets':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.presets.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  StorageSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = StorageSettingsResponseBuilder();
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


