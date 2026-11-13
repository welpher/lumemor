//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/date.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'person_birthdate_response.g.dart';

/// One person's birth date after the write (the entity's own attribute, shared by every account). 
///
/// Properties:
/// * [clusterId] - Identifier of the face cluster.
/// * [birthDate] - The stored birth date, or null when the person carries none. 
@BuiltValue()
abstract class PersonBirthdateResponse implements Built<PersonBirthdateResponse, PersonBirthdateResponseBuilder> {
  /// Identifier of the face cluster.
  @BuiltValueField(wireName: r'clusterId')
  String get clusterId;

  /// The stored birth date, or null when the person carries none. 
  @BuiltValueField(wireName: r'birthDate')
  Date? get birthDate;

  PersonBirthdateResponse._();

  factory PersonBirthdateResponse([void updates(PersonBirthdateResponseBuilder b)]) = _$PersonBirthdateResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PersonBirthdateResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PersonBirthdateResponse> get serializer => _$PersonBirthdateResponseSerializer();
}

class _$PersonBirthdateResponseSerializer implements PrimitiveSerializer<PersonBirthdateResponse> {
  @override
  final Iterable<Type> types = const [PersonBirthdateResponse, _$PersonBirthdateResponse];

  @override
  final String wireName = r'PersonBirthdateResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PersonBirthdateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'clusterId';
    yield serializers.serialize(
      object.clusterId,
      specifiedType: const FullType(String),
    );
    yield r'birthDate';
    yield object.birthDate == null ? null : serializers.serialize(
      object.birthDate,
      specifiedType: const FullType.nullable(Date),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PersonBirthdateResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PersonBirthdateResponseBuilder result,
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
        case r'birthDate':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(Date),
          ) as Date?;
          if (valueDes == null) continue;
          result.birthDate = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PersonBirthdateResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PersonBirthdateResponseBuilder();
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


