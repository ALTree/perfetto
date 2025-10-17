package perfetto

import (
	"fmt"
	"slices"
	"testing"

	pp "github.com/ALTree/perfetto/internal/proto"
	"google.golang.org/protobuf/proto"
)

// Adding a single basic track to the trace
func TestAddBasicTrack(t *testing.T) {
	trace := NewTrace()
	trace.AddTrack("track #1")
	tr := RoundTrip(t, trace)

	AssertEq("trace length", t, len(tr.Packet), 2)
	AssertEq("Name", t, BasicTrackName(tr.Packet[1]), "track #1")
}

// Adding a single process to the trace
func TestAddProcess(t *testing.T) {
	trace := NewTrace()
	trace.AddProcess(1, "process #6")
	tr := RoundTrip(t, trace)

	AssertEq("trace length", t, len(tr.Packet), 2)
	AssertEq("Name", t, ProcessName(tr.Packet[1]), "process #6")
	AssertEq("Pid", t, ProcessPid(tr.Packet[1]), 1)
}

// Adding a process and a few threads to the trace
func TestAddThread(t *testing.T) {
	trace := NewTrace()
	trace.AddProcess(1, "process #6")
	trace.AddThread(1, 20, "thread #1")
	trace.AddThread(1, 30, "thread #2")
	tr := RoundTrip(t, trace)

	AssertEq("len(Threads)", t, len(trace.Threads), 2)
	AssertEq("trace length", t, len(tr.Packet), 4)

	t1, t2 := tr.Packet[2], tr.Packet[3]
	AssertEq("Thread #1 Name", t, ThreadName(t1), "thread #1")
	AssertEq("Thread #1 Tid", t, ThreadTid(t1), 20)
	AssertEq("Thread #1 Pid", t, ThreadPid(t1), 1)
	AssertEq("Thread #2 Name", t, ThreadName(t2), "thread #2")
	AssertEq("Thread #2 Tid", t, ThreadTid(t2), 30)
	AssertEq("Thread #2 Pid", t, ThreadPid(t2), 1)
}

// Adding several threads
func TestAddManyThreads(t *testing.T) {
	trace := NewTrace()
	trace.AddProcess(1, "process")
	for i := range 100 {
		trace.AddThread(1, int32(i), fmt.Sprintf("Thread #%v", i))
	}
	tr := RoundTrip(t, trace)

	AssertEq("len(Threads)", t, len(trace.Threads), 100)
	AssertEq("trace length", t, len(tr.Packet), 102)
	packets := tr.Packet[2:]
	for i := range int32(100) {
		tr := packets[i]
		AssertEq("Thread Name", t, ThreadName(tr), fmt.Sprintf("Thread #%v", i))
		AssertEq("Thread Tid", t, ThreadTid(tr), i)
		AssertEq("Thread Pid", t, ThreadPid(tr), 1)
	}
}

// Adding an Instant Event (to process and to Thread)
func TestInstantEvent(t *testing.T) {
	trace := NewTrace()
	p := trace.AddProcess(1, "process #1")
	t1 := trace.AddThread(1, 2, "Thread #1")

	trace.InstantEvent(p, 500, "Process Instant Event")
	trace.InstantEvent(t1, 1000, "Thread Instant Event")

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 5)

	pe := tr.Packet[3]
	AssertEq("Timestamp", t, EventTimestamp(pe), 500)
	AssertEq("Name", t, EventName(pe), "")
	AssertNeq("Name", t, EventNameIid(pe), 0)
	AssertEq("Type", t, EventType(pe), "TYPE_INSTANT")
	AssertEq("Track UUID", t, EventTrackUuid(pe), p.Uuid)

	te := tr.Packet[4]
	AssertEq("Timestamp", t, EventTimestamp(te), 500) // incremental
	AssertEq("Name", t, EventName(te), "")
	AssertNeq("Name", t, EventNameIid(te), 0)
	AssertEq("Type", t, EventType(te), "TYPE_INSTANT")
	AssertEq("Track UUID", t, EventTrackUuid(te), t1.Uuid)
}

// Adding a Slice Event (to process and to Thread)
func TestSliceEvent(t *testing.T) {
	trace := NewTrace(Features{Interning: true, IncrementalTS: false})
	p := trace.AddProcess(1, "process #1")
	t1 := trace.AddThread(1, 1, "Thread #1")

	trace.StartSlice(p, 500, "Process Slice")
	trace.EndSlice(p, 1000)
	trace.StartSlice(t1, 1000, "Thread Slice")
	trace.EndSlice(t1, 1500)

	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 6)

	// the Process slice
	pe1, pe2 := tr.Packet[2], tr.Packet[3]
	AssertEq("start Timestamp", t, EventTimestamp(pe1), 500)
	AssertEq("start Name", t, EventName(pe1), "") // interned
	AssertEq("start Name Iid", t, EventNameIid(pe1), 1)
	AssertEq("start Type", t, EventType(pe1), "TYPE_SLICE_BEGIN")
	AssertEq("start Track UUID", t, EventTrackUuid(pe1), p.Uuid)
	AssertEq("end Timestamp", t, EventTimestamp(pe2), 1000)
	AssertEq("end Name", t, EventName(pe2), "")
	AssertEq("start Name Iid", t, EventNameIid(pe2), 0)
	AssertEq("end Type", t, EventType(pe2), "TYPE_SLICE_END")
	AssertEq("end Track UUID", t, EventTrackUuid(pe2), p.Uuid)

	// the Thread slice
	pe1, pe2 = tr.Packet[4], tr.Packet[5]
	AssertEq("start Timestamp", t, EventTimestamp(pe1), 1000)
	AssertEq("start Name", t, EventName(pe1), "") // interned
	AssertEq("start Name Iid", t, EventNameIid(pe1), 2)
	AssertEq("start Type", t, EventType(pe1), "TYPE_SLICE_BEGIN")
	AssertEq("start Track UUID", t, EventTrackUuid(pe1), t1.Uuid)
	AssertEq("end Timestamp", t, EventTimestamp(pe2), 1500)
	AssertEq("end Name", t, EventName(pe2), "")
	AssertEq("start Name Iid", t, EventNameIid(pe2), 0)
	AssertEq("end Type", t, EventType(pe2), "TYPE_SLICE_END")
	AssertEq("end Track UUID", t, EventTrackUuid(pe2), t1.Uuid)
}

// Adding a Counter track
func TestCounter(t *testing.T) {
	trace := NewTrace(Features{Interning: true, IncrementalTS: false})
	trace.AddProcess(1, "process #1")
	cpuload := trace.AddCounter("cpu load", "%")

	for i := range uint64(10) {
		trace.NewValue(cpuload, 100*i, int64(10*i))
	}

	tr := RoundTrip(t, trace)
	AssertEq("Counter tracks", t, len(trace.Counters), 1)
	AssertEq("trace length", t, len(tr.Packet), 2+10)

	p := tr.Packet[1]
	AssertEq("Counter Track name", t, CounterTrackName(p), "cpu load")

	packets := tr.Packet[2:]
	for i := range uint64(10) {
		p := packets[i]
		AssertEq("Timestamp", t, EventTimestamp(p), 100*i)
		AssertEq("Name", t, EventName(p), "") // interned
		AssertEq("Type", t, EventType(p), "TYPE_COUNTER")
		AssertEq("Value", t, EventValue(p), int64(10*i))
		AssertEq("Track UUID", t, EventTrackUuid(p), cpuload.Uuid)
	}
}

// Returns a 1-process, 2-threads trace, with 100 slice events and 10
// instant events.
func AddManyEvents(t *testing.T, feat ...Features) Trace {
	t.Helper()
	var trace Trace
	if len(feat) > 0 {
		trace = NewTrace(feat[0])
	} else {
		trace = NewTrace()
	}

	trace.AddProcess(1, "process #1")
	t1 := trace.AddThread(1, 1, "Thread #1")
	t2 := trace.AddThread(1, 2, "Thread #2")

	// Add 100 events, alternating between thread 1 and 2
	for i := range uint64(100) {
		if i%2 == 0 {
			trace.StartSlice(t1, i*100, "t1 func")
			trace.EndSlice(t1, i*100+50)
		} else {
			trace.StartSlice(t2, i*100, "t2 func")
			trace.EndSlice(t2, i*100+50)
		}
	}

	// Add 10 instant events
	for i := range uint64(10) {
		trace.InstantEvent(t1, i*100, "Instant event")
	}

	return trace
}

func TestManyEvents(t *testing.T) {
	t.Run("DefaultFeatures", ManyEventsDefaultFeatures)
	t.Run("NoInterning", ManyEventsNoInterning)
	t.Run("FullTimestamps", ManyEventsFullTimestamps)
}

func ManyEventsDefaultFeatures(t *testing.T) {
	trace := AddManyEvents(t)
	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 4+2*100+10)

	// Check slice events
	packets := tr.Packet[4:]
	for i := range uint64(200) {
		p := packets[i]
		ts, typ := func() (uint64, string) {
			switch i % 4 {
			case 0:
				var gap = uint64(50)
				if i == 0 {
					gap = 0
				}
				return gap, "TYPE_SLICE_BEGIN"
			case 1:
				return 50, "TYPE_SLICE_END"
			case 2:
				return 50, "TYPE_SLICE_BEGIN"
			default:
				return 50, "TYPE_SLICE_END"
			}
		}()
		AssertEq("Timestamp", t, EventTimestamp(p), ts)
		AssertEq("Name", t, EventName(p), "") // names are interned
		if typ == "TYPE_SLICE_BEGIN" {
			AssertNeq("NameIid", t, EventNameIid(p), 0)
		}
		AssertEq("Type", t, EventType(p), typ)
	}

	// Check Instant events
	packets = tr.Packet[4+200:]
	for i := range uint64(10) {
		p := packets[i]
		// These instant events have timestamps smaller than
		// Trace.LastTimestamp, so we expect Emit() to give up
		// emitting incremental timestamps, and use the default clock.
		AssertEq("Timestamp", t, EventTimestamp(p), 100*i)
		AssertEq("Name", t, EventName(p), "") // names are interned
		AssertNeq("NameIid", t, EventNameIid(p), 0)
		AssertEq("Type", t, EventType(p), "TYPE_INSTANT")
	}
}

func ManyEventsNoInterning(t *testing.T) {
	feat := DefaultFeatures
	feat.Interning = false
	trace := AddManyEvents(t, feat)
	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 4+2*100+10)

	// Check slice events
	packets := tr.Packet[4:]
	for i := range uint64(200) {
		p := packets[i]
		ts, typ, name := func() (uint64, string, string) {
			switch i % 4 {
			case 0:
				var gap = uint64(50)
				if i == 0 {
					gap = 0
				}
				return gap, "TYPE_SLICE_BEGIN", "t1 func"
			case 1:
				return 50, "TYPE_SLICE_END", ""
			case 2:
				return 50, "TYPE_SLICE_BEGIN", "t2 func"
			default:
				return 50, "TYPE_SLICE_END", ""
			}
		}()
		AssertEq("Timestamp", t, EventTimestamp(p), ts)
		AssertEq("Name", t, EventName(p), name)
		AssertEq("NameIid", t, EventNameIid(p), 0) // no interning
		AssertEq("Type", t, EventType(p), typ)
	}

	// Check Instant events
	packets = tr.Packet[4+200:]
	for i := range uint64(10) {
		p := packets[i]
		AssertEq("Timestamp", t, EventTimestamp(p), 100*i)
		AssertEq("Name", t, EventName(p), "Instant event")
		AssertEq("NameIid", t, EventNameIid(p), 0) // no interning
		AssertEq("Type", t, EventType(p), "TYPE_INSTANT")
	}
}

func ManyEventsFullTimestamps(t *testing.T) {
	feat := DefaultFeatures
	feat.IncrementalTS = false
	trace := AddManyEvents(t, feat)
	tr := RoundTrip(t, trace)
	AssertEq("trace length", t, len(tr.Packet), 3+2*100+10)

	// Check slice events
	packets := tr.Packet[3:]
	for i := range uint64(200) {
		p := packets[i]
		ts, typ := func() (uint64, string) {
			j := i / 2
			switch i % 4 {
			case 0:
				return j * 100, "TYPE_SLICE_BEGIN"
			case 1:
				return j*100 + 50, "TYPE_SLICE_END"
			case 2:
				return j * 100, "TYPE_SLICE_BEGIN"
			default:
				return j*100 + 50, "TYPE_SLICE_END"
			}
		}()
		AssertEq("Timestamp", t, EventTimestamp(p), ts)
		AssertEq("Name", t, EventName(p), "") // names are interned
		if typ == "TYPE_SLICE_BEGIN" {
			AssertNeq("NameIid", t, EventNameIid(p), 0)
		}
		AssertEq("Type", t, EventType(p), typ)
	}

	// Check Instant events
	packets = tr.Packet[3+200:]
	for i := range uint64(10) {
		p := packets[i]
		AssertEq("Timestamp", t, EventTimestamp(p), 100*i)
		AssertEq("Name", t, EventName(p), "") // names are interned
		AssertNeq("NameIid", t, EventNameIid(p), 0)
		AssertEq("Type", t, EventType(p), "TYPE_INSTANT")
	}
}

func TestAnnotations(t *testing.T) {
	trace := NewTrace(Features{Interning: false})
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
	AssertEq("trace length", t, len(tr.Packet), 5)
	if got := EventFlows(tr.Packet[3]); !slices.Equal(got, flows) {
		t.Errorf("For %s\ngot %v\nexp %v", "Flows", got, flows)
	}
	if got := EventFlows(tr.Packet[4]); !slices.Equal(got, flows) {
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

func BasicTrackName(p *pp.TracePacket) string {
	return p.GetTrackDescriptor().GetName()
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
