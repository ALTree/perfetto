package protoencoder

// Protobuf field numbers and enum constants for Perfetto trace format

// TrackEvent_Type enum values
const (
	TrackEventTypeUnspecified = 0
	TrackEventTypeSliceBegin  = 1
	TrackEventTypeSliceEnd    = 2
	TrackEventTypeInstant     = 3
	TrackEventTypeCounter     = 4
)

// BuiltinClock enum values
const (
	BuiltinClockBoottime = 6
)

// TracePacket_SequenceFlags enum values
const (
	SeqIncrementalStateCleared = 1
	SeqNeedsIncrementalState   = 2
)

// Field numbers for Trace message
const (
	TraceFieldPacket = 1
)

// Field numbers for TracePacket message
const (
	TracePacketFieldTimestamp                 = 8
	TracePacketFieldTimestampClockId          = 58
	TracePacketFieldTrustedPacketSequenceId   = 10
	TracePacketFieldSequenceFlags             = 13
	TracePacketFieldPreviousPacketDropped     = 42
	TracePacketFieldInternedData              = 12
	TracePacketFieldTracePacketDefaults       = 59
	TracePacketFieldTrackDescriptor           = 60
	TracePacketFieldTrackEvent                = 11
	TracePacketFieldClockSnapshot             = 6
)

// Field numbers for TrackDescriptor message
const (
	TrackDescriptorFieldUuid    = 1
	TrackDescriptorFieldName    = 2  // part of static_or_dynamic_name oneof
	TrackDescriptorFieldProcess = 3
	TrackDescriptorFieldThread  = 4
	TrackDescriptorFieldCounter = 8
)

// Field numbers for ProcessDescriptor message
const (
	ProcessDescriptorFieldPid         = 1
	ProcessDescriptorFieldProcessName = 6
)

// Field numbers for ThreadDescriptor message
const (
	ThreadDescriptorFieldPid        = 1
	ThreadDescriptorFieldTid        = 2
	ThreadDescriptorFieldThreadName = 5
)

// Field numbers for CounterDescriptor message
const (
	CounterDescriptorFieldUnitName = 6
)

// Field numbers for TrackEvent message
const (
	TrackEventFieldType              = 9
	TrackEventFieldTrackUuid         = 11
	TrackEventFieldName              = 23
	TrackEventFieldNameIid           = 10
	TrackEventFieldCounterValue      = 30
	TrackEventFieldFlowIds           = 47  // repeated fixed64
	TrackEventFieldDebugAnnotations  = 4
)

// Field numbers for ClockSnapshot message
const (
	ClockSnapshotFieldClocks = 1
)

// Field numbers for Clock message
const (
	ClockFieldClockId       = 1
	ClockFieldTimestamp     = 2
	ClockFieldIsIncremental = 4
)

// Field numbers for TracePacketDefaults message
const (
	TracePacketDefaultsFieldTimestampClockId   = 58
	TracePacketDefaultsFieldTrackEventDefaults = 11
)

// Field numbers for TrackEventDefaults message
const (
	TrackEventDefaultsFieldTrackUuid = 11
)

// Field numbers for InternedData message
const (
	InternedDataFieldEventNames                  = 2
	InternedDataFieldDebugAnnotationStringValues = 17
)

// Field numbers for EventName message
const (
	EventNameFieldIid  = 1
	EventNameFieldName = 2
)

// Field numbers for InternedString message
const (
	InternedStringFieldIid = 1
	InternedStringFieldStr = 2
)

// Field numbers for DebugAnnotation message
const (
	DebugAnnotationFieldName           = 10
	DebugAnnotationFieldNameIid        = 1
	DebugAnnotationFieldStringValue    = 6
	DebugAnnotationFieldStringValueIid = 17
)
