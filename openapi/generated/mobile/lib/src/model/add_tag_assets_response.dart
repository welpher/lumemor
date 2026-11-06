//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'add_tag_assets_response.g.dart';

/// The honest report of one assignment add over the request's distinct ids: what landed and what was skipped, invisible and already-tagged ids alike. 
///
/// Properties:
/// * [added] - The assignment rows the request inserted. Zero for an all-skipped request, which is still a 200 -- batch semantics, never an error. 
/// * [skipped] - The distinct ids that did not land: unknown ids, another account's unshared assets, any id the caller cannot see, and the ids already carrying the tag (the idempotent add). 
@BuiltValue()
abstract class AddTagAssetsResponse implements Built<AddTagAssetsResponse, AddTagAssetsResponseBuilder> {
  /// The assignment rows the request inserted. Zero for an all-skipped request, which is still a 200 -- batch semantics, never an error. 
  @BuiltValueField(wireName: r'added')
  int get added;

  /// The distinct ids that did not land: unknown ids, another account's unshared assets, any id the caller cannot see, and the ids already carrying the tag (the idempotent add). 
  @BuiltValueField(wireName: r'skipped')
  int get skipped;

  AddTagAssetsResponse._();

  factory AddTagAssetsResponse([void updates(AddTagAssetsResponseBuilder b)]) = _$AddTagAssetsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AddTagAssetsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AddTagAssetsResponse> get serializer => _$AddTagAssetsResponseSerializer();
}

class _$AddTagAssetsResponseSerializer implements PrimitiveSerializer<AddTagAssetsResponse> {
  @override
  final Iterable<Type> types = const [AddTagAssetsResponse, _$AddTagAssetsResponse];

  @override
  final String wireName = r'AddTagAssetsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AddTagAssetsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'added';
    yield serializers.serialize(
      object.added,
      specifiedType: const FullType(int),
    );
    yield r'skipped';
    yield serializers.serialize(
      object.skipped,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AddTagAssetsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AddTagAssetsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'added':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.added = valueDes;
          break;
        case r'skipped':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.skipped = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AddTagAssetsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AddTagAssetsResponseBuilder();
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


