//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/album_owner_ref.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_summary.g.dart';

/// One album card of the wall: the metadata, the member count, the PHOTO count and the collage source, plus the `callerRole` discriminator the client keys each card's face by (a viewer card carries the read-only mark; an own card its own shared state). Shared items carry `owner`, the honest attribution of another account's album; own items omit it. 
///
/// Properties:
/// * [id] - Identifier of the album.
/// * [title] - The album's display title.
/// * [type] - The album's type. `memory` albums are reference sets curated by their owner -- photos live in the albums they were ingested through, a memory holds references to them; `directory` albums mirror a filesystem path. The one `sync` card, when present, is the caller's own default album pinned first (never another account's): its `title` is the internal system label ('Sync') and clients render the localized display name instead (en \"My photos\", zh-Hans \"我的照片\"). 
/// * [isShared] - True when the album carries at least one `album_shares` row -- someone besides the owner can see it. Always true on shared-group items by construction. 
/// * [coverAssetIds] - The collage source of the wall card, in render order: the frames of the album's cover. A cover holds four frames (the album card's 2x2 collage) or six (the memory band's 3-across 2-down strip), and the server answers the group the card draws -- the card is the first thing rendered and cannot see the collection, so the automatic group is computed here, never by the client.  The owner's pinned set wins: when the owner picked a full set, these are its frames in pick order, exactly as written (the server never re-sorts a pin). An album without pins renders the automatic group: a weekly rotation, re-sampled per ISO week from the whole live collection (never the newest frames and never a random order), frozen for the week so an upload mid-week cannot move a frame, and never rotating at all while the album holds fewer than twice the frame count -- such an album keeps one stable sample of its own. A pinned frame that is no longer visible to the caller (moved out, trashed, archived) is filled silently from the caller's visible pool; a fully dead pin falls back to the automatic group.  The order is the render order and the length is at most the card's frame count: a collection with fewer members than frames answers the frames it has (one fills, two split, three lay one large beside two small), and an album with no members answers an empty array (the card's empty cover, no placeholder photo). 
/// * [itemCount] - The album's member count: trashed and Live Photo component rows never count, while an archived member does -- an album is a reference container that keeps its archived members (a memory, a family-flow surface, drops them). The members of every media kind count here -- this is the collection's own size, the number the cover pool and the removal's arithmetic consult. 
/// * [photoCount] - The album's PHOTO count: the same member predicate narrowed to `kind: image` members. The interface's count idiom beside the image glyph reads this field (it says \"12 photos\", so it counts photos); a video is a member but never a photo, and the wall marks it with its own glyph instead. Equal to `itemCount` for an album holding no videos. 
/// * [updatedAt] - When the album row last changed (RFC 3339 UTC).
/// * [callerRole] - The caller's relation to this album: `owner` items belong to the caller's own group on the wire, `viewer` items to the shared group. The client renders ONE wall list and keys each card's face off this discriminator (the read-only mark is the member's, the shared state the owner's). 
/// * [owner] - The owner attribution of a shared item: the account that owns the album. Present exactly when `callerRole` is `viewer`; own items omit it (the owner is the caller). 
/// * [inTimeline] - The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly on the excludable cards -- the own `directory` items and every shared item; the own `memory` items omit it (they never feed the caller's own timeline, so there is nothing to exclude), and the pinned `sync` card omits it too (the core stream is never excludable). The value is the preference row's switch, true when no row exists (the default). 
@BuiltValue(instantiable: false)
abstract class AlbumSummary  {
  /// Identifier of the album.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The album's display title.
  @BuiltValueField(wireName: r'title')
  String get title;

  /// The album's type. `memory` albums are reference sets curated by their owner -- photos live in the albums they were ingested through, a memory holds references to them; `directory` albums mirror a filesystem path. The one `sync` card, when present, is the caller's own default album pinned first (never another account's): its `title` is the internal system label ('Sync') and clients render the localized display name instead (en \"My photos\", zh-Hans \"我的照片\"). 
  @BuiltValueField(wireName: r'type')
  AlbumSummaryTypeEnum get type;
  // enum typeEnum {  memory,  directory,  sync,  };

  /// True when the album carries at least one `album_shares` row -- someone besides the owner can see it. Always true on shared-group items by construction. 
  @BuiltValueField(wireName: r'isShared')
  bool get isShared;

  /// The collage source of the wall card, in render order: the frames of the album's cover. A cover holds four frames (the album card's 2x2 collage) or six (the memory band's 3-across 2-down strip), and the server answers the group the card draws -- the card is the first thing rendered and cannot see the collection, so the automatic group is computed here, never by the client.  The owner's pinned set wins: when the owner picked a full set, these are its frames in pick order, exactly as written (the server never re-sorts a pin). An album without pins renders the automatic group: a weekly rotation, re-sampled per ISO week from the whole live collection (never the newest frames and never a random order), frozen for the week so an upload mid-week cannot move a frame, and never rotating at all while the album holds fewer than twice the frame count -- such an album keeps one stable sample of its own. A pinned frame that is no longer visible to the caller (moved out, trashed, archived) is filled silently from the caller's visible pool; a fully dead pin falls back to the automatic group.  The order is the render order and the length is at most the card's frame count: a collection with fewer members than frames answers the frames it has (one fills, two split, three lay one large beside two small), and an album with no members answers an empty array (the card's empty cover, no placeholder photo). 
  @BuiltValueField(wireName: r'coverAssetIds')
  BuiltList<String> get coverAssetIds;

  /// The album's member count: trashed and Live Photo component rows never count, while an archived member does -- an album is a reference container that keeps its archived members (a memory, a family-flow surface, drops them). The members of every media kind count here -- this is the collection's own size, the number the cover pool and the removal's arithmetic consult. 
  @BuiltValueField(wireName: r'itemCount')
  int get itemCount;

  /// The album's PHOTO count: the same member predicate narrowed to `kind: image` members. The interface's count idiom beside the image glyph reads this field (it says \"12 photos\", so it counts photos); a video is a member but never a photo, and the wall marks it with its own glyph instead. Equal to `itemCount` for an album holding no videos. 
  @BuiltValueField(wireName: r'photoCount')
  int get photoCount;

  /// When the album row last changed (RFC 3339 UTC).
  @BuiltValueField(wireName: r'updatedAt')
  DateTime get updatedAt;

  /// The caller's relation to this album: `owner` items belong to the caller's own group on the wire, `viewer` items to the shared group. The client renders ONE wall list and keys each card's face off this discriminator (the read-only mark is the member's, the shared state the owner's). 
  @BuiltValueField(wireName: r'callerRole')
  AlbumSummaryCallerRoleEnum get callerRole;
  // enum callerRoleEnum {  owner,  viewer,  };

  /// The owner attribution of a shared item: the account that owns the album. Present exactly when `callerRole` is `viewer`; own items omit it (the owner is the caller). 
  @BuiltValueField(wireName: r'owner')
  AlbumOwnerRef? get owner;

  /// The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly on the excludable cards -- the own `directory` items and every shared item; the own `memory` items omit it (they never feed the caller's own timeline, so there is nothing to exclude), and the pinned `sync` card omits it too (the core stream is never excludable). The value is the preference row's switch, true when no row exists (the default). 
  @BuiltValueField(wireName: r'inTimeline')
  bool? get inTimeline;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumSummary> get serializer => _$AlbumSummarySerializer();
}

class _$AlbumSummarySerializer implements PrimitiveSerializer<AlbumSummary> {
  @override
  final Iterable<Type> types = const [AlbumSummary];

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
    yield r'type';
    yield serializers.serialize(
      object.type,
      specifiedType: const FullType(AlbumSummaryTypeEnum),
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
    yield r'photoCount';
    yield serializers.serialize(
      object.photoCount,
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
    AlbumSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  @override
  AlbumSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return serializers.deserialize(serialized, specifiedType: FullType($AlbumSummary)) as $AlbumSummary;
  }
}


/// a concrete implementation of [AlbumSummary], since [AlbumSummary] is not instantiable
@BuiltValue(instantiable: true)
abstract class $AlbumSummary implements AlbumSummary, Built<$AlbumSummary, $AlbumSummaryBuilder> {
  $AlbumSummary._();

  factory $AlbumSummary([void Function($AlbumSummaryBuilder)? updates]) = _$$AlbumSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults($AlbumSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<$AlbumSummary> get serializer => _$$AlbumSummarySerializer();
}

class _$$AlbumSummarySerializer implements PrimitiveSerializer<$AlbumSummary> {
  @override
  final Iterable<Type> types = const [$AlbumSummary, _$$AlbumSummary];

  @override
  final String wireName = r'$AlbumSummary';

  @override
  Object serialize(
    Serializers serializers,
    $AlbumSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return serializers.serialize(object, specifiedType: FullType(AlbumSummary))!;
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
        case r'type':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AlbumSummaryTypeEnum),
          ) as AlbumSummaryTypeEnum;
          result.type = valueDes;
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
        case r'photoCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.photoCount = valueDes;
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
  $AlbumSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = $AlbumSummaryBuilder();
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

/// The album's type. `memory` albums are reference sets curated by their owner -- photos live in the albums they were ingested through, a memory holds references to them; `directory` albums mirror a filesystem path. The one `sync` card, when present, is the caller's own default album pinned first (never another account's): its `title` is the internal system label ('Sync') and clients render the localized display name instead (en \"My photos\", zh-Hans \"我的照片\"). 
class AlbumSummaryTypeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'memory')
  static const AlbumSummaryTypeEnum memory = _$albumSummaryTypeEnum_memory;
  @BuiltValueEnumConst(wireName: r'directory')
  static const AlbumSummaryTypeEnum directory = _$albumSummaryTypeEnum_directory;
  @BuiltValueEnumConst(wireName: r'sync')
  static const AlbumSummaryTypeEnum sync_ = _$albumSummaryTypeEnum_sync_;

  static Serializer<AlbumSummaryTypeEnum> get serializer => _$albumSummaryTypeEnumSerializer;

  const AlbumSummaryTypeEnum._(String name): super(name);

  static BuiltSet<AlbumSummaryTypeEnum> get values => _$albumSummaryTypeEnumValues;
  static AlbumSummaryTypeEnum valueOf(String name) => _$albumSummaryTypeEnumValueOf(name);
}

/// The caller's relation to this album: `owner` items belong to the caller's own group on the wire, `viewer` items to the shared group. The client renders ONE wall list and keys each card's face off this discriminator (the read-only mark is the member's, the shared state the owner's). 
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

