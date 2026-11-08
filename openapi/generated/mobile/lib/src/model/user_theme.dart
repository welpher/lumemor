//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'user_theme.g.dart';

/// The account's stored appearance preference for the client's color scheme. `auto` follows the device's system preference (the web app maps it to its `system` mode), `light` and `dark` are the manual choices. The value is declarative account state: a client that reads it applies it at sign-in, and the client performing the change applies it immediately. 
class UserTheme extends EnumClass {

  @BuiltValueEnumConst(wireName: r'auto')
  static const UserTheme auto = _$auto;
  @BuiltValueEnumConst(wireName: r'light')
  static const UserTheme light = _$light;
  @BuiltValueEnumConst(wireName: r'dark')
  static const UserTheme dark = _$dark;

  static Serializer<UserTheme> get serializer => _$userThemeSerializer;

  const UserTheme._(String name): super(name);

  static BuiltSet<UserTheme> get values => _$values;
  static UserTheme valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class UserThemeMixin = Object with _$UserThemeMixin;

