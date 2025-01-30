package perfetto

import (
	"fmt"
	"math/rand/v2"
	"os"

	"google.golang.org/protobuf/proto"

	pp "github.com/ALTree/perfetto/proto"
)

type Trace struct {
	Pt  pp.Trace
	TID uint32
}

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

func (t *Trace) AddProcess(pid int32, name string) Process {
	pr := NewProcess(pid, name)
	t.Pt.Packet = append(t.Pt.Packet, &pp.TracePacket{Data: pr.Emit()})
	return pr
}

func (t *Trace) AddThread(pid, tid int32, name string) Thread {
	tr := NewThread(pid, tid, name)
	t.Pt.Packet = append(t.Pt.Packet, &pp.TracePacket{Data: tr.Emit()})
	return tr
}

type Event struct {
	Timestamp uint64
	Type      pp.TrackEvent_Type
	Name      string
	TrackUuid uint64
}

var EventSliceBegin pp.TrackEvent_Type = pp.TrackEvent_TYPE_SLICE_BEGIN
var EventSliceEnd pp.TrackEvent_Type = pp.TrackEvent_TYPE_SLICE_END

func (p Process) NewEvent(Type pp.TrackEvent_Type, ts uint64, name string) Event {
	return Event{
		Timestamp: ts,
		Type:      Type,
		Name:      name,
		TrackUuid: p.Uuid,
	}
}

func (p Process) InstantEvent(ts uint64, name string) Event {
	return p.NewEvent(pp.TrackEvent_TYPE_INSTANT, ts, name)
}

func (p Process) StartSlice(ts uint64, name string) Event {
	return p.NewEvent(pp.TrackEvent_TYPE_SLICE_BEGIN, ts, name)
}

func (p Process) EndSlice(ts uint64) Event {
	return p.NewEvent(pp.TrackEvent_TYPE_SLICE_END, ts, "")
}

func (t Thread) NewEvent(Type pp.TrackEvent_Type, ts uint64, name string) Event {
	return Event{
		Timestamp: ts,
		Type:      Type,
		Name:      name,
		TrackUuid: t.Uuid,
	}
}

func (t Thread) InstantEvent(ts uint64, name string) Event {
	return t.NewEvent(pp.TrackEvent_TYPE_INSTANT, ts, name)
}

func (t Thread) StartSlice(ts uint64, name string) Event {
	return t.NewEvent(pp.TrackEvent_TYPE_SLICE_BEGIN, ts, name)
}

func (t Thread) EndSlice(ts uint64) Event {
	return t.NewEvent(pp.TrackEvent_TYPE_SLICE_END, ts, "")
}

func (e Event) Emit() *pp.TracePacket_TrackEvent {
	te := &pp.TracePacket_TrackEvent{
		&pp.TrackEvent{
			TrackUuid: &e.TrackUuid,
			Type:      &e.Type,
		},
	}

	if name := e.Name; name != "" {
		te.TrackEvent.NameField = &pp.TrackEvent_Name{name}
	}

	return te
}

func (t *Trace) AddEvent(e Event) {
	t.Pt.Packet = append(t.Pt.Packet,
		&pp.TracePacket{
			Timestamp:                       &e.Timestamp,
			Data:                            e.Emit(),
			OptionalTrustedPacketSequenceId: &pp.TracePacket_TrustedPacketSequenceId{t.TID},
		})
}

func Test() {
	trace := Trace{TID: 32}
	trace.AddProcess(1, "my process")
	t1 := trace.AddThread(1, 2, "thread1")
	t2 := trace.AddThread(1, 3, "thread2")

	trace.AddEvent(t1.StartSlice(200, "download"))
	trace.AddEvent(t1.InstantEvent(400, "event1"))
	trace.AddEvent(t1.StartSlice(500, "process"))
	trace.AddEvent(t1.EndSlice(800))
	trace.AddEvent(t1.EndSlice(1000))

	trace.AddEvent(t2.StartSlice(300, "slice2"))
	trace.AddEvent(t2.InstantEvent(500, "event2"))
	trace.AddEvent(t2.EndSlice(800))

	data, err := proto.Marshal(&trace.Pt)
	if err != nil {
		panic(err)
	}

	var rtt pp.Trace
	err = proto.Unmarshal(data, &rtt)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", rtt.Packet[4].GetTrackEvent().GetName())
	os.WriteFile("test.bin", data, 0666)
}
