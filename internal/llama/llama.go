package llama

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/llama.cpp/include -I${SRCDIR}/../../third_party/llama.cpp/src -I${SRCDIR}/../../third_party/llama.cpp/ggml/include
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/../../third_party/llama.cpp -I${SRCDIR}/../../third_party/llama.cpp/include -I${SRCDIR}/../../third_party/llama.cpp/ggml/include

#cgo darwin LDFLAGS: ${SRCDIR}/../../third_party/llama.cpp/build/src/libllama.a ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/libggml-base.a ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/libggml-cpu.a ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/libggml.a ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/ggml-blas/libggml-blas.a ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/ggml-metal/libggml-metal.a -lm -framework Accelerate -framework Metal -framework Foundation

#cgo linux LDFLAGS: ${SRCDIR}/../../third_party/llama.cpp/build/src/libllama.a ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/libggml.a ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/libggml-cpu.a ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/libggml-base.a -lm -lpthread -ldl -lstdc++ -lgomp

#cgo windows LDFLAGS: ${SRCDIR}/../../third_party/llama.cpp/build/src/Release/llama.lib ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/Release/ggml.lib ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/Release/ggml-cpu.lib ${SRCDIR}/../../third_party/llama.cpp/build/ggml/src/Release/ggml-base.lib -lstdc++

#include "llama_wrapper.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

type Model struct {
	h *C.LlamaModelHandle
}

// PredictOptions controls generation
type PredictOptions struct {
	MaxTokens int
	Temp      float32
	TopK      int
	TopP      float32
}

// Note: we currently use the non-streaming C API (llama_predict) provided by the
// wrapper. If streaming is added, define and export a Go callback matching
// the C signature (llama_stream_callback) and ensure the cgo preamble and
// build flags are correct.

// LoadModel loads a GGUF model at modelPath and returns a Model.
// nThreads sets how many CPU threads to use (0 = default).
func LoadModel(modelPath string, nThreads int) (*Model, error) {
	cpath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cpath))

	h := C.llama_load_model(cpath, C.int(nThreads))
	if h == nil {
		return nil, errors.New("failed to load model (see llama_wrapper.c for details)")
	}
	m := &Model{h: h}
	// Make sure finalizer closes model if GC collects it
	runtime.SetFinalizer(m, func(m *Model) { m.Close() })
	return m, nil
}

// Predict runs the model and returns the text output
func (m *Model) Predict(prompt string, opts PredictOptions) (string, error) {
	if m == nil || m.h == nil {
		return "", errors.New("model is nil")
	}
	cprompt := C.CString(prompt)
	defer C.free(unsafe.Pointer(cprompt))

	// Reset the model context before each prediction to ensure a fresh
	// KV cache and avoid sequence position mismatches when reusing the
	// same model handle for multiple independent predictions.
	C.llama_reset_context(m.h)

	cres := C.llama_predict(m.h, cprompt, C.int(opts.MaxTokens), C.float(opts.Temp), C.int(opts.TopK), C.float(opts.TopP))

	if cres == nil {
		return "", errors.New("prediction failed")
	}
	defer C.llama_free_string(cres)
	goStr := C.GoString(cres)
	return goStr, nil
}

func (m *Model) Close() {
	if m == nil || m.h == nil {
		return
	}
	C.llama_close_model(m.h)
	m.h = nil
}
