# MutliNWriter
This package provides an alternative to io.MultiWriter that enables dynamic addition and removal of io.Writers at run time.

# Reasoning

[io.Multiwriter](https://pkg.go.dev/io#MultiWriter) is a great tool for when you have a fixed at compile time set of [io.Writers](https://pkg.go.dev/io#Writer) you want to duplicate writes to such as writing [exec.Cmd](https://pkg.go.dev/os/exec#Cmd) output to a file _and_ stdout at the same time. What it can't do is allow for a dynamic set of Writers to be added during runtime.

This is where MultiNWriter comes in. It uses a mutex and a generic keyed map of io.Writers to enable N number of copied writes through a simple interface.

# Getting started

```
multiWriter := MutliNWriter.NewMultiNWriter()
multiWriter.AddWriter("key", writer)
multiWriter.AddWriter("key2", writer2)
```

That's all you need to get up and running with dynamic duplicate writers at runtime!

# Example
An example can be found [here](./examples/simple/main.go) which shows a simulated long running task which can be monitored from N number of websockets in real time.
