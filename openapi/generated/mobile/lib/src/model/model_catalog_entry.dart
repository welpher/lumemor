//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/model_slot.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_catalog_entry.g.dart';

/// One official-list entry: a manifest model with the facts the switch dialog's dropdown renders (name, license, size, dimension) and its download state. 
///
/// Properties:
/// * [id] - The manifest model's stable id.
/// * [slot] 
/// * [name] - The display name the manifest's notice carries, derived from the model id.
/// * [license] - The upstream license identifier, empty when undeclared.
/// * [dim] - The embedding dimension; absent for models that do not embed.
/// * [bytes] - The pinned byte count of the entry's files.
/// * [downloaded] - Whether the entry's files are present on disk (a presence probe, not a digest check).
@BuiltValue()
abstract class ModelCatalogEntry implements Built<ModelCatalogEntry, ModelCatalogEntryBuilder> {
  /// The manifest model's stable id.
  @BuiltValueField(wireName: r'id')
  String get id;

  @BuiltValueField(wireName: r'slot')
  ModelSlot get slot;
  // enum slotEnum {  face_detect,  face_recognize,  clip_embed,  clip_text,  ocr,  };

  /// The display name the manifest's notice carries, derived from the model id.
  @BuiltValueField(wireName: r'name')
  String get name;

  /// The upstream license identifier, empty when undeclared.
  @BuiltValueField(wireName: r'license')
  String get license;

  /// The embedding dimension; absent for models that do not embed.
  @BuiltValueField(wireName: r'dim')
  int? get dim;

  /// The pinned byte count of the entry's files.
  @BuiltValueField(wireName: r'bytes')
  int get bytes;

  /// Whether the entry's files are present on disk (a presence probe, not a digest check).
  @BuiltValueField(wireName: r'downloaded')
  bool get downloaded;

  ModelCatalogEntry._();

  factory ModelCatalogEntry([void updates(ModelCatalogEntryBuilder b)]) = _$ModelCatalogEntry;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelCatalogEntryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelCatalogEntry> get serializer => _$ModelCatalogEntrySerializer();
}

class _$ModelCatalogEntrySerializer implements PrimitiveSerializer<ModelCatalogEntry> {
  @override
  final Iterable<Type> types = const [ModelCatalogEntry, _$ModelCatalogEntry];

  @override
  final String wireName = r'ModelCatalogEntry';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelCatalogEntry object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'slot';
    yield serializers.serialize(
      object.slot,
      specifiedType: const FullType(ModelSlot),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
    yield r'license';
    yield serializers.serialize(
      object.license,
      specifiedType: const FullType(String),
    );
    if (object.dim != null) {
      yield r'dim';
      yield serializers.serialize(
        object.dim,
        specifiedType: const FullType(int),
      );
    }
    yield r'bytes';
    yield serializers.serialize(
      object.bytes,
      specifiedType: const FullType(int),
    );
    yield r'downloaded';
    yield serializers.serialize(
      object.downloaded,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelCatalogEntry object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelCatalogEntryBuilder result,
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
        case r'slot':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ModelSlot),
          ) as ModelSlot;
          result.slot = valueDes;
          break;
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        case r'license':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.license = valueDes;
          break;
        case r'dim':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.dim = valueDes;
          break;
        case r'bytes':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.bytes = valueDes;
          break;
        case r'downloaded':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.downloaded = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelCatalogEntry deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelCatalogEntryBuilder();
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


