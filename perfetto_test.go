package perfetto

import (
	"fmt"
	"testing"

	pp "github.com/ALTree/perfetto/internal/proto"
	"google.golang.org/protobuf/proto"
)

func RoundTrip(t *testing.T, trace Trace) pp.Trace {
	data, err := proto.Marshal(&trace.Pt)
	if err != nil {
		t.Fatal(err)
	}

	var rtt pp.Trace
	err = proto.Unmarshal(data, &rtt)
	if err != nil {
		t.Fatal(err)
	}

	return rtt
}

func ProcessName(p *pp.TracePacket) string {
	return p.GetTrackDescriptor().GetProcess().GetProcessName()
}

func ProcessPid(p *pp.TracePacket) int32 {
	return p.GetTrackDescriptor().GetProcess().GetPid()
}

func ThreadName(p *pp.TracePacket) string {
	return p.GetTrackDescriptor().GetThread().GetThreadName()
}

func ThreadPid(p *pp.TracePacket) int32 {
	return p.GetTrackDescriptor().GetThread().GetPid()
}

func ThreadTid(p *pp.TracePacket) int32 {
	return p.GetTrackDescriptor().GetThread().GetTid()
}

func CounterTrackName(p *pp.TracePacket) string {
	trn, ok := p.GetTrackDescriptor().GetStaticOrDynamicName().(*pp.TrackDescriptor_Name)
	if !ok {
		return "ERROR: not a TrackDescriptor_Name"
	}
	return trn.Name
}

func EventTimestamp(p *pp.TracePacket) uint64 {
	return p.GetTimestamp()
}

func EventName(p *pp.TracePacket) string {
	return p.GetTrackEvent().GetName()
}

func EventValue(p *pp.TracePacket) int64 {
	return p.GetTrackEvent().GetCounterValue()
}

func EventType(p *pp.TracePacket) string {
	return p.GetTrackEvent().GetType().String()
}

func EventTrackUuid(p *pp.TracePacket) uint64 {
	return p.GetTrackEvent().GetTrackUuid()
}

func TestAddProcess(t *testing.T) {
	trace := Trace{TID: 32}
	trace.AddProcess(1, "process #6")
	tr := RoundTrip(t, trace)

	AssertEq("trace length", t, len(tr.Packet), 1)
	AssertEq("Name", t, ProcessName(tr.Packet[0]), "process #6")
	AssertEq("Pid", t, ProcessPid(tr.Packet[0]), 1)
}

func TestAddThread(t *testing.T) {
	trace := Trace{TID: 32}
	trace.AddProcess(1, "process #6")
	trace.AddThread(1, 2, "thread #1")
	trace.AddThread(1, 3, "thread #2")
	tr := RoundTrip(t, trace)

	AssertEq("trace length", t, len(tr.Packet), 3)

	t1, t2 := tr.Packet[1], tr.Packet[2]
	AssertEq("Thread #1 Name", t, ThreadName(t1), "thread #1")
	AssertEq("Thread #1 Tid", t, ThreadTid(t1), 2)
	AssertEq("Thread #1 Pid", t, ThreadPid(t1), 1)
	AssertEq("Thread #2 Name", t, ThreadName(t2), "thread #2")
	AssertEq("Thread #2 Tid", t, ThreadTid(t2), 3)
	AssertEq("Thread #2 Pid", t, ThreadPid(t2), 1)
}

func TestAddManyThreads(t *testing.T) {
	trace := Trace{TID: 32}
	trace.AddProcess(1, "process #3")
	for i := range 100 {
		trace.AddThread(1, int32(i), fmt.Sprintf("Thread #%v", i))
	}

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 101)
	for i := range int32(100) {
		tr := tr.Packet[i+1]
		AssertEq("Thread Name", t, ThreadName(tr), fmt.Sprintf("Thread #%v", i))
		AssertEq("Thread Tid", t, ThreadTid(tr), i)
		AssertEq("Thread Pid", t, ThreadPid(tr), 1)
	}
}

func TestInstantEvent(t *testing.T) {
	trace := Trace{TID: 32}
	trace.AddProcess(1, "process #1")
	thr := trace.AddThread(1, 2, "Thread #1")

	trace.AddEvent(thr.InstantEvent(500, "Event #1"))

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 3)
	p := tr.Packet[2]
	AssertEq("Timestamp", t, EventTimestamp(p), 500)
	AssertEq("Name", t, EventName(p), "Event #1")
	AssertEq("Type", t, EventType(p), "TYPE_INSTANT")
	AssertEq("Track UUID", t, EventTrackUuid(p), thr.Uuid)

}

func TestSliceEvent(t *testing.T) {
	trace := Trace{TID: 32}
	trace.AddProcess(1, "process #1")
	thr := trace.AddThread(1, 2, "Thread #1")

	trace.AddEvent(thr.StartSlice(1000, "Slice #1"))
	trace.AddEvent(thr.EndSlice(1500))

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 4)
	p1, p2 := tr.Packet[2], tr.Packet[3]
	AssertEq("start Timestamp", t, EventTimestamp(p1), 1000)
	AssertEq("start Name", t, EventName(p1), "Slice #1")
	AssertEq("start Type", t, EventType(p1), "TYPE_SLICE_BEGIN")
	AssertEq("start Track UUID", t, EventTrackUuid(p1), thr.Uuid)
	AssertEq("end Timestamp", t, EventTimestamp(p2), 1500)
	AssertEq("end Name", t, EventName(p2), "")
	AssertEq("end Type", t, EventType(p2), "TYPE_SLICE_END")
	AssertEq("end Track UUID", t, EventTrackUuid(p2), thr.Uuid)
}

func TestCounter(t *testing.T) {
	trace := Trace{TID: 32}
	trace.AddProcess(1, "process #1")
	cpuload := trace.AddCounter("cpu load", "%")

	for i := range uint64(10) {
		trace.AddEvent(cpuload.NewValue(100*i, int64(10*i)))
	}

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 2+10)
	cpuPacket := tr.Packet[1]
	AssertEq("Counter Track name", t, CounterTrackName(cpuPacket), "cpu load")

	packets := tr.Packet[2:]
	for i := range uint64(10) {
		p := packets[i]
		AssertEq("Timestamp", t, EventTimestamp(p), 100*i)
		AssertEq("Name", t, EventName(p), "cpu load")
		AssertEq("Type", t, EventType(p), "TYPE_COUNTER")
		AssertEq("Value", t, EventType(p), "TYPE_COUNTER")
		AssertEq("Track UUID", t, EventTrackUuid(p), cpuload.Uuid)
	}
}

func TestManyEvents(t *testing.T) {
	trace := Trace{TID: 32}
	trace.AddProcess(1, "process #1")
	thr1 := trace.AddThread(1, 2, "Thread #1")
	thr2 := trace.AddThread(1, 3, "Thread #2")

	for i := range uint64(100) {
		if i%2 == 0 {
			trace.AddEvent(thr1.StartSlice(i*100, "thr1 func"))
			trace.AddEvent(thr1.EndSlice(i*100 + 50))
		} else {
			trace.AddEvent(thr2.StartSlice(i*100, "thr2 func"))
			trace.AddEvent(thr2.EndSlice(i*100 + 50))
		}
	}

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 3+2*100)

	packets := tr.Packet[3:]
	for i := range uint64(200) {
		p := packets[i]
		ts, name, typ, uuid := func() (uint64, string, string, uint64) {
			j := i / 2
			switch i % 4 {
			case 0:
				return j * 100, "thr1 func", "TYPE_SLICE_BEGIN", thr1.Uuid
			case 1:
				return j*100 + 50, "", "TYPE_SLICE_END", thr1.Uuid
			case 2:
				return j * 100, "thr2 func", "TYPE_SLICE_BEGIN", thr2.Uuid
			default:
				return j*100 + 50, "", "TYPE_SLICE_END", thr2.Uuid
			}
		}()
		AssertEq("Timestamp", t, EventTimestamp(p), ts)
		AssertEq("Name", t, EventName(p), name)
		AssertEq("Type", t, EventType(p), typ)
		AssertEq("Track UUID", t, EventTrackUuid(p), uuid)
	}
}

// ---- { testing helpers } --------------------------------

func AssertEq[V comparable](fmt string, t *testing.T, got, exp V) {
	t.Helper()
	if exp != got {
		t.Errorf("For %s\ngot %v\nexp %v", fmt, got, exp)
	}
}

func AssertNeq[V comparable](fmt string, t *testing.T, got, exp V) {
	t.Helper()
	if exp == got {
		t.Errorf("For %s\ngot %v\nexp != %v", fmt, got, exp)
	}
}
