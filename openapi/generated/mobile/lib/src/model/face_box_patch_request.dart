//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'face_box_patch_request.g.dart';

/// One adjusted face box, normalized to the image: x/y the top-left corner, width/height the extents, each a number in 0-1 with x + width <= 1 and y + height <= 1 and both extents at least 0.01. All four values are required. The server stores the box in the faces table's per-mille units. 
///
/// Properties:
/// * [x] - The box's left edge, as a fraction of the image's width.
/// * [y] - The box's top edge, as a fraction of the image's height.
/// * [width] - The box's width, as a fraction of the image's width (at least 0.01).
/// * [height] - The box's height, as a fraction of the image's height (at least 0.01).
@BuiltValue()
abstract class FaceBoxPatchRequest implements Built<FaceBoxPatchRequest, FaceBoxPatchRequestBuilder> {
  /// The box's left edge, as a fraction of the image's width.
  @BuiltValueField(wireName: r'x')
  double get x;

  /// The box's top edge, as a fraction of the image's height.
  @BuiltValueField(wireName: r'y')
  double get y;

  /// The box's width, as a fraction of the image's width (at least 0.01).
  @BuiltValueField(wireName: r'width')
  double get width;

  /// The box's height, as a fraction of the image's height (at least 0.01).
  @BuiltValueField(wireName: r'height')
  double get height;

  FaceBoxPatchRequest._();

  factory FaceBoxPatchRequest([void updates(FaceBoxPatchRequestBuilder b)]) = _$FaceBoxPatchRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FaceBoxPatchRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FaceBoxPatchRequest> get serializer => _$FaceBoxPatchRequestSerializer();
}

class _$FaceBoxPatchRequestSerializer implements PrimitiveSerializer<FaceBoxPatchRequest> {
  @override
  final Iterable<Type> types = const [FaceBoxPatchRequest, _$FaceBoxPatchRequest];

  @override
  final String wireName = r'FaceBoxPatchRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FaceBoxPatchRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'x';
    yield serializers.serialize(
      object.x,
      specifiedType: const FullType(double),
    );
    yield r'y';
    yield serializers.serialize(
      object.y,
      specifiedType: const FullType(double),
    );
    yield r'width';
    yield serializers.serialize(
      object.width,
      specifiedType: const FullType(double),
    );
    yield r'height';
    yield serializers.serialize(
      object.height,
      specifiedType: const FullType(double),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FaceBoxPatchRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FaceBoxPatchRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'x':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(double),
          ) as double;
          result.x = valueDes;
          break;
        case r'y':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(double),
          ) as double;
          result.y = valueDes;
          break;
        case r'width':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(double),
          ) as double;
          result.width = valueDes;
          break;
        case r'height':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(double),
          ) as double;
          result.height = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FaceBoxPatchRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FaceBoxPatchRequestBuilder();
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


