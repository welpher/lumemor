//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/derivative_format.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'image_settings_update_request.g.dart';

/// The image settings save. Every field is contract-required: the PUT replaces the document wholesale, so an omitted field would silently reset it. 
///
/// Properties:
/// * [tileSize] 
/// * [previewSize] 
/// * [format] 
/// * [fullSize] 
/// * [rawEmbedded] 
@BuiltValue()
abstract class ImageSettingsUpdateRequest implements Built<ImageSettingsUpdateRequest, ImageSettingsUpdateRequestBuilder> {
  @BuiltValueField(wireName: r'tileSize')
  ImageSettingsUpdateRequestTileSizeEnum get tileSize;
  // enum tileSizeEnum {  250,  320,  };

  @BuiltValueField(wireName: r'previewSize')
  ImageSettingsUpdateRequestPreviewSizeEnum get previewSize;
  // enum previewSizeEnum {  1080,  1440,  2048,  };

  @BuiltValueField(wireName: r'format')
  DerivativeFormat get format;
  // enum formatEnum {  avif,  webp,  };

  @BuiltValueField(wireName: r'fullSize')
  bool get fullSize;

  @BuiltValueField(wireName: r'rawEmbedded')
  bool get rawEmbedded;

  ImageSettingsUpdateRequest._();

  factory ImageSettingsUpdateRequest([void updates(ImageSettingsUpdateRequestBuilder b)]) = _$ImageSettingsUpdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ImageSettingsUpdateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ImageSettingsUpdateRequest> get serializer => _$ImageSettingsUpdateRequestSerializer();
}

class _$ImageSettingsUpdateRequestSerializer implements PrimitiveSerializer<ImageSettingsUpdateRequest> {
  @override
  final Iterable<Type> types = const [ImageSettingsUpdateRequest, _$ImageSettingsUpdateRequest];

  @override
  final String wireName = r'ImageSettingsUpdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ImageSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'tileSize';
    yield serializers.serialize(
      object.tileSize,
      specifiedType: const FullType(ImageSettingsUpdateRequestTileSizeEnum),
    );
    yield r'previewSize';
    yield serializers.serialize(
      object.previewSize,
      specifiedType: const FullType(ImageSettingsUpdateRequestPreviewSizeEnum),
    );
    yield r'format';
    yield serializers.serialize(
      object.format,
      specifiedType: const FullType(DerivativeFormat),
    );
    yield r'fullSize';
    yield serializers.serialize(
      object.fullSize,
      specifiedType: const FullType(bool),
    );
    yield r'rawEmbedded';
    yield serializers.serialize(
      object.rawEmbedded,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ImageSettingsUpdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ImageSettingsUpdateRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'tileSize':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ImageSettingsUpdateRequestTileSizeEnum),
          ) as ImageSettingsUpdateRequestTileSizeEnum;
          result.tileSize = valueDes;
          break;
        case r'previewSize':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ImageSettingsUpdateRequestPreviewSizeEnum),
          ) as ImageSettingsUpdateRequestPreviewSizeEnum;
          result.previewSize = valueDes;
          break;
        case r'format':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DerivativeFormat),
          ) as DerivativeFormat;
          result.format = valueDes;
          break;
        case r'fullSize':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.fullSize = valueDes;
          break;
        case r'rawEmbedded':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.rawEmbedded = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ImageSettingsUpdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ImageSettingsUpdateRequestBuilder();
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


class ImageSettingsUpdateRequestTileSizeEnum extends EnumClass {

  @BuiltValueEnumConst(wireNumber: 250)
  static const ImageSettingsUpdateRequestTileSizeEnum number250 = _$imageSettingsUpdateRequestTileSizeEnum_number250;
  @BuiltValueEnumConst(wireNumber: 320)
  static const ImageSettingsUpdateRequestTileSizeEnum number320 = _$imageSettingsUpdateRequestTileSizeEnum_number320;

  static Serializer<ImageSettingsUpdateRequestTileSizeEnum> get serializer => _$imageSettingsUpdateRequestTileSizeEnumSerializer;

  const ImageSettingsUpdateRequestTileSizeEnum._(String name): super(name);

  static BuiltSet<ImageSettingsUpdateRequestTileSizeEnum> get values => _$imageSettingsUpdateRequestTileSizeEnumValues;
  static ImageSettingsUpdateRequestTileSizeEnum valueOf(String name) => _$imageSettingsUpdateRequestTileSizeEnumValueOf(name);
}

class ImageSettingsUpdateRequestPreviewSizeEnum extends EnumClass {

  @BuiltValueEnumConst(wireNumber: 1080)
  static const ImageSettingsUpdateRequestPreviewSizeEnum number1080 = _$imageSettingsUpdateRequestPreviewSizeEnum_number1080;
  @BuiltValueEnumConst(wireNumber: 1440)
  static const ImageSettingsUpdateRequestPreviewSizeEnum number1440 = _$imageSettingsUpdateRequestPreviewSizeEnum_number1440;
  @BuiltValueEnumConst(wireNumber: 2048)
  static const ImageSettingsUpdateRequestPreviewSizeEnum number2048 = _$imageSettingsUpdateRequestPreviewSizeEnum_number2048;

  static Serializer<ImageSettingsUpdateRequestPreviewSizeEnum> get serializer => _$imageSettingsUpdateRequestPreviewSizeEnumSerializer;

  const ImageSettingsUpdateRequestPreviewSizeEnum._(String name): super(name);

  static BuiltSet<ImageSettingsUpdateRequestPreviewSizeEnum> get values => _$imageSettingsUpdateRequestPreviewSizeEnumValues;
  static ImageSettingsUpdateRequestPreviewSizeEnum valueOf(String name) => _$imageSettingsUpdateRequestPreviewSizeEnumValueOf(name);
}

