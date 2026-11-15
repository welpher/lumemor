//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'face_bbox.g.dart';

/// A face's bounding box in per-mille units of the asset's view derivative (the pixels the crop endpoint addresses and the on-photo face-box layer renders): `x`/`y` the top-left corner, `w`/`h` the box's extents, each an integer 0..1000. A detected face's box is the detector's own; a hand-drawn face (POST /faces) carries the drawn box as-is. 
///
/// Properties:
/// * [x] - Per-mille of the derivative's width where the box starts.
/// * [y] - Per-mille of the derivative's height where the box starts.
/// * [w] - The box's width, in per-mille of the derivative's width.
/// * [h] - The box's height, in per-mille of the derivative's height.
@BuiltValue()
abstract class FaceBbox implements Built<FaceBbox, FaceBboxBuilder> {
  /// Per-mille of the derivative's width where the box starts.
  @BuiltValueField(wireName: r'x')
  int get x;

  /// Per-mille of the derivative's height where the box starts.
  @BuiltValueField(wireName: r'y')
  int get y;

  /// The box's width, in per-mille of the derivative's width.
  @BuiltValueField(wireName: r'w')
  int get w;

  /// The box's height, in per-mille of the derivative's height.
  @BuiltValueField(wireName: r'h')
  int get h;

  FaceBbox._();

  factory FaceBbox([void updates(FaceBboxBuilder b)]) = _$FaceBbox;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FaceBboxBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FaceBbox> get serializer => _$FaceBboxSerializer();
}

class _$FaceBboxSerializer implements PrimitiveSerializer<FaceBbox> {
  @override
  final Iterable<Type> types = const [FaceBbox, _$FaceBbox];

  @override
  final String wireName = r'FaceBbox';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FaceBbox object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'x';
    yield serializers.serialize(
      object.x,
      specifiedType: const FullType(int),
    );
    yield r'y';
    yield serializers.serialize(
      object.y,
      specifiedType: const FullType(int),
    );
    yield r'w';
    yield serializers.serialize(
      object.w,
      specifiedType: const FullType(int),
    );
    yield r'h';
    yield serializers.serialize(
      object.h,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FaceBbox object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FaceBboxBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'x':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.x = valueDes;
          break;
        case r'y':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.y = valueDes;
          break;
        case r'w':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.w = valueDes;
          break;
        case r'h':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.h = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FaceBbox deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FaceBboxBuilder();
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


