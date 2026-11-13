//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'person_label_response.g.dart';

/// The calling user's label state of one face cluster after a label write: the caller's own row only (another account's label never shows here). 
///
/// Properties:
/// * [clusterId] - Identifier of the face cluster.
/// * [name] - The caller's (trimmed) name for the person.
@BuiltValue()
abstract class PersonLabelResponse implements Built<PersonLabelResponse, PersonLabelResponseBuilder> {
  /// Identifier of the face cluster.
  @BuiltValueField(wireName: r'clusterId')
  String get clusterId;

  /// The caller's (trimmed) name for the person.
  @BuiltValueField(wireName: r'name')
  String get name;

  PersonLabelResponse._();

  factory PersonLabelResponse([void updates(PersonLabelResponseBuilder b)]) = _$PersonLabelResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PersonLabelResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PersonLabelResponse> get serializer => _$PersonLabelResponseSerializer();
}

class _$PersonLabelResponseSerializer implements PrimitiveSerializer<PersonLabelResponse> {
  @override
  final Iterable<Type> types = const [PersonLabelResponse, _$PersonLabelResponse];

  @override
  final String wireName = r'PersonLabelResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PersonLabelResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'clusterId';
    yield serializers.serialize(
      object.clusterId,
      specifiedType: const FullType(String),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PersonLabelResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PersonLabelResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'clusterId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.clusterId = valueDes;
          break;
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PersonLabelResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PersonLabelResponseBuilder();
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


