package perfetto

import (
	"math/rand/v2"

	"google.golang.org/protobuf/proto"

	pp "github.com/ALTree/perfetto/internal/proto"
)

type Track interface {
	GetUuid() uint64
}

// Trusted Packet Sequence ID
const TPSID = 99

// -- { Process } --------------------------------

// Process represents a perfetto track of kind 'process'
type Process struct {
	Pid  int32
	Name string
	Uuid uint64
}

func NewProcess(pid int32, name string) Process {
	return Process{
		Pid:  pid,
		Name: name,
		Uuid: rand.Uint64(),
	}
}

func (p Process) GetUuid() uint64 {
	return p.Uuid
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

func (p Process) InstantEvent(ts uint64, name string) Event {
	return NewEvent(p, pp.TrackEvent_TYPE_INSTANT, ts, name)
}

func (p Process) StartSlice(ts uint64, name string, ann ...Annotations) Event {
	return NewEvent(p, pp.TrackEvent_TYPE_SLICE_BEGIN, ts, name, ann...)
}

func (p Process) EndSlice(ts uint64) Event {
	return NewEvent(p, pp.TrackEvent_TYPE_SLICE_END, ts, "")
}

// -- { Thread } --------------------------------

// Thread represents a perfetto track of kind 'thread'
type Thread struct {
	Pid, Tid int32
	Name     string
	Uuid     uint64
}

func NewThread(pid, tid int32, name string) Thread {
	return Thread{
		Pid:  pid,
		Tid:  tid,
		Name: name,
		Uuid: rand.Uint64(),
	}
}

func (t Thread) GetUuid() uint64 {
	return t.Uuid
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

func (t Thread) InstantEvent(ts uint64, name string) Event {
	return NewEvent(t, pp.TrackEvent_TYPE_INSTANT, ts, name)
}

func (t Thread) StartSlice(ts uint64, name string, ann ...Annotations) Event {
	return NewEvent(t, pp.TrackEvent_TYPE_SLICE_BEGIN, ts, name, ann...)
}

func (t Thread) EndSlice(ts uint64) Event {
	return NewEvent(t, pp.TrackEvent_TYPE_SLICE_END, ts, "")
}

// -- { Counter } --------------------------------

// Counter represents a perfetto track of kind 'Counter'
type Counter struct {
	Uuid uint64
	Name string
	Unit string
}

func NewCounter(name, unit string) Counter {
	return Counter{
		Uuid: rand.Uint64(),
		Name: name,
		Unit: unit,
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

func (c Counter) NewValue(ts uint64, value int64) Event {
	return Event{
		Timestamp: ts,
		Type:      pp.TrackEvent_TYPE_COUNTER,
		Name:      c.Name,
		Value:     value,
		IsCounter: true,
		TrackUuid: c.Uuid,
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

func NewEvent(track any, Type pp.TrackEvent_Type, ts uint64, name string, ann ...Annotations) Event {
	var uuid uint64
	switch t := track.(type) {
	case Thread:
		uuid = t.Uuid
	}
	e := Event{
		Timestamp: ts,
		Type:      Type,
		Name:      name,
		TrackUuid: uuid,
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

func (t *Trace) Reset() {
	t.pt = pp.Trace{}
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

// AddProcess adds a process with the given pid and name to the trace.
// It returns a Process handle that can be used to associate events to
// the process.
func (t *Trace) AddProcess(pid int32, name string) Process {
	pr := NewProcess(pid, name)
	t.pt.Packet = append(t.pt.Packet, &pp.TracePacket{Data: pr.Emit()})
	return pr
}

// AddThread adds a thread with the given tid and name to the trace,
// under the process with the given pid. It returns a Thread handle
// that can be used to associate events to the thread.
func (t *Trace) AddThread(pid, tid int32, name string) Thread {
	tr := NewThread(pid, tid, name)
	t.pt.Packet = append(t.pt.Packet, &pp.TracePacket{Data: tr.Emit()})
	t.Threads[tid] = tr
	return tr
}

// AddCounter adds a Counter track with the given name to the trace.
// It returns a Counter handle that can be used to associate events to
// the track.
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
	if !ok && e.Name != "" {
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
