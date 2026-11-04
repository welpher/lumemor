//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'person_label.g.dart';

/// The calling user's private label of one face cluster.
///
/// Properties:
/// * [name] - The caller's name for the person.
/// * [isHidden] - The caller's hidden switch. Hidden people stay listed in `GET /people` carrying this flag (badge semantics). 
@BuiltValue()
abstract class PersonLabel implements Built<PersonLabel, PersonLabelBuilder> {
  /// The caller's name for the person.
  @BuiltValueField(wireName: r'name')
  String get name;

  /// The caller's hidden switch. Hidden people stay listed in `GET /people` carrying this flag (badge semantics). 
  @BuiltValueField(wireName: r'isHidden')
  bool get isHidden;

  PersonLabel._();

  factory PersonLabel([void updates(PersonLabelBuilder b)]) = _$PersonLabel;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PersonLabelBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PersonLabel> get serializer => _$PersonLabelSerializer();
}

class _$PersonLabelSerializer implements PrimitiveSerializer<PersonLabel> {
  @override
  final Iterable<Type> types = const [PersonLabel, _$PersonLabel];

  @override
  final String wireName = r'PersonLabel';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PersonLabel object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
    yield r'isHidden';
    yield serializers.serialize(
      object.isHidden,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PersonLabel object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PersonLabelBuilder result,
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
        case r'isHidden':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isHidden = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PersonLabel deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PersonLabelBuilder();
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


