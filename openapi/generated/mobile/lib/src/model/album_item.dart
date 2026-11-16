//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/album_item_asset.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_item.g.dart';

/// One member of the album: the membership's identity and the minimal asset context the detail grid renders. 
///
/// Properties:
/// * [assetId] - Identifier of the member asset.
/// * [addedAt] - When the membership row was added to the album (RFC 3339 UTC). 
/// * [asset] - The minimal context of the member asset.
@BuiltValue()
abstract class AlbumItem implements Built<AlbumItem, AlbumItemBuilder> {
  /// Identifier of the member asset.
  @BuiltValueField(wireName: r'assetId')
  String get assetId;

  /// When the membership row was added to the album (RFC 3339 UTC). 
  @BuiltValueField(wireName: r'addedAt')
  DateTime get addedAt;

  /// The minimal context of the member asset.
  @BuiltValueField(wireName: r'asset')
  AlbumItemAsset get asset;

  AlbumItem._();

  factory AlbumItem([void updates(AlbumItemBuilder b)]) = _$AlbumItem;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumItemBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumItem> get serializer => _$AlbumItemSerializer();
}

class _$AlbumItemSerializer implements PrimitiveSerializer<AlbumItem> {
  @override
  final Iterable<Type> types = const [AlbumItem, _$AlbumItem];

  @override
  final String wireName = r'AlbumItem';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumItem object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'assetId';
    yield serializers.serialize(
      object.assetId,
      specifiedType: const FullType(String),
    );
    yield r'addedAt';
    yield serializers.serialize(
      object.addedAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'asset';
    yield serializers.serialize(
      object.asset,
      specifiedType: const FullType(AlbumItemAsset),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumItem object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumItemBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'assetId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.assetId = valueDes;
          break;
        case r'addedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.addedAt = valueDes;
          break;
        case r'asset':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AlbumItemAsset),
          ) as AlbumItemAsset;
          result.asset.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumItem deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumItemBuilder();
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


