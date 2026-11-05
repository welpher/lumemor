//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/album_owner_ref.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_summary.g.dart';

/// One album card of the wall: the metadata, the live member count and the collage source, plus the `callerRole` discriminator the web client sections the wall by. Shared items carry `owner`, the honest attribution of another account's album; own items omit it. 
///
/// Properties:
/// * [id] - Identifier of the album.
/// * [title] - The album's display title.
/// * [description] - The album's free-text description (empty when unset).
/// * [type] - The album's type. `virtual` albums are curated by their owner; `directory` albums mirror a filesystem path. Sync albums never reach this shape. 
/// * [locked] - The owner's lock: a locked album refuses its owner's curation and property writes until unlocked. It is metadata to a viewer (viewers are read-only regardless). 
/// * [isShared] - True when the album carries at least one `album_shares` row -- someone besides the owner can see it. Always true on shared-group items by construction. 
/// * [coverAssetIds] - The collage source of the wall card: the pinned cover asset as a single-element list when the owner set one, otherwise the album's most recent live member assets (up to four, `takenAt` descending, the asset id as the tiebreak). Empty for an album with no live members. 
/// * [itemCount] - The album's live member count: trashed, hidden and Live Photo component rows never count (A5 -- a hidden asset is invisible to the whole family). 
/// * [updatedAt] - When the album row last changed (RFC 3339 UTC).
/// * [callerRole] - The caller's relation to this album: `owner` items belong to the own group, `viewer` items to the shared group. The web client sections the wall off this discriminator. 
/// * [owner] - The owner attribution of a shared item: the account that owns the album. Present exactly when `callerRole` is `viewer`; own items omit it (the owner is the caller). 
@BuiltValue()
abstract class AlbumSummary implements Built<AlbumSummary, AlbumSummaryBuilder> {
  /// Identifier of the album.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The album's display title.
  @BuiltValueField(wireName: r'title')
  String get title;

  /// The album's free-text description (empty when unset).
  @BuiltValueField(wireName: r'description')
  String get description;

  /// The album's type. `virtual` albums are curated by their owner; `directory` albums mirror a filesystem path. Sync albums never reach this shape. 
  @BuiltValueField(wireName: r'type')
  AlbumSummaryTypeEnum get type;
  // enum typeEnum {  virtual,  directory,  };

  /// The owner's lock: a locked album refuses its owner's curation and property writes until unlocked. It is metadata to a viewer (viewers are read-only regardless). 
  @BuiltValueField(wireName: r'locked')
  bool get locked;

  /// True when the album carries at least one `album_shares` row -- someone besides the owner can see it. Always true on shared-group items by construction. 
  @BuiltValueField(wireName: r'isShared')
  bool get isShared;

  /// The collage source of the wall card: the pinned cover asset as a single-element list when the owner set one, otherwise the album's most recent live member assets (up to four, `takenAt` descending, the asset id as the tiebreak). Empty for an album with no live members. 
  @BuiltValueField(wireName: r'coverAssetIds')
  BuiltList<String> get coverAssetIds;

  /// The album's live member count: trashed, hidden and Live Photo component rows never count (A5 -- a hidden asset is invisible to the whole family). 
  @BuiltValueField(wireName: r'itemCount')
  int get itemCount;

  /// When the album row last changed (RFC 3339 UTC).
  @BuiltValueField(wireName: r'updatedAt')
  DateTime get updatedAt;

  /// The caller's relation to this album: `owner` items belong to the own group, `viewer` items to the shared group. The web client sections the wall off this discriminator. 
  @BuiltValueField(wireName: r'callerRole')
  AlbumSummaryCallerRoleEnum get callerRole;
  // enum callerRoleEnum {  owner,  viewer,  };

  /// The owner attribution of a shared item: the account that owns the album. Present exactly when `callerRole` is `viewer`; own items omit it (the owner is the caller). 
  @BuiltValueField(wireName: r'owner')
  AlbumOwnerRef? get owner;

  AlbumSummary._();

  factory AlbumSummary([void updates(AlbumSummaryBuilder b)]) = _$AlbumSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumSummary> get serializer => _$AlbumSummarySerializer();
}

class _$AlbumSummarySerializer implements PrimitiveSerializer<AlbumSummary> {
  @override
  final Iterable<Type> types = const [AlbumSummary, _$AlbumSummary];

  @override
  final String wireName = r'AlbumSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumSummary object, {
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
      specifiedType: const FullType(AlbumSummaryTypeEnum),
    );
    yield r'locked';
    yield serializers.serialize(
      object.locked,
      specifiedType: const FullType(bool),
    );
    yield r'isShared';
    yield serializers.serialize(
      object.isShared,
      specifiedType: const FullType(bool),
    );
    yield r'coverAssetIds';
    yield serializers.serialize(
      object.coverAssetIds,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
    yield r'itemCount';
    yield serializers.serialize(
      object.itemCount,
      specifiedType: const FullType(int),
    );
    yield r'updatedAt';
    yield serializers.serialize(
      object.updatedAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'callerRole';
    yield serializers.serialize(
      object.callerRole,
      specifiedType: const FullType(AlbumSummaryCallerRoleEnum),
    );
    if (object.owner != null) {
      yield r'owner';
      yield serializers.serialize(
        object.owner,
        specifiedType: const FullType(AlbumOwnerRef),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumSummaryBuilder result,
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
            specifiedType: const FullType(AlbumSummaryTypeEnum),
          ) as AlbumSummaryTypeEnum;
          result.type = valueDes;
          break;
        case r'locked':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.locked = valueDes;
          break;
        case r'isShared':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isShared = valueDes;
          break;
        case r'coverAssetIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.coverAssetIds.replace(valueDes);
          break;
        case r'itemCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.itemCount = valueDes;
          break;
        case r'updatedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.updatedAt = valueDes;
          break;
        case r'callerRole':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AlbumSummaryCallerRoleEnum),
          ) as AlbumSummaryCallerRoleEnum;
          result.callerRole = valueDes;
          break;
        case r'owner':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(AlbumOwnerRef),
          ) as AlbumOwnerRef?;
          if (valueDes == null) continue;
          result.owner.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumSummaryBuilder();
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


/// The album's type. `virtual` albums are curated by their owner; `directory` albums mirror a filesystem path. Sync albums never reach this shape. 
class AlbumSummaryTypeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'virtual')
  static const AlbumSummaryTypeEnum virtual = _$albumSummaryTypeEnum_virtual;
  @BuiltValueEnumConst(wireName: r'directory')
  static const AlbumSummaryTypeEnum directory = _$albumSummaryTypeEnum_directory;

  static Serializer<AlbumSummaryTypeEnum> get serializer => _$albumSummaryTypeEnumSerializer;

  const AlbumSummaryTypeEnum._(String name): super(name);

  static BuiltSet<AlbumSummaryTypeEnum> get values => _$albumSummaryTypeEnumValues;
  static AlbumSummaryTypeEnum valueOf(String name) => _$albumSummaryTypeEnumValueOf(name);
}

/// The caller's relation to this album: `owner` items belong to the own group, `viewer` items to the shared group. The web client sections the wall off this discriminator. 
class AlbumSummaryCallerRoleEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'owner')
  static const AlbumSummaryCallerRoleEnum owner = _$albumSummaryCallerRoleEnum_owner;
  @BuiltValueEnumConst(wireName: r'viewer')
  static const AlbumSummaryCallerRoleEnum viewer = _$albumSummaryCallerRoleEnum_viewer;

  static Serializer<AlbumSummaryCallerRoleEnum> get serializer => _$albumSummaryCallerRoleEnumSerializer;

  const AlbumSummaryCallerRoleEnum._(String name): super(name);

  static BuiltSet<AlbumSummaryCallerRoleEnum> get values => _$albumSummaryCallerRoleEnumValues;
  static AlbumSummaryCallerRoleEnum valueOf(String name) => _$albumSummaryCallerRoleEnumValueOf(name);
}

