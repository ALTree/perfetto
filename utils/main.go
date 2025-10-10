package main

import (
	"fmt"
	"os"

	"github.com/ALTree/perfetto"
)

func main() {
	trace := perfetto.NewTrace(1)
	trace.AddProcess(1, "Process #1")
	t1 := trace.AddThread(1, 2, "Thread #1")
	t2 := trace.AddThread(1, 3, "Thread #2")
	cpu := trace.AddCounter("cpu load", "%")

	stack := []perfetto.KV{
		{"level1", "func1"},
		{"level2", "func2"},
		{"level3", "func3"},
	}

	for i := range uint64(100) {
		trace.AddEvent(t1.StartSlice(i*100, "func1", stack))
		trace.AddEvent(t1.EndSlice(i*100 + 50))

		trace.AddEvent(t2.StartSlice(i*90, "func2"))
		trace.AddEvent(t2.StartSlice(i*90+10, "func2a"))
		trace.AddEvent(t2.EndSlice(i*90 + 40))
		trace.AddEvent(t2.StartSlice(i*90+40, "func2b"))
		trace.AddEvent(t2.EndSlice(i*90 + 80))
		trace.AddEvent(t2.EndSlice(i*90 + 80))

		trace.AddEvent(cpu.NewValue(100*i, int64(i)))
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
