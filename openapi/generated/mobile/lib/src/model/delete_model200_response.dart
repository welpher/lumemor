//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'delete_model200_response.g.dart';

/// The deletion's report.
///
/// Properties:
/// * [deleted] 
@BuiltValue()
abstract class DeleteModel200Response implements Built<DeleteModel200Response, DeleteModel200ResponseBuilder> {
  @BuiltValueField(wireName: r'deleted')
  bool get deleted;

  DeleteModel200Response._();

  factory DeleteModel200Response([void updates(DeleteModel200ResponseBuilder b)]) = _$DeleteModel200Response;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(DeleteModel200ResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<DeleteModel200Response> get serializer => _$DeleteModel200ResponseSerializer();
}

class _$DeleteModel200ResponseSerializer implements PrimitiveSerializer<DeleteModel200Response> {
  @override
  final Iterable<Type> types = const [DeleteModel200Response, _$DeleteModel200Response];

  @override
  final String wireName = r'DeleteModel200Response';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    DeleteModel200Response object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'deleted';
    yield serializers.serialize(
      object.deleted,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    DeleteModel200Response object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required DeleteModel200ResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'deleted':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.deleted = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  DeleteModel200Response deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = DeleteModel200ResponseBuilder();
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


