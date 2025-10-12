package perfetto

import (
	"math/rand/v2"

	"google.golang.org/protobuf/proto"

	pp "github.com/ALTree/perfetto/internal/proto"
)

// Common Trusted Packet Sequence ID
const TPSID = 99

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

func (t BasicTrack) Emit() *pp.TracePacket_TrackDescriptor {
	return &pp.TracePacket_TrackDescriptor{
		&pp.TrackDescriptor{
			Uuid:                &t.Uuid,
			StaticOrDynamicName: &pp.TrackDescriptor_Name{Name: t.Name},
		},
	}
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

func (p Process) Emit() *pp.TracePacket_TrackDescriptor {
	return &pp.TracePacket_TrackDescriptor{
		&pp.TrackDescriptor{
			Uuid: &p.Uuid,
			Process: &pp.ProcessDescriptor{
				Pid:         &p.Pid,
				ProcessName: &p.Name,
			}},
	}
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

func (t Thread) Emit() *pp.TracePacket_TrackDescriptor {
	return &pp.TracePacket_TrackDescriptor{
		&pp.TrackDescriptor{
			Uuid: &t.Uuid,
			Thread: &pp.ThreadDescriptor{
				Pid:        &t.Pid,
				Tid:        &t.Tid,
				ThreadName: &t.Name,
			}},
	}
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

func (c Counter) Emit() *pp.TracePacket_TrackDescriptor {
	return &pp.TracePacket_TrackDescriptor{
		&pp.TrackDescriptor{
			Uuid:                &c.Uuid,
			StaticOrDynamicName: &pp.TrackDescriptor_Name{c.Name},
			Counter: &pp.CounterDescriptor{
				UnitName: proto.String(c.Unit),
			},
		},
	}
}

// -- { Event } --------------------------------

// Event is a perfetto Event
type Event struct {
	Timestamp uint64
	Name      string
	Type      pp.TrackEvent_Type
	IsCounter bool        // true iff Even is a TrackEvent_Counter
	Value     int64       // set for TrackEvent_Counters
	TrackUuid uint64      // Uuid of the track this event is part of
	Ann       Annotations // optional Debug Annotations
}

func NewEvent(track Track, Type pp.TrackEvent_Type, ts uint64, name string, ann ...Annotations) Event {
	e := Event{
		Timestamp: ts,
		Type:      Type,
		Name:      name,
		TrackUuid: track.GetUuid(),
	}
	if len(ann) > 0 {
		e.Ann = ann[0]
	}
	return e
}

func (e Event) Emit(iid uint64) *pp.TracePacket_TrackEvent {
	te := &pp.TracePacket_TrackEvent{
		&pp.TrackEvent{
			TrackUuid:        &e.TrackUuid,
			Type:             &e.Type,
			DebugAnnotations: e.Ann.Emit(),
		},
	}

	if debug.DisableInterning {
		te.TrackEvent.NameField = &pp.TrackEvent_Name{e.Name}
	} else {
		te.TrackEvent.NameField = &pp.TrackEvent_NameIid{iid}
	}

	if e.IsCounter {
		te.TrackEvent.CounterValueField = &pp.TrackEvent_CounterValue{e.Value}
	}

	return te
}

// -- { Trace } --------------------------------

type Trace struct {
	Threads  map[int32]Thread   // Thread tracks added to the trace
	Counters map[string]Counter // Counter tracks added to the trace

	pt        pp.Trace
	interning Interning
}

type Interning struct {
	EventNames map[string]uint64
	NextId     uint64
}

func NewTrace() Trace {
	return Trace{
		Threads:  make(map[int32]Thread),
		Counters: make(map[string]Counter),
		interning: Interning{
			EventNames: make(map[string]uint64),
			NextId:     1,
		},
	}
}

// AddTrack adds a BasicTrack with the give name to the trace. It
// returns a handle that can be used to associate events to the
// track.
func (t *Trace) AddTrack(name string) BasicTrack {
	tr := NewTrack(name)
	t.pt.Packet = append(t.pt.Packet, &pp.TracePacket{Data: tr.Emit()})
	return tr
}

// AddProcess adds a process with the given pid and name to the trace.
// It returns a handle that can be used to associate events to the
// process.
func (t *Trace) AddProcess(pid int32, name string) Process {
	pr := NewProcess(pid, name)
	t.pt.Packet = append(t.pt.Packet, &pp.TracePacket{Data: pr.Emit()})
	return pr
}

// AddThread adds a thread with the given tid and name to the trace,
// under the process with the given pid. It returns a handle that can
// be used to associate events to the thread.
func (t *Trace) AddThread(pid, tid int32, name string) Thread {
	tr := NewThread(pid, tid, name)
	t.pt.Packet = append(t.pt.Packet, &pp.TracePacket{Data: tr.Emit()})
	t.Threads[tid] = tr
	return tr
}

// AddCounter adds a Counter track with the given name to the trace.
// It returns a handle that can be used to associate events to the
// track.
func (t *Trace) AddCounter(name, unit string) Counter {
	ct := NewCounter(name, unit)
	t.pt.Packet = append(t.pt.Packet, &pp.TracePacket{Data: ct.Emit()})
	t.Counters[name] = ct
	return ct
}

// AddEvent adds the given event to the trace.
func (t *Trace) AddEvent(e Event) {

	// If the Event name string is not already interned, do so
	var internedData *pp.InternedData
	iid, ok := t.interning.EventNames[e.Name]
	if !debug.DisableInterning && !ok && e.Name != "" {
		iid = t.interning.NextId
		internedData = &pp.InternedData{
			EventNames: []*pp.EventName{
				&pp.EventName{Iid: &iid, Name: &e.Name},
			},
		}
	}

	tp := &pp.TracePacket{
		Timestamp:                       &e.Timestamp,
		Data:                            e.Emit(iid),
		OptionalTrustedPacketSequenceId: &pp.TracePacket_TrustedPacketSequenceId{TPSID},
	}

	// In addition to this Event's data, emit the interning data for
	// the new name
	if internedData != nil {
		tp.InternedData = internedData
		if t.interning.NextId == 1 {
			// First packet with interning data needs to set this, apparently
			tp.PreviousPacketDropped = proto.Bool(true)
			tp.SequenceFlags = proto.Uint32(uint32(
				pp.TracePacket_SEQ_INCREMENTAL_STATE_CLEARED |
					pp.TracePacket_SEQ_NEEDS_INCREMENTAL_STATE))
		}
		t.interning.EventNames[e.Name] = iid
		t.interning.NextId++
	} else {
		// Packets using interned data need to set this
		tp.SequenceFlags = proto.Uint32(uint32(
			pp.TracePacket_SEQ_NEEDS_INCREMENTAL_STATE))
	}

	t.pt.Packet = append(t.pt.Packet, tp)
}

func (t *Trace) InstantEvent(track Track, ts uint64, name string) {
	t.AddEvent(NewEvent(track, pp.TrackEvent_TYPE_INSTANT, ts, name))
}

func (t *Trace) StartSlice(track Track, ts uint64, name string, ann ...Annotations) {
	t.AddEvent(NewEvent(track, pp.TrackEvent_TYPE_SLICE_BEGIN, ts, name, ann...))
}

func (t *Trace) EndSlice(track Track, ts uint64) {
	t.AddEvent(NewEvent(track, pp.TrackEvent_TYPE_SLICE_END, ts, ""))
}

func (t *Trace) NewValue(track Counter, ts uint64, val int64) {
	t.AddEvent(Event{
		Timestamp: ts,
		Type:      pp.TrackEvent_TYPE_COUNTER,
		Name:      track.Name,
		Value:     val,
		IsCounter: true,
		TrackUuid: track.Uuid,
	})
}

func (t *Trace) Reset() {
	t.pt = pp.Trace{}
}

// Marshal calls proto.Marshal on the protobuf trace
func (t Trace) Marshal() ([]byte, error) {
	return proto.Marshal(&t.pt)
}

// -- { Misc } ----------------------------------------------------------------

// KV is a (key, value) tuple representing a Debug Annotation
type KV struct {
	K, V string
}

type Annotations []KV

func (a Annotations) Emit() []*pp.DebugAnnotation {
	var res []*pp.DebugAnnotation
	for i := range a {
		name := &pp.DebugAnnotation_Name{Name: a[i].K}
		value := &pp.DebugAnnotation_StringValue{StringValue: a[i].V}
		res = append(res, &pp.DebugAnnotation{NameField: name, Value: value})
	}
	return res
}

var debug = struct {
	DisableInterning bool
}{
	DisableInterning: false,
}
