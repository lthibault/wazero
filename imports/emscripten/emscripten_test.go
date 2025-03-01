package emscripten

import (
	"bytes"
	"context"
	_ "embed"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/experimental/logging"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

// growWasm was compiled from testdata/grow.cc
//
//go:embed testdata/grow.wasm
var growWasm []byte

// invokeWasm was generated by the following:
//
//	cd testdata; wat2wasm --debug-names invoke.wat
//
//go:embed testdata/invoke.wasm
var invokeWasm []byte

// testCtx is an arbitrary, non-default context. Non-nil also prevents linter errors.
var testCtx = context.WithValue(context.Background(), struct{}{}, "arbitrary")

// TestGrow is an integration test until we have an Emscripten example.
func TestGrow(t *testing.T) {
	var log bytes.Buffer

	// Set context to one that has an experimental listener
	ctx := context.WithValue(testCtx, experimental.FunctionListenerFactoryKey{},
		logging.NewHostLoggingListenerFactory(&log, logging.LogScopeMemory))

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	_, err := Instantiate(ctx, r)
	require.NoError(t, err)

	// Emscripten exits main with zero by default, which coerces to nul.
	_, err = r.Instantiate(ctx, growWasm)
	require.Nil(t, err)

	// We expect the memory no-op memory growth hook to be invoked as wasm.
	require.Contains(t, log.String(), "==> env.emscripten_notify_memory_growth(memory_index=0)")
}

func TestInvoke(t *testing.T) {
	var log bytes.Buffer

	// Set context to one that has an experimental listener
	ctx := context.WithValue(testCtx, experimental.FunctionListenerFactoryKey{}, logging.NewLoggingListenerFactory(&log))

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	_, err := Instantiate(ctx, r)
	require.NoError(t, err)

	mod, err := r.Instantiate(ctx, invokeWasm)
	require.NoError(t, err)

	tests := []struct {
		name, funcName          string
		tableOffset             int
		params, expectedResults []uint64
		expectedLog             string
	}{
		{
			name:            "invoke_i",
			funcName:        "call_v_i32",
			expectedResults: []uint64{42},
			expectedLog: `--> .call_v_i32(0)
	==> env.invoke_i(index=0)
		--> .v_i32()
		<-- 42
	<== 42
<-- 42
`,
		},
		{
			name:            "invoke_ii",
			funcName:        "call_i32_i32",
			tableOffset:     2,
			params:          []uint64{42},
			expectedResults: []uint64{42},
			expectedLog: `--> .call_i32_i32(2,42)
	==> env.invoke_ii(index=2,a1=42)
		--> .i32_i32(42)
		<-- 42
	<== 42
<-- 42
`,
		},
		{
			name:            "invoke_iii",
			funcName:        "call_i32i32_i32",
			tableOffset:     4,
			params:          []uint64{1, 2},
			expectedResults: []uint64{3},
			expectedLog: `--> .call_i32i32_i32(4,1,2)
	==> env.invoke_iii(index=4,a1=1,a2=2)
		--> .i32i32_i32(1,2)
		<-- 3
	<== 3
<-- 3
`,
		},
		{
			name:            "invoke_iiii",
			funcName:        "call_i32i32i32_i32",
			tableOffset:     6,
			params:          []uint64{1, 2, 4},
			expectedResults: []uint64{7},
			expectedLog: `--> .call_i32i32i32_i32(6,1,2,4)
	==> env.invoke_iiii(index=6,a1=1,a2=2,a3=4)
		--> .i32i32i32_i32(1,2,4)
		<-- 7
	<== 7
<-- 7
`,
		},
		{
			name:            "invoke_iiiii",
			funcName:        "calli32_i32i32i32i32_i32",
			tableOffset:     8,
			params:          []uint64{1, 2, 4, 8},
			expectedResults: []uint64{15},
			expectedLog: `--> .calli32_i32i32i32i32_i32(8,1,2,4,8)
	==> env.invoke_iiiii(index=8,a1=1,a2=2,a3=4,a4=8)
		--> .i32i32i32i32_i32(1,2,4,8)
		<-- 15
	<== 15
<-- 15
`,
		},
		{
			name:        "invoke_v",
			funcName:    "call_v_v",
			tableOffset: 10,
			expectedLog: `--> .call_v_v(10)
	==> env.invoke_v(index=10)
		--> .v_v()
		<--
	<==
<--
`,
		},
		{
			name:        "invoke_vi",
			funcName:    "call_i32_v",
			tableOffset: 12,
			params:      []uint64{42},
			expectedLog: `--> .call_i32_v(12,42)
	==> env.invoke_vi(index=12,a1=42)
		--> .i32_v(42)
		<--
	<==
<--
`,
		},
		{
			name:        "invoke_vii",
			funcName:    "call_i32i32_v",
			tableOffset: 14,
			params:      []uint64{1, 2},
			expectedLog: `--> .call_i32i32_v(14,1,2)
	==> env.invoke_vii(index=14,a1=1,a2=2)
		--> .i32i32_v(1,2)
		<--
	<==
<--
`,
		},
		{
			name:        "invoke_viii",
			funcName:    "call_i32i32i32_v",
			tableOffset: 16,
			params:      []uint64{1, 2, 4},
			expectedLog: `--> .call_i32i32i32_v(16,1,2,4)
	==> env.invoke_viii(index=16,a1=1,a2=2,a3=4)
		--> .i32i32i32_v(1,2,4)
		<--
	<==
<--
`,
		},
		{
			name:        "invoke_viiii",
			funcName:    "calli32_i32i32i32i32_v",
			tableOffset: 18,
			params:      []uint64{1, 2, 4, 8},
			expectedLog: `--> .calli32_i32i32i32i32_v(18,1,2,4,8)
	==> env.invoke_viiii(index=18,a1=1,a2=2,a3=4,a4=8)
		--> .i32i32i32i32_v(1,2,4,8)
		<--
	<==
<--
`,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			defer log.Reset()

			params := tc.params
			params = append([]uint64{uint64(tc.tableOffset)}, params...)

			results, err := mod.ExportedFunction(tc.funcName).Call(testCtx, params...)
			require.NoError(t, err)
			require.Equal(t, tc.expectedResults, results)

			// We expect to see the dynamic function call target
			require.Equal(t, log.String(), tc.expectedLog)

			// We expect an unreachable function to err
			params[0]++
			_, err = mod.ExportedFunction(tc.funcName).Call(testCtx, params...)
			require.Error(t, err)
		})
	}
}
