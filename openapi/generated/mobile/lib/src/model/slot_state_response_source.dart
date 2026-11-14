//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'slot_state_response_source.g.dart';

/// Where the slot's active model came from. `official` models were registered by the download service; `custom` models were imported by the administrator (a URL or a local file probe). `host` and `sha256` come from the model's on-disk manifest (the host is null when the manifest is unreadable or carries no source URL); `downloadedAt` is the registration time. Null for an unfilled slot. 
///
/// Properties:
/// * [kind] 
/// * [host] - The host the bytes were fetched from, when known.
/// * [sha256] - The primary file's pinned digest, or null when unknown.
/// * [downloadedAt] - When the model was registered.
@BuiltValue()
abstract class SlotStateResponseSource implements Built<SlotStateResponseSource, SlotStateResponseSourceBuilder> {
  @BuiltValueField(wireName: r'kind')
  SlotStateResponseSourceKindEnum get kind;
  // enum kindEnum {  official,  custom,  };

  /// The host the bytes were fetched from, when known.
  @BuiltValueField(wireName: r'host')
  String? get host;

  /// The primary file's pinned digest, or null when unknown.
  @BuiltValueField(wireName: r'sha256')
  String? get sha256;

  /// When the model was registered.
  @BuiltValueField(wireName: r'downloadedAt')
  DateTime get downloadedAt;

  SlotStateResponseSource._();

  factory SlotStateResponseSource([void updates(SlotStateResponseSourceBuilder b)]) = _$SlotStateResponseSource;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(SlotStateResponseSourceBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<SlotStateResponseSource> get serializer => _$SlotStateResponseSourceSerializer();
}

class _$SlotStateResponseSourceSerializer implements PrimitiveSerializer<SlotStateResponseSource> {
  @override
  final Iterable<Type> types = const [SlotStateResponseSource, _$SlotStateResponseSource];

  @override
  final String wireName = r'SlotStateResponseSource';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    SlotStateResponseSource object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'kind';
    yield serializers.serialize(
      object.kind,
      specifiedType: const FullType(SlotStateResponseSourceKindEnum),
    );
    yield r'host';
    yield object.host == null ? null : serializers.serialize(
      object.host,
      specifiedType: const FullType.nullable(String),
    );
    yield r'sha256';
    yield object.sha256 == null ? null : serializers.serialize(
      object.sha256,
      specifiedType: const FullType.nullable(String),
    );
    yield r'downloadedAt';
    yield serializers.serialize(
      object.downloadedAt,
      specifiedType: const FullType(DateTime),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    SlotStateResponseSource object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required SlotStateResponseSourceBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'kind':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(SlotStateResponseSourceKindEnum),
          ) as SlotStateResponseSourceKindEnum;
          result.kind = valueDes;
          break;
        case r'host':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.host = valueDes;
          break;
        case r'sha256':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.sha256 = valueDes;
          break;
        case r'downloadedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.downloadedAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  SlotStateResponseSource deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = SlotStateResponseSourceBuilder();
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


class SlotStateResponseSourceKindEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'official')
  static const SlotStateResponseSourceKindEnum official = _$slotStateResponseSourceKindEnum_official;
  @BuiltValueEnumConst(wireName: r'custom')
  static const SlotStateResponseSourceKindEnum custom = _$slotStateResponseSourceKindEnum_custom;

  static Serializer<SlotStateResponseSourceKindEnum> get serializer => _$slotStateResponseSourceKindEnumSerializer;

  const SlotStateResponseSourceKindEnum._(String name): super(name);

  static BuiltSet<SlotStateResponseSourceKindEnum> get values => _$slotStateResponseSourceKindEnumValues;
  static SlotStateResponseSourceKindEnum valueOf(String name) => _$slotStateResponseSourceKindEnumValueOf(name);
}

