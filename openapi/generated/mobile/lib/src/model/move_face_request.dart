//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'move_face_request.g.dart';

/// One face's move onto an existing face cluster.
///
/// Properties:
/// * [targetClusterId] - The cluster the face joins. It must be visible to the caller; its representative recomputes when the move changes the strongest member. 
@BuiltValue()
abstract class MoveFaceRequest implements Built<MoveFaceRequest, MoveFaceRequestBuilder> {
  /// The cluster the face joins. It must be visible to the caller; its representative recomputes when the move changes the strongest member. 
  @BuiltValueField(wireName: r'targetClusterId')
  String get targetClusterId;

  MoveFaceRequest._();

  factory MoveFaceRequest([void updates(MoveFaceRequestBuilder b)]) = _$MoveFaceRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(MoveFaceRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<MoveFaceRequest> get serializer => _$MoveFaceRequestSerializer();
}

class _$MoveFaceRequestSerializer implements PrimitiveSerializer<MoveFaceRequest> {
  @override
  final Iterable<Type> types = const [MoveFaceRequest, _$MoveFaceRequest];

  @override
  final String wireName = r'MoveFaceRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    MoveFaceRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'targetClusterId';
    yield serializers.serialize(
      object.targetClusterId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    MoveFaceRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required MoveFaceRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'targetClusterId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.targetClusterId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  MoveFaceRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = MoveFaceRequestBuilder();
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


