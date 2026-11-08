//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/album_share_entry.dart';
import 'package:lumemor_api/src/model/album_item.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_detail.g.dart';

/// One album's detail as the calling user sees it: the metadata, the caller's role, the live member count and one keyset page of the live items. 
///
/// Properties:
/// * [id] - Identifier of the album.
/// * [title] - The album's display title.
/// * [description] - The album's free-text description (empty when unset).
/// * [type] - The album's type; `sync` is the caller's own default album -- a read-only, system-managed detail (no write affordance applies, the display name is the client's localized one keyed by the type). Another account's sync album never reaches this shape: it answers the single-shape 404. 
/// * [locked] - The owner's lock (metadata to a viewer).
/// * [coverAssetId] - The pinned cover asset, or null when unset -- the raw column; the wall derives its collage fallback, the detail carries the pin itself. 
/// * [rootPath] - The album's source directory: the RESOLVED absolute path the album row stores, present exactly on `directory` albums (the edit dialog's read-only source line; the property patch carries no field for it -- the source is pinned at creation). Virtual and sync albums omit it. 
/// * [callerRole] - The caller's relation to this album: the owner holds the write affordances, a viewer is read-only. 
/// * [itemCount] - The album's live member count, the same predicate as the items below. 
/// * [items] - One page of the album's live members, `takenAt` descending with the asset id as the tiebreak. 
/// * [nextCursor] - The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page. 
/// * [shares] - The album's grantee list, riding ONLY the owner's view: one entry per `album_shares` row in the grant order (`grantedAt` ascending) -- the share management dialog's list. The owner's response always carries the field, an empty array when the album holds no grants; viewer responses never carry it at all. 
/// * [inTimeline] - The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly when the album is excludable for the caller -- an own `directory` album or an album shared to them; an own `virtual` album omits the field (it never feeds the caller's own timeline), and the caller's own `sync` album omits it too (the core stream is never excludable). The value is the preference row's switch, true when no row exists (the default). The preference scopes the aggregated timeline only: it never filters this album's own items, whatever the switch says. 
@BuiltValue()
abstract class AlbumDetail implements Built<AlbumDetail, AlbumDetailBuilder> {
  /// Identifier of the album.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The album's display title.
  @BuiltValueField(wireName: r'title')
  String get title;

  /// The album's free-text description (empty when unset).
  @BuiltValueField(wireName: r'description')
  String get description;

  /// The album's type; `sync` is the caller's own default album -- a read-only, system-managed detail (no write affordance applies, the display name is the client's localized one keyed by the type). Another account's sync album never reaches this shape: it answers the single-shape 404. 
  @BuiltValueField(wireName: r'type')
  AlbumDetailTypeEnum get type;
  // enum typeEnum {  virtual,  directory,  sync,  };

  /// The owner's lock (metadata to a viewer).
  @BuiltValueField(wireName: r'locked')
  bool get locked;

  /// The pinned cover asset, or null when unset -- the raw column; the wall derives its collage fallback, the detail carries the pin itself. 
  @BuiltValueField(wireName: r'coverAssetId')
  String? get coverAssetId;

  /// The album's source directory: the RESOLVED absolute path the album row stores, present exactly on `directory` albums (the edit dialog's read-only source line; the property patch carries no field for it -- the source is pinned at creation). Virtual and sync albums omit it. 
  @BuiltValueField(wireName: r'rootPath')
  String? get rootPath;

  /// The caller's relation to this album: the owner holds the write affordances, a viewer is read-only. 
  @BuiltValueField(wireName: r'callerRole')
  AlbumDetailCallerRoleEnum get callerRole;
  // enum callerRoleEnum {  owner,  viewer,  };

  /// The album's live member count, the same predicate as the items below. 
  @BuiltValueField(wireName: r'itemCount')
  int get itemCount;

  /// One page of the album's live members, `takenAt` descending with the asset id as the tiebreak. 
  @BuiltValueField(wireName: r'items')
  BuiltList<AlbumItem> get items;

  /// The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page. 
  @BuiltValueField(wireName: r'nextCursor')
  String? get nextCursor;

  /// The album's grantee list, riding ONLY the owner's view: one entry per `album_shares` row in the grant order (`grantedAt` ascending) -- the share management dialog's list. The owner's response always carries the field, an empty array when the album holds no grants; viewer responses never carry it at all. 
  @BuiltValueField(wireName: r'shares')
  BuiltList<AlbumShareEntry>? get shares;

  /// The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly when the album is excludable for the caller -- an own `directory` album or an album shared to them; an own `virtual` album omits the field (it never feeds the caller's own timeline), and the caller's own `sync` album omits it too (the core stream is never excludable). The value is the preference row's switch, true when no row exists (the default). The preference scopes the aggregated timeline only: it never filters this album's own items, whatever the switch says. 
  @BuiltValueField(wireName: r'inTimeline')
  bool? get inTimeline;

  AlbumDetail._();

  factory AlbumDetail([void updates(AlbumDetailBuilder b)]) = _$AlbumDetail;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumDetailBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumDetail> get serializer => _$AlbumDetailSerializer();
}

class _$AlbumDetailSerializer implements PrimitiveSerializer<AlbumDetail> {
  @override
  final Iterable<Type> types = const [AlbumDetail, _$AlbumDetail];

  @override
  final String wireName = r'AlbumDetail';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'title';
    yield serializers.serialize(
      object.title,
      specifiedType: const FullType(String),
    );
    yield r'description';
    yield serializers.serialize(
      object.description,
      specifiedType: const FullType(String),
    );
    yield r'type';
    yield serializers.serialize(
      object.type,
      specifiedType: const FullType(AlbumDetailTypeEnum),
    );
    yield r'locked';
    yield serializers.serialize(
      object.locked,
      specifiedType: const FullType(bool),
    );
    yield r'coverAssetId';
    yield object.coverAssetId == null ? null : serializers.serialize(
      object.coverAssetId,
      specifiedType: const FullType.nullable(String),
    );
    if (object.rootPath != null) {
      yield r'rootPath';
      yield serializers.serialize(
        object.rootPath,
        specifiedType: const FullType(String),
      );
    }
    yield r'callerRole';
    yield serializers.serialize(
      object.callerRole,
      specifiedType: const FullType(AlbumDetailCallerRoleEnum),
    );
    yield r'itemCount';
    yield serializers.serialize(
      object.itemCount,
      specifiedType: const FullType(int),
    );
    yield r'items';
    yield serializers.serialize(
      object.items,
      specifiedType: const FullType(BuiltList, [FullType(AlbumItem)]),
    );
    if (object.nextCursor != null) {
      yield r'nextCursor';
      yield serializers.serialize(
        object.nextCursor,
        specifiedType: const FullType(String),
      );
    }
    if (object.shares != null) {
      yield r'shares';
      yield serializers.serialize(
        object.shares,
        specifiedType: const FullType(BuiltList, [FullType(AlbumShareEntry)]),
      );
    }
    if (object.inTimeline != null) {
      yield r'inTimeline';
      yield serializers.serialize(
        object.inTimeline,
        specifiedType: const FullType(bool),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumDetailBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'id':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.id = valueDes;
          break;
        case r'title':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.title = valueDes;
          break;
        case r'description':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.description = valueDes;
          break;
        case r'type':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AlbumDetailTypeEnum),
          ) as AlbumDetailTypeEnum;
          result.type = valueDes;
          break;
        case r'locked':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.locked = valueDes;
          break;
        case r'coverAssetId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.coverAssetId = valueDes;
          break;
        case r'rootPath':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.rootPath = valueDes;
          break;
        case r'callerRole':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AlbumDetailCallerRoleEnum),
          ) as AlbumDetailCallerRoleEnum;
          result.callerRole = valueDes;
          break;
        case r'itemCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.itemCount = valueDes;
          break;
        case r'items':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(AlbumItem)]),
          ) as BuiltList<AlbumItem>;
          result.items.replace(valueDes);
          break;
        case r'nextCursor':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.nextCursor = valueDes;
          break;
        case r'shares':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(AlbumShareEntry)]),
          ) as BuiltList<AlbumShareEntry>?;
          if (valueDes == null) continue;
          result.shares.replace(valueDes);
          break;
        case r'inTimeline':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.inTimeline = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumDetail deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumDetailBuilder();
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


/// The album's type; `sync` is the caller's own default album -- a read-only, system-managed detail (no write affordance applies, the display name is the client's localized one keyed by the type). Another account's sync album never reaches this shape: it answers the single-shape 404. 
class AlbumDetailTypeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'virtual')
  static const AlbumDetailTypeEnum virtual = _$albumDetailTypeEnum_virtual;
  @BuiltValueEnumConst(wireName: r'directory')
  static const AlbumDetailTypeEnum directory = _$albumDetailTypeEnum_directory;
  @BuiltValueEnumConst(wireName: r'sync')
  static const AlbumDetailTypeEnum sync_ = _$albumDetailTypeEnum_sync_;

  static Serializer<AlbumDetailTypeEnum> get serializer => _$albumDetailTypeEnumSerializer;

  const AlbumDetailTypeEnum._(String name): super(name);

  static BuiltSet<AlbumDetailTypeEnum> get values => _$albumDetailTypeEnumValues;
  static AlbumDetailTypeEnum valueOf(String name) => _$albumDetailTypeEnumValueOf(name);
}

/// The caller's relation to this album: the owner holds the write affordances, a viewer is read-only. 
class AlbumDetailCallerRoleEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'owner')
  static const AlbumDetailCallerRoleEnum owner = _$albumDetailCallerRoleEnum_owner;
  @BuiltValueEnumConst(wireName: r'viewer')
  static const AlbumDetailCallerRoleEnum viewer = _$albumDetailCallerRoleEnum_viewer;

  static Serializer<AlbumDetailCallerRoleEnum> get serializer => _$albumDetailCallerRoleEnumSerializer;

  const AlbumDetailCallerRoleEnum._(String name): super(name);

  static BuiltSet<AlbumDetailCallerRoleEnum> get values => _$albumDetailCallerRoleEnumValues;
  static AlbumDetailCallerRoleEnum valueOf(String name) => _$albumDetailCallerRoleEnumValueOf(name);
}

