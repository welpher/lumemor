//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/derivative_format.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'image_settings_response.g.dart';

/// The derivative image settings: the sizes and encoding every NEW derivation runs at. Saving them recomputes nothing -- serving copies already on disk stay as they are until the explicit rebuild re-derives them. 
///
/// Properties:
/// * [tileSize] - The timeline tile's bounding box in pixels: 250 is the shipped default, 320 reads sharper on high-density screens. 
/// * [previewSize] - The viewer's inline preview long edge in pixels: 1080 is the shipped default (enough to look at, cheapest to derive), 1440 and 2048 zoom sharper at a bigger encode. 
/// * [format] 
/// * [fullSize] - Whether derivations also produce the full-resolution serving copy. Shipped OFF: the viewer reads the preview, and the full-size copy is disk the family never opens. 
/// * [rawEmbedded] - Whether a RAW source derives from its embedded preview. Shipped ON. 
@BuiltValue()
abstract class ImageSettingsResponse implements Built<ImageSettingsResponse, ImageSettingsResponseBuilder> {
  /// The timeline tile's bounding box in pixels: 250 is the shipped default, 320 reads sharper on high-density screens. 
  @BuiltValueField(wireName: r'tileSize')
  ImageSettingsResponseTileSizeEnum get tileSize;
  // enum tileSizeEnum {  250,  320,  };

  /// The viewer's inline preview long edge in pixels: 1080 is the shipped default (enough to look at, cheapest to derive), 1440 and 2048 zoom sharper at a bigger encode. 
  @BuiltValueField(wireName: r'previewSize')
  ImageSettingsResponsePreviewSizeEnum get previewSize;
  // enum previewSizeEnum {  1080,  1440,  2048,  };

  @BuiltValueField(wireName: r'format')
  DerivativeFormat get format;
  // enum formatEnum {  avif,  webp,  };

  /// Whether derivations also produce the full-resolution serving copy. Shipped OFF: the viewer reads the preview, and the full-size copy is disk the family never opens. 
  @BuiltValueField(wireName: r'fullSize')
  bool get fullSize;

  /// Whether a RAW source derives from its embedded preview. Shipped ON. 
  @BuiltValueField(wireName: r'rawEmbedded')
  bool get rawEmbedded;

  ImageSettingsResponse._();

  factory ImageSettingsResponse([void updates(ImageSettingsResponseBuilder b)]) = _$ImageSettingsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ImageSettingsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ImageSettingsResponse> get serializer => _$ImageSettingsResponseSerializer();
}

class _$ImageSettingsResponseSerializer implements PrimitiveSerializer<ImageSettingsResponse> {
  @override
  final Iterable<Type> types = const [ImageSettingsResponse, _$ImageSettingsResponse];

  @override
  final String wireName = r'ImageSettingsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ImageSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'tileSize';
    yield serializers.serialize(
      object.tileSize,
      specifiedType: const FullType(ImageSettingsResponseTileSizeEnum),
    );
    yield r'previewSize';
    yield serializers.serialize(
      object.previewSize,
      specifiedType: const FullType(ImageSettingsResponsePreviewSizeEnum),
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
    ImageSettingsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ImageSettingsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'tileSize':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ImageSettingsResponseTileSizeEnum),
          ) as ImageSettingsResponseTileSizeEnum;
          result.tileSize = valueDes;
          break;
        case r'previewSize':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ImageSettingsResponsePreviewSizeEnum),
          ) as ImageSettingsResponsePreviewSizeEnum;
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
  ImageSettingsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ImageSettingsResponseBuilder();
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


/// The timeline tile's bounding box in pixels: 250 is the shipped default, 320 reads sharper on high-density screens. 
class ImageSettingsResponseTileSizeEnum extends EnumClass {

  @BuiltValueEnumConst(wireNumber: 250)
  static const ImageSettingsResponseTileSizeEnum number250 = _$imageSettingsResponseTileSizeEnum_number250;
  @BuiltValueEnumConst(wireNumber: 320)
  static const ImageSettingsResponseTileSizeEnum number320 = _$imageSettingsResponseTileSizeEnum_number320;

  static Serializer<ImageSettingsResponseTileSizeEnum> get serializer => _$imageSettingsResponseTileSizeEnumSerializer;

  const ImageSettingsResponseTileSizeEnum._(String name): super(name);

  static BuiltSet<ImageSettingsResponseTileSizeEnum> get values => _$imageSettingsResponseTileSizeEnumValues;
  static ImageSettingsResponseTileSizeEnum valueOf(String name) => _$imageSettingsResponseTileSizeEnumValueOf(name);
}

/// The viewer's inline preview long edge in pixels: 1080 is the shipped default (enough to look at, cheapest to derive), 1440 and 2048 zoom sharper at a bigger encode. 
class ImageSettingsResponsePreviewSizeEnum extends EnumClass {

  @BuiltValueEnumConst(wireNumber: 1080)
  static const ImageSettingsResponsePreviewSizeEnum number1080 = _$imageSettingsResponsePreviewSizeEnum_number1080;
  @BuiltValueEnumConst(wireNumber: 1440)
  static const ImageSettingsResponsePreviewSizeEnum number1440 = _$imageSettingsResponsePreviewSizeEnum_number1440;
  @BuiltValueEnumConst(wireNumber: 2048)
  static const ImageSettingsResponsePreviewSizeEnum number2048 = _$imageSettingsResponsePreviewSizeEnum_number2048;

  static Serializer<ImageSettingsResponsePreviewSizeEnum> get serializer => _$imageSettingsResponsePreviewSizeEnumSerializer;

  const ImageSettingsResponsePreviewSizeEnum._(String name): super(name);

  static BuiltSet<ImageSettingsResponsePreviewSizeEnum> get values => _$imageSettingsResponsePreviewSizeEnumValues;
  static ImageSettingsResponsePreviewSizeEnum valueOf(String name) => _$imageSettingsResponsePreviewSizeEnumValueOf(name);
}

