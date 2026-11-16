//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'job_run_state.g.dart';

/// The lifecycle of a job run: `queued` from creation until the driver picks it up, `running` while it works, then `succeeded`, `failed` -- or `cancelled` when the administrator's cancel ended it at a chunk boundary (the committed portion keeps its counts; a cancelled run is the admin's own deliberate ending, never a broken one). 
class JobRunState extends EnumClass {

  @BuiltValueEnumConst(wireName: r'queued')
  static const JobRunState queued = _$queued;
  @BuiltValueEnumConst(wireName: r'running')
  static const JobRunState running = _$running;
  @BuiltValueEnumConst(wireName: r'succeeded')
  static const JobRunState succeeded = _$succeeded;
  @BuiltValueEnumConst(wireName: r'failed')
  static const JobRunState failed = _$failed;
  @BuiltValueEnumConst(wireName: r'cancelled')
  static const JobRunState cancelled = _$cancelled;

  static Serializer<JobRunState> get serializer => _$jobRunStateSerializer;

  const JobRunState._(String name): super(name);

  static BuiltSet<JobRunState> get values => _$values;
  static JobRunState valueOf(String name) => _$valueOf(name);
}

/// Optionally, enum_class can generate a mixin to go with your enum for use
/// with Angular. It exposes your enum constants as getters. So, if you mix it
/// in to your Dart component class, the values become available to the
/// corresponding Angular template.
///
/// Trigger mixin generation by writing a line like this one next to your enum.
abstract class JobRunStateMixin = Object with _$JobRunStateMixin;

