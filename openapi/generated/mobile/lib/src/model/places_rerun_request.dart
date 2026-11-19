//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'places_rerun_request.g.dart';

/// The mode of one place re-resolution run: the honest pair the map & places card's dialog chooses between. 
///
/// Properties:
/// * [mode] - `missing` re-resolves only the GPS-bearing rows whose place is absent (nothing that already resolved is rewritten); `all` re-resolves every GPS-bearing row. Absent means `all`. 
@BuiltValue()
abstract class PlacesRerunRequest implements Built<PlacesRerunRequest, PlacesRerunRequestBuilder> {
  /// `missing` re-resolves only the GPS-bearing rows whose place is absent (nothing that already resolved is rewritten); `all` re-resolves every GPS-bearing row. Absent means `all`. 
  @BuiltValueField(wireName: r'mode')
  PlacesRerunRequestModeEnum? get mode;
  // enum modeEnum {  missing,  all,  };

  PlacesRerunRequest._();

  factory PlacesRerunRequest([void updates(PlacesRerunRequestBuilder b)]) = _$PlacesRerunRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PlacesRerunRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PlacesRerunRequest> get serializer => _$PlacesRerunRequestSerializer();
}

class _$PlacesRerunRequestSerializer implements PrimitiveSerializer<PlacesRerunRequest> {
  @override
  final Iterable<Type> types = const [PlacesRerunRequest, _$PlacesRerunRequest];

  @override
  final String wireName = r'PlacesRerunRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PlacesRerunRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.mode != null) {
      yield r'mode';
      yield serializers.serialize(
        object.mode,
        specifiedType: const FullType(PlacesRerunRequestModeEnum),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    PlacesRerunRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PlacesRerunRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'mode':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(PlacesRerunRequestModeEnum),
          ) as PlacesRerunRequestModeEnum?;
          if (valueDes == null) continue;
          result.mode = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PlacesRerunRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PlacesRerunRequestBuilder();
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


/// `missing` re-resolves only the GPS-bearing rows whose place is absent (nothing that already resolved is rewritten); `all` re-resolves every GPS-bearing row. Absent means `all`. 
class PlacesRerunRequestModeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'missing')
  static const PlacesRerunRequestModeEnum missing = _$placesRerunRequestModeEnum_missing;
  @BuiltValueEnumConst(wireName: r'all')
  static const PlacesRerunRequestModeEnum all = _$placesRerunRequestModeEnum_all;

  static Serializer<PlacesRerunRequestModeEnum> get serializer => _$placesRerunRequestModeEnumSerializer;

  const PlacesRerunRequestModeEnum._(String name): super(name);

  static BuiltSet<PlacesRerunRequestModeEnum> get values => _$placesRerunRequestModeEnumValues;
  static PlacesRerunRequestModeEnum valueOf(String name) => _$placesRerunRequestModeEnumValueOf(name);
}

