package perfetto

import (
	"math/rand/v2"

	"google.golang.org/protobuf/proto"

	pp "github.com/ALTree/perfetto/internal/proto"
)

// -- { Process }

type Track interface {
	GetUuid() uint64
}

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

func (p Process) StartSlice(ts uint64, name string) Event {
	return NewEvent(p, pp.TrackEvent_TYPE_SLICE_BEGIN, ts, name)
}

func (p Process) EndSlice(ts uint64) Event {
	return NewEvent(p, pp.TrackEvent_TYPE_SLICE_END, ts, "")
}

// -- { Thread }

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

func (t Thread) StartSlice(ts uint64, name string) Event {
	return NewEvent(t, pp.TrackEvent_TYPE_SLICE_BEGIN, ts, name)
}

func (t Thread) EndSlice(ts uint64) Event {
	return NewEvent(t, pp.TrackEvent_TYPE_SLICE_END, ts, "")
}

// -- { Counter }

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

// NewValue creates a CounterValue event setting the value of the
// counter associated with the c CounterTrack.
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

// -- { Trace }

// Trace is a perfetto trace file
type Trace struct {
	Pt  pp.Trace
	TID uint32
}

// AddProcess adds a process with the given pid and name to the trace.
// It returns a Process handle that can be used to associate events to
// the process.
func (t *Trace) AddProcess(pid int32, name string) Process {
	pr := NewProcess(pid, name)
	t.Pt.Packet = append(t.Pt.Packet, &pp.TracePacket{Data: pr.Emit()})
	return pr
}

// AddThread adds a thread with the given tid and name to the trace,
// under the process with the given pid. It returns a Thread handle
// that can be used to associate events to the thread.
func (t *Trace) AddThread(pid, tid int32, name string) Thread {
	tr := NewThread(pid, tid, name)
	t.Pt.Packet = append(t.Pt.Packet, &pp.TracePacket{Data: tr.Emit()})
	return tr
}

// AddCounter adds a Counter track with the given name to the trace.
// It returns a Counter handle that can be used to associate events to
// the track.
func (t *Trace) AddCounter(name, unit string) Counter {
	ct := NewCounter(name, unit)
	t.Pt.Packet = append(t.Pt.Packet, &pp.TracePacket{Data: ct.Emit()})
	return ct
}

// AddEvent adds the given event to the trace.
func (t *Trace) AddEvent(e Event) {
	t.Pt.Packet = append(t.Pt.Packet,
		&pp.TracePacket{
			Timestamp:                       &e.Timestamp,
			Data:                            e.Emit(),
			OptionalTrustedPacketSequenceId: &pp.TracePacket_TrustedPacketSequenceId{t.TID},
		})
}

// Marshal calls proto.Marshal on the protobuf trace
func (t Trace) Marshal() ([]byte, error) {
	return proto.Marshal(&t.Pt)
}

// -- { Event }

// Event is a perfetto Event
type Event struct {
	Timestamp uint64
	Name      string
	Type      pp.TrackEvent_Type
	IsCounter bool // true iff TrackEvent_Counter
	Value     int64
	TrackUuid uint64
}

func NewEvent(track any, Type pp.TrackEvent_Type, ts uint64, name string) Event {
	var uuid uint64
	switch t := track.(type) {
	case Process:
	case Thread:
		uuid = t.Uuid
	}
	return Event{
		Timestamp: ts,
		Type:      Type,
		Name:      name,
		TrackUuid: uuid,
	}
}

func (e Event) Emit() *pp.TracePacket_TrackEvent {
	te := &pp.TracePacket_TrackEvent{
		&pp.TrackEvent{
			TrackUuid: &e.TrackUuid,
			Type:      &e.Type,
		},
	}

	if e.Name != "" {
		te.TrackEvent.NameField = &pp.TrackEvent_Name{e.Name}
	}
	if e.IsCounter {
		te.TrackEvent.CounterValueField = &pp.TrackEvent_CounterValue{e.Value}
	}

	return te
}
