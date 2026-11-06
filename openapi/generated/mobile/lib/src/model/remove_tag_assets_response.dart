//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'remove_tag_assets_response.g.dart';

/// The honest report of one assignment remove over the request's distinct ids: the rows that left and the ids that carried no assignment of the tag. 
///
/// Properties:
/// * [removed] - The assignment rows the request deleted. Zero for an all-skipped request, which is still a 200. 
/// * [skipped] - The distinct ids that carried no assignment of this tag -- never tagged, or already untagged. 
@BuiltValue()
abstract class RemoveTagAssetsResponse implements Built<RemoveTagAssetsResponse, RemoveTagAssetsResponseBuilder> {
  /// The assignment rows the request deleted. Zero for an all-skipped request, which is still a 200. 
  @BuiltValueField(wireName: r'removed')
  int get removed;

  /// The distinct ids that carried no assignment of this tag -- never tagged, or already untagged. 
  @BuiltValueField(wireName: r'skipped')
  int get skipped;

  RemoveTagAssetsResponse._();

  factory RemoveTagAssetsResponse([void updates(RemoveTagAssetsResponseBuilder b)]) = _$RemoveTagAssetsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(RemoveTagAssetsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<RemoveTagAssetsResponse> get serializer => _$RemoveTagAssetsResponseSerializer();
}

class _$RemoveTagAssetsResponseSerializer implements PrimitiveSerializer<RemoveTagAssetsResponse> {
  @override
  final Iterable<Type> types = const [RemoveTagAssetsResponse, _$RemoveTagAssetsResponse];

  @override
  final String wireName = r'RemoveTagAssetsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    RemoveTagAssetsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'removed';
    yield serializers.serialize(
      object.removed,
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
    RemoveTagAssetsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required RemoveTagAssetsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'removed':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.removed = valueDes;
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
  RemoveTagAssetsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = RemoveTagAssetsResponseBuilder();
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


