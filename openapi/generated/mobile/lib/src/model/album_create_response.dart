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

part 'album_create_response.g.dart';

/// The creation answer: the fresh directory album's wall card (the summary exactly as `GET /albums` lists it) plus the honest audience report -- the requested ids that did not become audience, each with its reason -- and the creation probe's `sourceWritable` fact. The card's companions are the creation-time constants: `callerRole` `owner`, `isShared` true exactly when the audience landed a row, an empty `coverAssetIds`, `itemCount` 0. 
///
/// Properties:
/// * [id] - Identifier of the album.
/// * [title] - The album's display title.
/// * [type] - The album's type. `memory` albums are reference sets curated by their owner -- photos live in the albums they were ingested through, a memory holds references to them; `directory` albums mirror a filesystem path. The one `sync` card, when present, is the caller's own default album pinned first (never another account's): its `title` is the internal system label ('Sync') and clients render the localized display name instead (en \"My photos\", zh-Hans \"我的照片\"). 
/// * [isShared] - True when the album carries at least one `album_shares` row -- someone besides the owner can see it. Always true on shared-group items by construction. 
/// * [coverAssetIds] - The collage source of the wall card: the pinned cover asset as a single-element list when the owner set one, otherwise the album's most recent live member assets (up to four, `takenAt` descending, the asset id as the tiebreak). Empty for an album with no live members. 
/// * [itemCount] - The album's live member count: trashed, hidden and Live Photo component rows never count (a hidden asset is invisible to the whole family). 
/// * [updatedAt] - When the album row last changed (RFC 3339 UTC).
/// * [callerRole] - The caller's relation to this album: `owner` items belong to the own group, `viewer` items to the shared group. The web client sections the wall off this discriminator. 
/// * [owner] - The owner attribution of a shared item: the account that owns the album. Present exactly when `callerRole` is `viewer`; own items omit it (the owner is the caller). 
/// * [inTimeline] - The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly on the excludable cards -- the own `directory` items and every shared item; the own `memory` items omit it (they never feed the caller's own timeline, so there is nothing to exclude), and the pinned `sync` card omits it too (the core stream is never excludable). The value is the preference row's switch, true when no row exists (the default). 
/// * [sourceWritable] - The creation-time write probe's fact of the album this response just created: `true` when the probe file landed, read back and removed cleanly; `false` for a readable but read-only source. The creation SUCCEEDED either way -- a read-only directory is a legitimate album -- and `false` is the honest feedback the dialog renders (photos of this directory cannot be deleted through the application). 
/// * [skipped] - The requested audience ids that stayed ungranted: the album's own owner and every id that names no usable account. Empty when the whole audience landed. 
@BuiltValue()
abstract class AlbumCreateResponse implements AlbumSummary, Built<AlbumCreateResponse, AlbumCreateResponseBuilder> {
  /// The creation-time write probe's fact of the album this response just created: `true` when the probe file landed, read back and removed cleanly; `false` for a readable but read-only source. The creation SUCCEEDED either way -- a read-only directory is a legitimate album -- and `false` is the honest feedback the dialog renders (photos of this directory cannot be deleted through the application). 
  @BuiltValueField(wireName: r'sourceWritable')
  bool get sourceWritable;

  /// The requested audience ids that stayed ungranted: the album's own owner and every id that names no usable account. Empty when the whole audience landed. 
  @BuiltValueField(wireName: r'skipped')
  BuiltList<AlbumAudienceSkip> get skipped;

  AlbumCreateResponse._();

  factory AlbumCreateResponse([void updates(AlbumCreateResponseBuilder b)]) = _$AlbumCreateResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumCreateResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumCreateResponse> get serializer => _$AlbumCreateResponseSerializer();
}

class _$AlbumCreateResponseSerializer implements PrimitiveSerializer<AlbumCreateResponse> {
  @override
  final Iterable<Type> types = const [AlbumCreateResponse, _$AlbumCreateResponse];

  @override
  final String wireName = r'AlbumCreateResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumCreateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.owner != null) {
      yield r'owner';
      yield serializers.serialize(
        object.owner,
        specifiedType: const FullType(AlbumOwnerRef),
      );
    }
    yield r'sourceWritable';
    yield serializers.serialize(
      object.sourceWritable,
      specifiedType: const FullType(bool),
    );
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
    yield r'updatedAt';
    yield serializers.serialize(
      object.updatedAt,
      specifiedType: const FullType(DateTime),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumCreateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumCreateResponseBuilder result,
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
        case r'sourceWritable':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.sourceWritable = valueDes;
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
  AlbumCreateResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumCreateResponseBuilder();
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
class AlbumCreateResponseTypeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'memory')
  static const AlbumCreateResponseTypeEnum memory = _$albumCreateResponseTypeEnum_memory;
  @BuiltValueEnumConst(wireName: r'directory')
  static const AlbumCreateResponseTypeEnum directory = _$albumCreateResponseTypeEnum_directory;
  @BuiltValueEnumConst(wireName: r'sync')
  static const AlbumCreateResponseTypeEnum sync_ = _$albumCreateResponseTypeEnum_sync_;

  static Serializer<AlbumCreateResponseTypeEnum> get serializer => _$albumCreateResponseTypeEnumSerializer;

  const AlbumCreateResponseTypeEnum._(String name): super(name);

  static BuiltSet<AlbumCreateResponseTypeEnum> get values => _$albumCreateResponseTypeEnumValues;
  static AlbumCreateResponseTypeEnum valueOf(String name) => _$albumCreateResponseTypeEnumValueOf(name);
}

/// The caller's relation to this album: `owner` items belong to the own group, `viewer` items to the shared group. The web client sections the wall off this discriminator. 
class AlbumCreateResponseCallerRoleEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'owner')
  static const AlbumCreateResponseCallerRoleEnum owner = _$albumCreateResponseCallerRoleEnum_owner;
  @BuiltValueEnumConst(wireName: r'viewer')
  static const AlbumCreateResponseCallerRoleEnum viewer = _$albumCreateResponseCallerRoleEnum_viewer;

  static Serializer<AlbumCreateResponseCallerRoleEnum> get serializer => _$albumCreateResponseCallerRoleEnumSerializer;

  const AlbumCreateResponseCallerRoleEnum._(String name): super(name);

  static BuiltSet<AlbumCreateResponseCallerRoleEnum> get values => _$albumCreateResponseCallerRoleEnumValues;
  static AlbumCreateResponseCallerRoleEnum valueOf(String name) => _$albumCreateResponseCallerRoleEnumValueOf(name);
}

