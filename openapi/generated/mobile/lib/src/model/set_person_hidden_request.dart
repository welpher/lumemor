//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'set_person_hidden_request.g.dart';

/// The caller's hidden switch for one face cluster. The switch rides on the caller's label row, so hiding requires the caller to have named the cluster first (`person_hide_requires_label` otherwise). 
///
/// Properties:
/// * [isHidden] - true hides the person for the caller (a badge on their people list, not a filter); false unhides. 
@BuiltValue()
abstract class SetPersonHiddenRequest implements Built<SetPersonHiddenRequest, SetPersonHiddenRequestBuilder> {
  /// true hides the person for the caller (a badge on their people list, not a filter); false unhides. 
  @BuiltValueField(wireName: r'isHidden')
  bool get isHidden;

  SetPersonHiddenRequest._();

  factory SetPersonHiddenRequest([void updates(SetPersonHiddenRequestBuilder b)]) = _$SetPersonHiddenRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(SetPersonHiddenRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<SetPersonHiddenRequest> get serializer => _$SetPersonHiddenRequestSerializer();
}

class _$SetPersonHiddenRequestSerializer implements PrimitiveSerializer<SetPersonHiddenRequest> {
  @override
  final Iterable<Type> types = const [SetPersonHiddenRequest, _$SetPersonHiddenRequest];

  @override
  final String wireName = r'SetPersonHiddenRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    SetPersonHiddenRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'isHidden';
    yield serializers.serialize(
      object.isHidden,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    SetPersonHiddenRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required SetPersonHiddenRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
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
  SetPersonHiddenRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = SetPersonHiddenRequestBuilder();
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


