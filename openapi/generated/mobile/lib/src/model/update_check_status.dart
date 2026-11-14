//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'update_check_status.g.dart';

/// The verdict of the release check. `up_to_date` and `update_available` are the two claims a successful tag comparison makes; `unknown` is the honest silence of a check that could not conclude (a failed probe, a feed that named nothing readable, a running version that is not a release tag); `disabled` means no check runs. 
class UpdateCheckStatus extends EnumClass {

  @BuiltValueEnumConst(wireName: r'disabled')
  static const UpdateCheckStatus disabled = _$disabled;
  @BuiltValueEnumConst(wireName: r'unknown')
  static const UpdateCheckStatus unknown = _$unknown;
  @BuiltValueEnumConst(wireName: r'up_to_date')
  static const UpdateCheckStatus upToDate = _$upToDate;
  @BuiltValueEnumConst(wireName: r'update_available')
  static const UpdateCheckStatus updateAvailable = _$updateAvailable;

  static Serializer<UpdateCheckStatus> get serializer => _$updateCheckStatusSerializer;

  const UpdateCheckStatus._(String name): super(name);

  static BuiltSet<UpdateCheckStatus> get values => _$values;
  static UpdateCheckStatus valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class UpdateCheckStatusMixin = Object with _$UpdateCheckStatusMixin;

