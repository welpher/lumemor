//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_album_ref.g.dart';

/// One entry of the asset detail's `albums` array: the album holding the asset as the calling user sees it -- identity, title, type and the viewer's own relation. The type rides the same convention as the wall cards: the sync entry's `title` is the internal system label ('Sync') and clients render the localized display name keyed by the type. 
///
/// Properties:
/// * [id] - Identifier of the album (the detail face's deep link).
/// * [allowAppWrites] - The album's write switch as its OWNER set it. True (the creation default) means the application may modify the album's source files -- delete them through the trash move and write capture-time edits into their metadata. Carried on the owner's rows; a viewer's row carries false (a viewer holds no write to read it for). The detail's delete affordance folds this with `sourceWritable`. 
/// * [sourceWritable] - Whether the album's source directory can take the application's writes. False only for a directory album whose creation-time write probe answered no -- every other type (sync, memory, a writable directory) is true. The photo-detail faces read this to gate the destructive affordances: a photo housed only in a read-only source is the scan's verdict alone and offers no delete. 
/// * [title] - The album's display title (the sync card's internal label).
/// * [type] - The album's type. A `memory` entry is one of the viewer's own or shared reference sets; `directory` entries are sources. The one `sync` entry, when present, is the viewer's own default album (never another account's). 
/// * [callerRole] - The caller's relation to this album: `owner` items are the caller's own (the sync stream's entry is always one), `viewer` items the albums shared to them. 
@BuiltValue()
abstract class AssetAlbumRef implements Built<AssetAlbumRef, AssetAlbumRefBuilder> {
  /// Identifier of the album (the detail face's deep link).
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The album's write switch as its OWNER set it. True (the creation default) means the application may modify the album's source files -- delete them through the trash move and write capture-time edits into their metadata. Carried on the owner's rows; a viewer's row carries false (a viewer holds no write to read it for). The detail's delete affordance folds this with `sourceWritable`. 
  @BuiltValueField(wireName: r'allowAppWrites')
  bool get allowAppWrites;

  /// Whether the album's source directory can take the application's writes. False only for a directory album whose creation-time write probe answered no -- every other type (sync, memory, a writable directory) is true. The photo-detail faces read this to gate the destructive affordances: a photo housed only in a read-only source is the scan's verdict alone and offers no delete. 
  @BuiltValueField(wireName: r'sourceWritable')
  bool get sourceWritable;

  /// The album's display title (the sync card's internal label).
  @BuiltValueField(wireName: r'title')
  String get title;

  /// The album's type. A `memory` entry is one of the viewer's own or shared reference sets; `directory` entries are sources. The one `sync` entry, when present, is the viewer's own default album (never another account's). 
  @BuiltValueField(wireName: r'type')
  AssetAlbumRefTypeEnum get type;
  // enum typeEnum {  memory,  directory,  sync,  };

  /// The caller's relation to this album: `owner` items are the caller's own (the sync stream's entry is always one), `viewer` items the albums shared to them. 
  @BuiltValueField(wireName: r'callerRole')
  AssetAlbumRefCallerRoleEnum get callerRole;
  // enum callerRoleEnum {  owner,  viewer,  };

  AssetAlbumRef._();

  factory AssetAlbumRef([void updates(AssetAlbumRefBuilder b)]) = _$AssetAlbumRef;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetAlbumRefBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetAlbumRef> get serializer => _$AssetAlbumRefSerializer();
}

class _$AssetAlbumRefSerializer implements PrimitiveSerializer<AssetAlbumRef> {
  @override
  final Iterable<Type> types = const [AssetAlbumRef, _$AssetAlbumRef];

  @override
  final String wireName = r'AssetAlbumRef';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetAlbumRef object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'allowAppWrites';
    yield serializers.serialize(
      object.allowAppWrites,
      specifiedType: const FullType(bool),
    );
    yield r'sourceWritable';
    yield serializers.serialize(
      object.sourceWritable,
      specifiedType: const FullType(bool),
    );
    yield r'title';
    yield serializers.serialize(
      object.title,
      specifiedType: const FullType(String),
    );
    yield r'type';
    yield serializers.serialize(
      object.type,
      specifiedType: const FullType(AssetAlbumRefTypeEnum),
    );
    yield r'callerRole';
    yield serializers.serialize(
      object.callerRole,
      specifiedType: const FullType(AssetAlbumRefCallerRoleEnum),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetAlbumRef object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetAlbumRefBuilder result,
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
        case r'allowAppWrites':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.allowAppWrites = valueDes;
          break;
        case r'sourceWritable':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.sourceWritable = valueDes;
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
            specifiedType: const FullType(AssetAlbumRefTypeEnum),
          ) as AssetAlbumRefTypeEnum;
          result.type = valueDes;
          break;
        case r'callerRole':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AssetAlbumRefCallerRoleEnum),
          ) as AssetAlbumRefCallerRoleEnum;
          result.callerRole = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetAlbumRef deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetAlbumRefBuilder();
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


/// The album's type. A `memory` entry is one of the viewer's own or shared reference sets; `directory` entries are sources. The one `sync` entry, when present, is the viewer's own default album (never another account's). 
class AssetAlbumRefTypeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'memory')
  static const AssetAlbumRefTypeEnum memory = _$assetAlbumRefTypeEnum_memory;
  @BuiltValueEnumConst(wireName: r'directory')
  static const AssetAlbumRefTypeEnum directory = _$assetAlbumRefTypeEnum_directory;
  @BuiltValueEnumConst(wireName: r'sync')
  static const AssetAlbumRefTypeEnum sync_ = _$assetAlbumRefTypeEnum_sync_;

  static Serializer<AssetAlbumRefTypeEnum> get serializer => _$assetAlbumRefTypeEnumSerializer;

  const AssetAlbumRefTypeEnum._(String name): super(name);

  static BuiltSet<AssetAlbumRefTypeEnum> get values => _$assetAlbumRefTypeEnumValues;
  static AssetAlbumRefTypeEnum valueOf(String name) => _$assetAlbumRefTypeEnumValueOf(name);
}

/// The caller's relation to this album: `owner` items are the caller's own (the sync stream's entry is always one), `viewer` items the albums shared to them. 
class AssetAlbumRefCallerRoleEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'owner')
  static const AssetAlbumRefCallerRoleEnum owner = _$assetAlbumRefCallerRoleEnum_owner;
  @BuiltValueEnumConst(wireName: r'viewer')
  static const AssetAlbumRefCallerRoleEnum viewer = _$assetAlbumRefCallerRoleEnum_viewer;

  static Serializer<AssetAlbumRefCallerRoleEnum> get serializer => _$assetAlbumRefCallerRoleEnumSerializer;

  const AssetAlbumRefCallerRoleEnum._(String name): super(name);

  static BuiltSet<AssetAlbumRefCallerRoleEnum> get values => _$assetAlbumRefCallerRoleEnumValues;
  static AssetAlbumRefCallerRoleEnum valueOf(String name) => _$assetAlbumRefCallerRoleEnumValueOf(name);
}

