//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_patch.g.dart';

/// The partial edit of one asset: every field is optional, at least one must be present (a body that carries neither -- null values count as absent -- is a client bug and answers 400), and only the fields the request carries update; the rest keep their values. Both fields are owner-level asset state, so a patch that actually changes one of them publishes a single `asset.updated` event; a patch that repeats the stored values is the idempotent no-op and publishes nothing. 
///
/// Properties:
/// * [description] - The owner's free-text description of the asset, the text the detail panel renders. The value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters -- the count is over runes, so a text of multi-byte characters stays within the bound at the same length. 
/// * [isHidden] - The owner-level visibility switch: `true` keeps the asset out of every live view (the timeline, the favorites) until it is unhidden; `false` returns it. The same switch the batch `hide` / `unhide` actions flip. 
@BuiltValue()
abstract class AssetPatch implements Built<AssetPatch, AssetPatchBuilder> {
  /// The owner's free-text description of the asset, the text the detail panel renders. The value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters -- the count is over runes, so a text of multi-byte characters stays within the bound at the same length. 
  @BuiltValueField(wireName: r'description')
  String? get description;

  /// The owner-level visibility switch: `true` keeps the asset out of every live view (the timeline, the favorites) until it is unhidden; `false` returns it. The same switch the batch `hide` / `unhide` actions flip. 
  @BuiltValueField(wireName: r'isHidden')
  bool? get isHidden;

  AssetPatch._();

  factory AssetPatch([void updates(AssetPatchBuilder b)]) = _$AssetPatch;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetPatchBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetPatch> get serializer => _$AssetPatchSerializer();
}

class _$AssetPatchSerializer implements PrimitiveSerializer<AssetPatch> {
  @override
  final Iterable<Type> types = const [AssetPatch, _$AssetPatch];

  @override
  final String wireName = r'AssetPatch';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.description != null) {
      yield r'description';
      yield serializers.serialize(
        object.description,
        specifiedType: const FullType(String),
      );
    }
    if (object.isHidden != null) {
      yield r'isHidden';
      yield serializers.serialize(
        object.isHidden,
        specifiedType: const FullType(bool),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetPatchBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'description':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.description = valueDes;
          break;
        case r'isHidden':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.isHidden = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetPatch deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetPatchBuilder();
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


