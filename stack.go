package zerrors

import (
	"fmt"
	"runtime"
	"strings"
)

const defaultStackDepth = 32

type stackFrame struct {
	pc       uintptr
	file     string
	function string
	line     int
}

func (f *stackFrame) String() string {
	if f.function != "" {
		return fmt.Sprintf("%s:%d %s()", f.file, f.line, f.function)
	}
	return fmt.Sprintf("%s:%d", f.file, f.line)
}

type stack struct {
	frames []stackFrame
}

func (s *stack) String() string {
	var sb strings.Builder
	for _, frame := range s.frames {
		sb.WriteString("\n    at ")
		sb.WriteString(frame.String())
	}
	return sb.String()
}

// Capture a new stacktrace, skipping the first 'skip' frames.
func captureStack(skip int) *stack {
	pcs := make([]uintptr, defaultStackDepth)
	n := runtime.Callers(skip+1, pcs)
	if n == 0 {
		return nil
	}

	frames := make([]stackFrame, 0, n)
	iter := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := iter.Next()
		if !more {
			break
		}

		// Skip runtime frames and testing frames
		if strings.Contains(frame.File, "runtime/") || strings.Contains(frame.File, "_test.go") {
			continue
		}

		frames = append(frames, stackFrame{
			pc:       frame.PC,
			file:     trimGoPath(frame.File),
			function: trimFuncName(frame.Function),
			line:     frame.Line,
		})
	}

	return &stack{frames: frames}
}

// Helper function to trim the GOPATH from file paths.
func trimGoPath(path string) string {
	if i := strings.LastIndex(path, "/go/src/"); i != -1 {
		return path[i+8:]
	}
	return path
}

// Helper function to get the short function name.
func trimFuncName(name string) string {
	if i := strings.LastIndex(name, "/"); i != -1 {
		name = name[i+1:]
	}
	if i := strings.Index(name, "."); i != -1 {
		name = name[i+1:]
	}
	return name
}
