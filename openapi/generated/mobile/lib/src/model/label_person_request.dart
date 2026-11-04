//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'label_person_request.g.dart';

/// The caller's name for one face cluster. The server trims the value and requires 1-64 characters; duplicate names are allowed (no NAME_TAKEN -- families have real duplicates). 
///
/// Properties:
/// * [name] - The caller's name for the person, trimmed.
@BuiltValue()
abstract class LabelPersonRequest implements Built<LabelPersonRequest, LabelPersonRequestBuilder> {
  /// The caller's name for the person, trimmed.
  @BuiltValueField(wireName: r'name')
  String get name;

  LabelPersonRequest._();

  factory LabelPersonRequest([void updates(LabelPersonRequestBuilder b)]) = _$LabelPersonRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(LabelPersonRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<LabelPersonRequest> get serializer => _$LabelPersonRequestSerializer();
}

class _$LabelPersonRequestSerializer implements PrimitiveSerializer<LabelPersonRequest> {
  @override
  final Iterable<Type> types = const [LabelPersonRequest, _$LabelPersonRequest];

  @override
  final String wireName = r'LabelPersonRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    LabelPersonRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    LabelPersonRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required LabelPersonRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
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
  LabelPersonRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = LabelPersonRequestBuilder();
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


