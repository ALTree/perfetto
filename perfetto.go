package perfetto

import (
	"math/rand/v2"

	"github.com/ALTree/perfetto/internal/protoencoder"
)

// Common Trusted Packet Sequence ID
const TPSID = 1

// Clock ID for incremental timestamps
const CustomClockID uint32 = 64

// A Track is anything with a Name and a Uuid
type Track interface {
	GetName() string
	GetUuid() uint64
}

// -- { Track } --------------------------------

// BasicTrack represents a basic perfetto track. Process, Thread, and
// Counter all embed BasicTrack.
type BasicTrack struct {
	Name string
	Uuid uint64
}

func (t BasicTrack) GetName() string {
	return t.Name
}

func (t BasicTrack) GetUuid() uint64 {
	return t.Uuid
}

func NewTrack(name string) BasicTrack {
	return BasicTrack{
		Name: name,
		Uuid: rand.Uint64(),
	}
}

func (t BasicTrack) encodeTrackDescriptor(enc *protoencoder.Encoder) {
	enc.WriteUint64(protoencoder.TrackDescriptorFieldUuid, &t.Uuid)
	enc.WriteString(protoencoder.TrackDescriptorFieldName, &t.Name)
}

// The global track
func GlobalTrack() BasicTrack {
	return BasicTrack{Uuid: 0}
}

// -- { Process } --------------------------------

// Process represents a perfetto track of kind 'process'
type Process struct {
	BasicTrack
	Pid int32 // process id
}

func NewProcess(pid int32, name string) Process {
	return Process{
		BasicTrack: NewTrack(name),
		Pid:        pid,
	}
}

func (p Process) encodeTrackDescriptor(enc *protoencoder.Encoder) {
	enc.WriteUint64(protoencoder.TrackDescriptorFieldUuid, &p.Uuid)
	enc.WriteMessage(protoencoder.TrackDescriptorFieldProcess, func(e *protoencoder.Encoder) {
		e.WriteInt32(protoencoder.ProcessDescriptorFieldPid, &p.Pid)
		e.WriteString(protoencoder.ProcessDescriptorFieldProcessName, &p.Name)
	})
}

// -- { Thread } --------------------------------

// Thread represents a perfetto track of kind 'thread'
type Thread struct {
	BasicTrack
	Pid int32 // Parent process id
	Tid int32 // Thread id
}

func NewThread(pid, tid int32, name string) Thread {
	return Thread{
		BasicTrack: NewTrack(name),
		Pid:        pid,
		Tid:        tid,
	}
}

func (t Thread) encodeTrackDescriptor(enc *protoencoder.Encoder) {
	enc.WriteUint64(protoencoder.TrackDescriptorFieldUuid, &t.Uuid)
	enc.WriteMessage(protoencoder.TrackDescriptorFieldThread, func(e *protoencoder.Encoder) {
		e.WriteInt32(protoencoder.ThreadDescriptorFieldPid, &t.Pid)
		e.WriteInt32(protoencoder.ThreadDescriptorFieldTid, &t.Tid)
		e.WriteString(protoencoder.ThreadDescriptorFieldThreadName, &t.Name)
	})
}

// -- { Counter } --------------------------------

// Counter represents a perfetto track of kind 'Counter'
type Counter struct {
	BasicTrack
	Unit string
}

func NewCounter(name, unit string) Counter {
	return Counter{
		BasicTrack: NewTrack(name),
		Unit:       unit,
	}
}

func (c Counter) encodeTrackDescriptor(enc *protoencoder.Encoder) {
	enc.WriteUint64(protoencoder.TrackDescriptorFieldUuid, &c.Uuid)
	enc.WriteString(protoencoder.TrackDescriptorFieldName, &c.Name)
	enc.WriteMessage(protoencoder.TrackDescriptorFieldCounter, func(e *protoencoder.Encoder) {
		e.WriteString(protoencoder.CounterDescriptorFieldUnitName, &c.Unit)
	})
}

// -- { Event } --------------------------------

// Event is a perfetto Event
type Event struct {
	Timestamp uint64
	Name      string
	Type      int32       // TrackEvent_Type enum value
	IsCounter bool        // true iff Even is a TrackEvent_Counter
	Value     int64       // set for TrackEvent_Counters
	TrackUuid uint64      // Uuid of the track this event is part of
	Flows     []uint64    // optional flows IDs
	Ann       Annotations // optional Debug Annotations
}

func NewEvent(track Track, eventType int32, ts uint64, name string, flows []uint64, ann ...Annotations) Event {
	e := Event{
		Timestamp: ts,
		Type:      eventType,
		Name:      name,
		Flows:     flows,
		TrackUuid: track.GetUuid(),
	}
	if len(ann) > 0 {
		e.Ann = ann[0]
	}
	return e
}

func (e Event) encodeTrackEvent(enc *protoencoder.Encoder, tr *Trace) {
	enc.WriteEnum(protoencoder.TrackEventFieldType, &e.Type)
	enc.WriteUint64(protoencoder.TrackEventFieldTrackUuid, &e.TrackUuid)

	if tr.features.Interning {
		iid, _ := tr.interning.EventNames[e.Name]
		enc.WriteUint64(protoencoder.TrackEventFieldNameIid, &iid)
	} else {
		if e.Name != "" {
			enc.WriteString(protoencoder.TrackEventFieldName, &e.Name)
		}
	}

	if e.IsCounter {
		enc.WriteInt64(protoencoder.TrackEventFieldCounterValue, &e.Value)
	}

	// Write flow IDs (repeated fixed64)
	if len(e.Flows) > 0 {
		enc.WriteRepeatedFixed64(protoencoder.TrackEventFieldFlowIds, e.Flows)
	}

	// Write debug annotations
	e.Ann.encode(enc, tr)
}

// -- { Clock Snapshot  } --------------------------------

// emitClockSnapshot emits a global ClockSnapshot packet to enable incremental timestamps
func (t *Trace) emitClockSnapshot() {
	boottimeClockId := uint32(protoencoder.BuiltinClockBoottime)
	customClockId := uint32(CustomClockID)
	isIncremental := true
	timestamp := uint64(0)
	seqId := uint32(TPSID)

	t.buf.WriteMessage(protoencoder.TraceFieldPacket, func(packet *protoencoder.Encoder) {
		packet.WriteUint32(protoencoder.TracePacketFieldTrustedPacketSequenceId, &seqId)
		packet.WriteMessage(protoencoder.TracePacketFieldClockSnapshot, func(snapshot *protoencoder.Encoder) {
			// BOOTTIME clock
			snapshot.WriteMessage(protoencoder.ClockSnapshotFieldClocks, func(clock *protoencoder.Encoder) {
				clock.WriteUint32(protoencoder.ClockFieldClockId, &boottimeClockId)
				clock.WriteUint64(protoencoder.ClockFieldTimestamp, &timestamp)
			})
			// Custom incremental clock
			snapshot.WriteMessage(protoencoder.ClockSnapshotFieldClocks, func(clock *protoencoder.Encoder) {
				clock.WriteUint32(protoencoder.ClockFieldClockId, &customClockId)
				clock.WriteUint64(protoencoder.ClockFieldTimestamp, &timestamp)
				clock.WriteBool(protoencoder.ClockFieldIsIncremental, &isIncremental)
			})
		})
	})
}

// -- { Trace } --------------------------------

type Trace struct {
	Threads  map[int32]Thread   // Thread tracks added to the trace
	Counters map[string]Counter // Counter tracks added to the trace

	buf           *protoencoder.Encoder // Buffer for encoding the entire trace
	features      Features
	interning     Interning // interning maps (used if features.Interning)
	lastTimestamp uint64    // for incremental timestmaps (used if features.IncrementalTS)
}

type Features struct {
	Interning     bool // Use string interninng
	IncrementalTS bool // Emit incremental timestamp
}

var DefaultFeatures = Features{
	Interning:     true,
	IncrementalTS: true,
}

type Interning struct {
	EventNames map[string]uint64
	NextNameId uint64
	AnnValues  map[string]uint64
	NextAnnId  uint64
}

func NewTrace(features ...Features) Trace {
	tr := Trace{
		Threads:  make(map[int32]Thread),
		Counters: make(map[string]Counter),
		buf:      protoencoder.NewEncoder(),
		interning: Interning{
			EventNames: make(map[string]uint64),
			NextNameId: 1,
			AnnValues:  make(map[string]uint64),
			NextAnnId:  1,
		},
	}

	if len(features) > 0 {
		tr.features = features[0]
	} else {
		tr.features = DefaultFeatures
	}

	if tr.features.IncrementalTS {
		tr.emitClockSnapshot()
	}

	return tr
}

// AddTrack adds a BasicTrack with the given name to the trace. It
// returns a handle that can be used to associate events to the
// track.
func (t *Trace) AddTrack(name string) BasicTrack {
	tr := NewTrack(name)

	// Write TracePacket containing TrackDescriptor directly to trace buffer
	t.buf.WriteMessage(protoencoder.TraceFieldPacket, func(packet *protoencoder.Encoder) {
		packet.WriteMessage(protoencoder.TracePacketFieldTrackDescriptor, func(desc *protoencoder.Encoder) {
			tr.encodeTrackDescriptor(desc)
		})
	})
	return tr
}

// AddProcess adds a process with the given pid and name to the trace.
// It returns a handle that can be used to associate events to the
// process.
func (t *Trace) AddProcess(pid int32, name string) Process {
	pr := NewProcess(pid, name)

	// Write TracePacket containing TrackDescriptor directly to trace buffer
	t.buf.WriteMessage(protoencoder.TraceFieldPacket, func(packet *protoencoder.Encoder) {
		packet.WriteMessage(protoencoder.TracePacketFieldTrackDescriptor, func(desc *protoencoder.Encoder) {
			pr.encodeTrackDescriptor(desc)
		})
	})
	return pr
}

// AddThread adds a thread with the given tid and name to the trace,
// under the process with the given pid. It returns a handle that can
// be used to associate events to the thread.
func (t *Trace) AddThread(pid, tid int32, name string) Thread {
	tr := NewThread(pid, tid, name)

	// Write TracePacket containing TrackDescriptor directly to trace buffer
	t.buf.WriteMessage(protoencoder.TraceFieldPacket, func(packet *protoencoder.Encoder) {
		packet.WriteMessage(protoencoder.TracePacketFieldTrackDescriptor, func(desc *protoencoder.Encoder) {
			tr.encodeTrackDescriptor(desc)
		})
	})
	t.Threads[tid] = tr
	return tr
}

// AddCounter adds a Counter track with the given name to the trace.
// It returns a handle that can be used to associate events to the
// track.
func (t *Trace) AddCounter(name, unit string) Counter {
	ct := NewCounter(name, unit)

	// Write TracePacket containing TrackDescriptor directly to trace buffer
	t.buf.WriteMessage(protoencoder.TraceFieldPacket, func(packet *protoencoder.Encoder) {
		packet.WriteMessage(protoencoder.TracePacketFieldTrackDescriptor, func(desc *protoencoder.Encoder) {
			ct.encodeTrackDescriptor(desc)
		})
	})
	t.Counters[name] = ct
	return ct
}

// AddEvent adds the given event to the trace.
func (t *Trace) AddEvent(e Event) {

	var hasEventNameInterning bool
	var annStringValues []struct{ iid uint64; str []byte }

	if t.features.Interning {
		// Event Names Interning
		if _, ok := t.interning.EventNames[e.Name]; !ok && e.Name != "" {
			iid := t.interning.NextNameId
			t.interning.EventNames[e.Name] = iid
			t.interning.NextNameId++
			hasEventNameInterning = true
		}

		// Debug Annotations Values Interning
		for _, ann := range e.Ann {
			_, ok := t.interning.AnnValues[ann.V]
			if !ok && ann.V != "" {
				iid := t.interning.NextAnnId
				annStringValues = append(annStringValues, struct{ iid uint64; str []byte }{iid, []byte(ann.V)})
				t.interning.AnnValues[ann.V] = iid
				t.interning.NextAnnId++
			}
		}
	}

	seqId := uint32(TPSID)

	// Write TracePacket directly to trace buffer
	t.buf.WriteMessage(protoencoder.TraceFieldPacket, func(packet *protoencoder.Encoder) {
		// We emit an incremental timestamp if 1) the feature is enabled
		// and 2) the delta since the last timestamp is positive. If (2)
		// is not true, emit the event on the default, non-incremental
		// clock to avoid a wraparound on the uint64 delta.
		var timestamp uint64
		if t.features.IncrementalTS && e.Timestamp >= t.lastTimestamp {
			delta := e.Timestamp - t.lastTimestamp
			t.lastTimestamp = e.Timestamp
			timestamp = delta
			customClockId := uint32(CustomClockID)
			packet.WriteUint64(protoencoder.TracePacketFieldTimestamp, &timestamp)
			packet.WriteUint32(protoencoder.TracePacketFieldTimestampClockId, &customClockId)
		} else {
			timestamp = e.Timestamp
			packet.WriteUint64(protoencoder.TracePacketFieldTimestamp, &timestamp)
		}

		packet.WriteUint32(protoencoder.TracePacketFieldTrustedPacketSequenceId, &seqId)

		// Encode TrackEvent
		packet.WriteMessage(protoencoder.TracePacketFieldTrackEvent, func(trackEvent *protoencoder.Encoder) {
			e.encodeTrackEvent(trackEvent, t)
		})

		// Encode interning data if needed
		if hasEventNameInterning || len(annStringValues) > 0 {
			packet.WriteMessage(protoencoder.TracePacketFieldInternedData, func(internedData *protoencoder.Encoder) {
				if hasEventNameInterning {
					internedData.WriteMessage(protoencoder.InternedDataFieldEventNames, func(eventName *protoencoder.Encoder) {
						iid := t.interning.EventNames[e.Name]
						eventName.WriteUint64(protoencoder.EventNameFieldIid, &iid)
						eventName.WriteString(protoencoder.EventNameFieldName, &e.Name)
					})
				}

				for _, ann := range annStringValues {
					internedData.WriteMessage(protoencoder.InternedDataFieldDebugAnnotationStringValues, func(internedStr *protoencoder.Encoder) {
						internedStr.WriteUint64(protoencoder.InternedStringFieldIid, &ann.iid)
						internedStr.WriteBytes(protoencoder.InternedStringFieldStr, ann.str)
					})
				}
			})

			if len(t.interning.EventNames) == 1 {
				// First packet with interning data needs to set these
				prevDropped := true
				seqFlags := uint32(protoencoder.SeqIncrementalStateCleared | protoencoder.SeqNeedsIncrementalState)
				packet.WriteBool(protoencoder.TracePacketFieldPreviousPacketDropped, &prevDropped)
				packet.WriteUint32(protoencoder.TracePacketFieldSequenceFlags, &seqFlags)
			}
		} else {
			// Later packets using interned data need to set this
			if t.features.Interning {
				seqFlags := uint32(protoencoder.SeqNeedsIncrementalState)
				packet.WriteUint32(protoencoder.TracePacketFieldSequenceFlags, &seqFlags)
			}
		}
	})
}

func (t *Trace) InstantEvent(track Track, ts uint64, name string) {
	t.AddEvent(NewEvent(track, protoencoder.TrackEventTypeInstant, ts, name, nil))
}

func (t *Trace) StartSlice(track Track, ts uint64, name string, ann ...Annotations) {
	t.AddEvent(NewEvent(track, protoencoder.TrackEventTypeSliceBegin, ts, name, nil, ann...))
}

func (t *Trace) StartSliceWithFlow(track Track, ts uint64, name string, flows []uint64, ann ...Annotations) {
	t.AddEvent(NewEvent(track, protoencoder.TrackEventTypeSliceBegin, ts, name, flows, ann...))
}

func (t *Trace) EndSlice(track Track, ts uint64) {
	t.AddEvent(NewEvent(track, protoencoder.TrackEventTypeSliceEnd, ts, "", nil))
}

func (t *Trace) EndSliceWithFlow(track Track, ts uint64, flows []uint64) {
	t.AddEvent(NewEvent(track, protoencoder.TrackEventTypeSliceEnd, ts, "", flows))
}

func (t *Trace) NewValue(track Counter, ts uint64, val int64) {
	t.AddEvent(Event{
		Timestamp: ts,
		Type:      protoencoder.TrackEventTypeCounter,
		Name:      track.Name,
		Value:     val,
		IsCounter: true,
		TrackUuid: track.Uuid,
	})
}

func (t *Trace) Reset() {
	t.buf.Reset()
	t.lastTimestamp = 0
}

// Marshal returns the encoded trace as protobuf binary format
func (t Trace) Marshal() ([]byte, error) {
	// The buffer already contains the complete Trace message
	return t.buf.Bytes(), nil
}

// -- { Misc } ----------------------------------------------------------------

// KV is a (key, value) tuple representing a Debug Annotation
type KV struct {
	K, V string
}

type Annotations []KV

func (a Annotations) encode(enc *protoencoder.Encoder, tr *Trace) {
	for i := range a {
		key := a[i].K
		value := a[i].V
		enc.WriteMessage(protoencoder.TrackEventFieldDebugAnnotations, func(ann *protoencoder.Encoder) {
			ann.WriteString(protoencoder.DebugAnnotationFieldName, &key)
			if tr.features.Interning {
				iid, _ := tr.interning.AnnValues[value]
				ann.WriteUint64(protoencoder.DebugAnnotationFieldStringValueIid, &iid)
			} else {
				ann.WriteString(protoencoder.DebugAnnotationFieldStringValue, &value)
			}
		})
	}
}
