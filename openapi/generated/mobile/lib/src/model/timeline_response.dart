//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/timeline_item.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'timeline_response.g.dart';

/// One page of a timeline-shaped view: the aggregated timeline, the favorites view or the archived view all answer with this page shape (the same ordering and the same cursor semantics). `nextCursor` is present exactly when another page follows the `items` of this one: the client passes it back as the `cursor` query parameter to continue, and a response without it ends the pagination. The last page is the one whose follow-up page would be empty, so no run ever observes an empty page mid-pagination. 
///
/// Properties:
/// * [items] - The timeline items of this page, ordered by `takenAt` descending with the id as the tiebreak. 
/// * [nextCursor] - The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page. 
@BuiltValue()
abstract class TimelineResponse implements Built<TimelineResponse, TimelineResponseBuilder> {
  /// The timeline items of this page, ordered by `takenAt` descending with the id as the tiebreak. 
  @BuiltValueField(wireName: r'items')
  BuiltList<TimelineItem> get items;

  /// The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page. 
  @BuiltValueField(wireName: r'nextCursor')
  String? get nextCursor;

  TimelineResponse._();

  factory TimelineResponse([void updates(TimelineResponseBuilder b)]) = _$TimelineResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TimelineResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TimelineResponse> get serializer => _$TimelineResponseSerializer();
}

class _$TimelineResponseSerializer implements PrimitiveSerializer<TimelineResponse> {
  @override
  final Iterable<Type> types = const [TimelineResponse, _$TimelineResponse];

  @override
  final String wireName = r'TimelineResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TimelineResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'items';
    yield serializers.serialize(
      object.items,
      specifiedType: const FullType(BuiltList, [FullType(TimelineItem)]),
    );
    if (object.nextCursor != null) {
      yield r'nextCursor';
      yield serializers.serialize(
        object.nextCursor,
        specifiedType: const FullType(String),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    TimelineResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TimelineResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'items':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(TimelineItem)]),
          ) as BuiltList<TimelineItem>;
          result.items.replace(valueDes);
          break;
        case r'nextCursor':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.nextCursor = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TimelineResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TimelineResponseBuilder();
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


