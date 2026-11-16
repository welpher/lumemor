//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/album_owner_ref.dart';
import 'package:lumemor_api/src/model/album_summary.dart';
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/album_audience_skip.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'memory_create_response.g.dart';

/// The creation answer: the fresh memory's wall card (the summary exactly as `GET /albums` lists it) plus the honest audience report. The card's companions are the creation-time constants: `callerRole` `owner`, `type` `memory`, `isShared` true exactly when the audience landed a row, an empty `coverAssetIds`, `itemCount` 0, `photoCount` 0, no `inTimeline` field (an own memory card omits it -- the memory never feeds its creator's timeline). 
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
/// * [skipped] - The requested audience ids that stayed ungranted: the memory's own owner and every id that names no usable account. Empty when the whole audience landed. 
@BuiltValue()
abstract class MemoryCreateResponse implements AlbumSummary, Built<MemoryCreateResponse, MemoryCreateResponseBuilder> {
  /// The requested audience ids that stayed ungranted: the memory's own owner and every id that names no usable account. Empty when the whole audience landed. 
  @BuiltValueField(wireName: r'skipped')
  BuiltList<AlbumAudienceSkip> get skipped;

  MemoryCreateResponse._();

  factory MemoryCreateResponse([void updates(MemoryCreateResponseBuilder b)]) = _$MemoryCreateResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(MemoryCreateResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<MemoryCreateResponse> get serializer => _$MemoryCreateResponseSerializer();
}

class _$MemoryCreateResponseSerializer implements PrimitiveSerializer<MemoryCreateResponse> {
  @override
  final Iterable<Type> types = const [MemoryCreateResponse, _$MemoryCreateResponse];

  @override
  final String wireName = r'MemoryCreateResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    MemoryCreateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.owner != null) {
      yield r'owner';
      yield serializers.serialize(
        object.owner,
        specifiedType: const FullType(AlbumOwnerRef),
      );
    }
    yield r'coverAssetIds';
    yield serializers.serialize(
      object.coverAssetIds,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
    yield r'callerRole';
    yield serializers.serialize(
      object.callerRole,
      specifiedType: const FullType(AlbumSummaryCallerRoleEnum),
    );
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
    if (object.inTimeline != null) {
      yield r'inTimeline';
      yield serializers.serialize(
        object.inTimeline,
        specifiedType: const FullType(bool),
      );
    }
    yield r'isShared';
    yield serializers.serialize(
      object.isShared,
      specifiedType: const FullType(bool),
    );
    yield r'skipped';
    yield serializers.serialize(
      object.skipped,
      specifiedType: const FullType(BuiltList, [FullType(AlbumAudienceSkip)]),
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
  }

  @override
  Object serialize(
    Serializers serializers,
    MemoryCreateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required MemoryCreateResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'owner':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(AlbumOwnerRef),
          ) as AlbumOwnerRef?;
          if (valueDes == null) continue;
          result.owner.replace(valueDes);
          break;
        case r'coverAssetIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.coverAssetIds.replace(valueDes);
          break;
        case r'callerRole':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AlbumSummaryCallerRoleEnum),
          ) as AlbumSummaryCallerRoleEnum;
          result.callerRole = valueDes;
          break;
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
        case r'inTimeline':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.inTimeline = valueDes;
          break;
        case r'isShared':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isShared = valueDes;
          break;
        case r'skipped':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(AlbumAudienceSkip)]),
          ) as BuiltList<AlbumAudienceSkip>;
          result.skipped.replace(valueDes);
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
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  MemoryCreateResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = MemoryCreateResponseBuilder();
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
class MemoryCreateResponseTypeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'memory')
  static const MemoryCreateResponseTypeEnum memory = _$memoryCreateResponseTypeEnum_memory;
  @BuiltValueEnumConst(wireName: r'directory')
  static const MemoryCreateResponseTypeEnum directory = _$memoryCreateResponseTypeEnum_directory;
  @BuiltValueEnumConst(wireName: r'sync')
  static const MemoryCreateResponseTypeEnum sync_ = _$memoryCreateResponseTypeEnum_sync_;

  static Serializer<MemoryCreateResponseTypeEnum> get serializer => _$memoryCreateResponseTypeEnumSerializer;

  const MemoryCreateResponseTypeEnum._(String name): super(name);

  static BuiltSet<MemoryCreateResponseTypeEnum> get values => _$memoryCreateResponseTypeEnumValues;
  static MemoryCreateResponseTypeEnum valueOf(String name) => _$memoryCreateResponseTypeEnumValueOf(name);
}

/// The caller's relation to this album: `owner` items belong to the caller's own group on the wire, `viewer` items to the shared group. The client renders ONE wall list and keys each card's face off this discriminator (the read-only mark is the member's, the shared state the owner's). 
class MemoryCreateResponseCallerRoleEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'owner')
  static const MemoryCreateResponseCallerRoleEnum owner = _$memoryCreateResponseCallerRoleEnum_owner;
  @BuiltValueEnumConst(wireName: r'viewer')
  static const MemoryCreateResponseCallerRoleEnum viewer = _$memoryCreateResponseCallerRoleEnum_viewer;

  static Serializer<MemoryCreateResponseCallerRoleEnum> get serializer => _$memoryCreateResponseCallerRoleEnumSerializer;

  const MemoryCreateResponseCallerRoleEnum._(String name): super(name);

  static BuiltSet<MemoryCreateResponseCallerRoleEnum> get values => _$memoryCreateResponseCallerRoleEnumValues;
  static MemoryCreateResponseCallerRoleEnum valueOf(String name) => _$memoryCreateResponseCallerRoleEnumValueOf(name);
}

