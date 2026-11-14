import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for RunBudgetSettingsResponse
void main() {
  final instance = RunBudgetSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(RunBudgetSettingsResponse, () {
    // ConcurrencyPreset concurrencyPreset
    test('to test the property `concurrencyPreset`', () async {
      // TODO
    });

    // Whether the night window is on. Off, the heavy work runs whenever there is work. 
    // bool nightlyEnabled
    test('to test the property `nightlyEnabled`', () async {
      // TODO
    });

    // The window's opening time, `HH:MM` local to the server. Shipped `02:00`. 
    // String nightlyStart
    test('to test the property `nightlyStart`', () async {
      // TODO
    });

    // The window's closing time, `HH:MM` local to the server. Shipped `06:00`; earlier than the start is a window that spans midnight. 
    // String nightlyEnd
    test('to test the property `nightlyEnd`', () async {
      // TODO
    });

    // PausedJobsState pausedJobs
    test('to test the property `pausedJobs`', () async {
      // TODO
    });

  });
}
