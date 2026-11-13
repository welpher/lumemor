//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/date.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'set_person_birthdate_request.g.dart';

/// One person's birth date: a `YYYY-MM-DD` date, or null to clear the stored value. The field is always present (a body without it answers 400) so a clear is an explicit act. 
///
/// Properties:
/// * [birthDate] - The person's birth date, or null to clear it. 
@BuiltValue()
abstract class SetPersonBirthdateRequest implements Built<SetPersonBirthdateRequest, SetPersonBirthdateRequestBuilder> {
  /// The person's birth date, or null to clear it. 
  @BuiltValueField(wireName: r'birthDate')
  Date? get birthDate;

  SetPersonBirthdateRequest._();

  factory SetPersonBirthdateRequest([void updates(SetPersonBirthdateRequestBuilder b)]) = _$SetPersonBirthdateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(SetPersonBirthdateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<SetPersonBirthdateRequest> get serializer => _$SetPersonBirthdateRequestSerializer();
}

class _$SetPersonBirthdateRequestSerializer implements PrimitiveSerializer<SetPersonBirthdateRequest> {
  @override
  final Iterable<Type> types = const [SetPersonBirthdateRequest, _$SetPersonBirthdateRequest];

  @override
  final String wireName = r'SetPersonBirthdateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    SetPersonBirthdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'birthDate';
    yield object.birthDate == null ? null : serializers.serialize(
      object.birthDate,
      specifiedType: const FullType.nullable(Date),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    SetPersonBirthdateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required SetPersonBirthdateRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
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
  SetPersonBirthdateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = SetPersonBirthdateRequestBuilder();
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


