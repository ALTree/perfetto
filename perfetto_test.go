package perfetto

import (
	"fmt"
	"slices"
	"testing"

	pp "github.com/ALTree/perfetto/internal/proto"
	"google.golang.org/protobuf/proto"
)

func TestAddProcess(t *testing.T) {
	trace := NewTrace()
	trace.AddProcess(1, "process #6")
	tr := RoundTrip(t, trace)

	AssertEq("trace length", t, len(tr.Packet), 1)
	AssertEq("Name", t, ProcessName(tr.Packet[0]), "process #6")
	AssertEq("Pid", t, ProcessPid(tr.Packet[0]), 1)
}

func TestAddThread(t *testing.T) {
	trace := NewTrace()
	trace.AddProcess(1, "process #6")
	trace.AddThread(1, 2, "thread #1")
	trace.AddThread(1, 3, "thread #2")

	AssertEq("len(Threads)", t, len(trace.Threads), 2)

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
	trace := NewTrace()
	trace.AddProcess(1, "process #3")
	for i := range 100 {
		trace.AddThread(1, int32(i), fmt.Sprintf("Thread #%v", i))
	}
	AssertEq("len(Threads)", t, len(trace.Threads), 100)

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
	trace := NewTrace()
	trace.AddProcess(1, "process #1")
	thr := trace.AddThread(1, 2, "Thread #1")

	trace.InstantEvent(thr, 500, "Event #1")

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 3)
	p := tr.Packet[2]
	AssertEq("Timestamp", t, EventTimestamp(p), 500)
	AssertEq("Name", t, EventName(p), "")
	AssertNeq("Name", t, EventNameIid(p), 0)
	AssertEq("Type", t, EventType(p), "TYPE_INSTANT")
	AssertEq("Track UUID", t, EventTrackUuid(p), thr.Uuid)

}

func TestSliceEvent(t *testing.T) {
	debug.DisableInterning = true
	defer func() { debug.DisableInterning = false }()

	trace := NewTrace()
	trace.AddProcess(1, "process #1")
	thr := trace.AddThread(1, 2, "Thread #1")

	trace.StartSlice(thr, 1000, "Slice #1")
	trace.EndSlice(thr, 1500)

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 4)
	p1, p2 := tr.Packet[2], tr.Packet[3]

	AssertEq("start Timestamp", t, EventTimestamp(p1), 1000)
	AssertEq("start Name", t, EventName(p1), "Slice #1")
	AssertEq("start Name Iid", t, EventNameIid(p1), 0)
	AssertEq("start Type", t, EventType(p1), "TYPE_SLICE_BEGIN")
	AssertEq("start Track UUID", t, EventTrackUuid(p1), thr.Uuid)

	AssertEq("end Timestamp", t, EventTimestamp(p2), 1500)
	AssertEq("end Name", t, EventName(p2), "")
	AssertEq("start Name Iid", t, EventNameIid(p2), 0)
	AssertEq("end Type", t, EventType(p2), "TYPE_SLICE_END")
	AssertEq("end Track UUID", t, EventTrackUuid(p2), thr.Uuid)
}

func TestCounter(t *testing.T) {
	trace := NewTrace()
	trace.AddProcess(1, "process #1")
	cpuload := trace.AddCounter("cpu load", "%")
	AssertEq("Counter tracks", t, len(trace.Counters), 1)
	for i := range uint64(10) {
		trace.NewValue(cpuload, 100*i, int64(10*i))
	}

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 2+10)
	cpuPacket := tr.Packet[1]
	AssertEq("Counter Track name", t, CounterTrackName(cpuPacket), "cpu load")

	packets := tr.Packet[2:]
	for i := range uint64(10) {
		p := packets[i]
		AssertEq("Timestamp", t, EventTimestamp(p), 100*i)
		AssertEq("Name", t, EventName(p), "")
		AssertEq("Type", t, EventType(p), "TYPE_COUNTER")
		AssertEq("Value", t, EventType(p), "TYPE_COUNTER")
		AssertEq("Track UUID", t, EventTrackUuid(p), cpuload.Uuid)
	}
}

func TestManyEvents(t *testing.T) {
	trace := NewTrace()
	trace.AddProcess(1, "process #1")
	thr1 := trace.AddThread(1, 1, "Thread #1")
	thr2 := trace.AddThread(1, 2, "Thread #2")

	for i := range uint64(100) {
		if i%2 == 0 {
			trace.StartSlice(thr1, i*100, "thr1 func")
			trace.EndSlice(thr1, i*100+50)
		} else {
			trace.StartSlice(thr2, i*100, "thr2 func")
			trace.EndSlice(thr2, i*100+50)
		}
	}

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 3+2*100)

	packets := tr.Packet[3:]
	for i := range uint64(200) {
		p := packets[i]
		ts, typ, uuid := func() (uint64, string, uint64) {
			j := i / 2
			switch i % 4 {
			case 0:
				return j * 100, "TYPE_SLICE_BEGIN", thr1.Uuid
			case 1:
				return j*100 + 50, "TYPE_SLICE_END", thr1.Uuid
			case 2:
				return j * 100, "TYPE_SLICE_BEGIN", thr2.Uuid
			default:
				return j*100 + 50, "TYPE_SLICE_END", thr2.Uuid
			}
		}()
		AssertEq("Timestamp", t, EventTimestamp(p), ts)
		AssertEq("Name", t, EventName(p), "") // names are interned
		if typ == "TYPE_SLICE_BEGIN" {
			if uuid == thr1.Uuid {
				AssertEq("NameIid", t, EventNameIid(p), 1) // Interning ID is 1
			} else if uuid == thr2.Uuid {
				AssertEq("NameIid", t, EventNameIid(p), 2) // Interning ID is 2
			}
		}
		AssertEq("Type", t, EventType(p), typ)
		AssertEq("Track UUID", t, EventTrackUuid(p), uuid)
	}
}

func TestAnnotations(t *testing.T) {
	trace := NewTrace()
	trace.AddProcess(1, "process #1")
	t1 := trace.AddThread(1, 2, "Thread #1")

	ann := []KV{{"k1", "v1"}, {"k2", "v2"}, {"k3", "v3"}}
	trace.StartSlice(t1, 100, "t1 func", ann)
	trace.EndSlice(t1, 150)

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 4)
	if got := EventAnnotations(tr.Packet[2]); !slices.Equal(ann, got) {
		t.Errorf("For %s\ngot %v\nexp %v", "Annotations", got, ann)
	}
}

func TestFlows(t *testing.T) {
	trace := NewTrace()
	trace.AddProcess(1, "process #1")
	t1 := trace.AddThread(1, 1, "Thread #1")

	flows := []uint64{1, 2, 3}
	trace.StartSliceWithFlow(t1, 100, "t1 func", flows)
	trace.EndSliceWithFlow(t1, 150, flows)

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 4)
	if got := EventFlows(tr.Packet[2]); !slices.Equal(got, flows) {
		t.Errorf("For %s\ngot %v\nexp %v", "Flows", got, flows)
	}
	if got := EventFlows(tr.Packet[3]); !slices.Equal(got, flows) {
		t.Errorf("For %s\ngot %v\nexp %v", "Flows", got, flows)
	}
}

// ---- { testing helpers } --------------------------------

func RoundTrip(t *testing.T, trace Trace) pp.Trace {
	data, err := trace.Marshal()
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

func EventNameIid(p *pp.TracePacket) uint64 {
	return p.GetTrackEvent().GetNameIid()
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

func EventFlows(p *pp.TracePacket) []uint64 {
	return p.GetTrackEvent().GetFlowIds()
}

func EventAnnotations(p *pp.TracePacket) []KV {
	var res []KV
	for _, a := range p.GetTrackEvent().GetDebugAnnotations() {
		res = append(res, KV{a.GetName(), a.GetStringValue()})
	}
	return res
}

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
