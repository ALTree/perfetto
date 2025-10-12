package main

import (
	"fmt"
	"os"

	"github.com/ALTree/perfetto"
)

func main() {
	trace := perfetto.NewTrace()
	glb := perfetto.GlobalTrack()
	trace.AddProcess(1, "Process #1")
	t1 := trace.AddThread(1, 1, "Thread #1")
	t2 := trace.AddThread(1, 2, "Thread #2")
	t3 := trace.AddTrack("HTTP Request")
	t4 := trace.AddTrack("HTTP Request")
	cpu := trace.AddCounter("cpu load", "%")

	stack := []perfetto.KV{
		{"1", "func1"},
		{"2", "func2"},
		{"3", "func3"},
	}

	trace.StartSlice(t3, 100, "Request 1")
	trace.EndSlice(t3, 200)
	trace.StartSlice(t4, 150, "Request 2")
	trace.EndSlice(t4, 250)
	trace.StartSlice(t3, 230, "Request 3")
	trace.EndSlice(t3, 300)

	for i := range uint64(50) {
		trace.StartSlice(glb, i*100, "global func")
		trace.EndSlice(glb, i*100+50)
	}

	for i := range uint64(100) {
		trace.StartSlice(t1, i*100, "func1", stack)
		trace.EndSlice(t1, i*100+50)
		trace.InstantEvent(t1, i*100+60, "Instant event")

		trace.StartSlice(t2, i*90, "func2")
		trace.StartSlice(t2, i*90+10, "func2a")
		trace.EndSlice(t2, i*90+40)
		trace.StartSlice(t2, i*90+40, "func2b")
		trace.EndSlice(t2, i*90+80)
		trace.EndSlice(t2, i*90+80)

		trace.NewValue(cpu, 100*i, int64(i))
	}

	data, err := trace.Marshal()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = os.WriteFile("trace.proto", data, 0666)
	if err != nil {
		fmt.Println(err)
	}
}
