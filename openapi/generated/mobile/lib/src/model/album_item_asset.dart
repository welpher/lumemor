//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_item_asset.g.dart';

/// The minimal context of one album member asset: the lines the detail grid renders per tile. 
///
/// Properties:
/// * [takenAt] - When the asset was taken (RFC 3339 UTC).
/// * [originalFilename] - The file name the asset was uploaded under.
/// * [kind] - The asset's media kind.
/// * [thumbnail] - URL of the asset's tile-size serving copy: the asset thumbnail endpoint `GET /api/v1/assets/{assetId}/thumbnail?size=tile`. 
@BuiltValue()
abstract class AlbumItemAsset implements Built<AlbumItemAsset, AlbumItemAssetBuilder> {
  /// When the asset was taken (RFC 3339 UTC).
  @BuiltValueField(wireName: r'takenAt')
  DateTime get takenAt;

  /// The file name the asset was uploaded under.
  @BuiltValueField(wireName: r'originalFilename')
  String get originalFilename;

  /// The asset's media kind.
  @BuiltValueField(wireName: r'kind')
  AlbumItemAssetKindEnum get kind;
  // enum kindEnum {  image,  video,  };

  /// URL of the asset's tile-size serving copy: the asset thumbnail endpoint `GET /api/v1/assets/{assetId}/thumbnail?size=tile`. 
  @BuiltValueField(wireName: r'thumbnail')
  String get thumbnail;

  AlbumItemAsset._();

  factory AlbumItemAsset([void updates(AlbumItemAssetBuilder b)]) = _$AlbumItemAsset;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumItemAssetBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumItemAsset> get serializer => _$AlbumItemAssetSerializer();
}

class _$AlbumItemAssetSerializer implements PrimitiveSerializer<AlbumItemAsset> {
  @override
  final Iterable<Type> types = const [AlbumItemAsset, _$AlbumItemAsset];

  @override
  final String wireName = r'AlbumItemAsset';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumItemAsset object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'takenAt';
    yield serializers.serialize(
      object.takenAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'originalFilename';
    yield serializers.serialize(
      object.originalFilename,
      specifiedType: const FullType(String),
    );
    yield r'kind';
    yield serializers.serialize(
      object.kind,
      specifiedType: const FullType(AlbumItemAssetKindEnum),
    );
    yield r'thumbnail';
    yield serializers.serialize(
      object.thumbnail,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumItemAsset object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumItemAssetBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'takenAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.takenAt = valueDes;
          break;
        case r'originalFilename':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.originalFilename = valueDes;
          break;
        case r'kind':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AlbumItemAssetKindEnum),
          ) as AlbumItemAssetKindEnum;
          result.kind = valueDes;
          break;
        case r'thumbnail':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.thumbnail = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumItemAsset deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumItemAssetBuilder();
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


/// The asset's media kind.
class AlbumItemAssetKindEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'image')
  static const AlbumItemAssetKindEnum image = _$albumItemAssetKindEnum_image;
  @BuiltValueEnumConst(wireName: r'video')
  static const AlbumItemAssetKindEnum video = _$albumItemAssetKindEnum_video;

  static Serializer<AlbumItemAssetKindEnum> get serializer => _$albumItemAssetKindEnumSerializer;

  const AlbumItemAssetKindEnum._(String name): super(name);

  static BuiltSet<AlbumItemAssetKindEnum> get values => _$albumItemAssetKindEnumValues;
  static AlbumItemAssetKindEnum valueOf(String name) => _$albumItemAssetKindEnumValueOf(name);
}

