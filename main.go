package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var database map[string]json.RawMessage

//go:embed module.wasm
var wasm []byte

func main() {
	userID, meetingID, perm, err := parseArgs(os.Args)
	if err != nil {
		fmt.Printf("wrong call: %v\n", err)
		os.Exit(2)
	}

	if err := run(userID, meetingID, perm); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(args []string) (int, int, string, error) {
	if len(args) < 4 {
		return 0, 0, "", fmt.Errorf("Run: %s USER_ID MEETING_ID PERM", args[0])
	}

	userID, err := strconv.Atoi(args[1])
	if err != nil {
		return 0, 0, "", fmt.Errorf("USERID has to be int")
	}

	meetingID, err := strconv.Atoi(args[2])
	if err != nil {
		return 0, 0, "", fmt.Errorf("USERID has to be int")
	}

	return userID, meetingID, args[3], nil
}

func run(userID int, meetingID int, perm string) error {
	if err := initDB(); err != nil {
		return fmt.Errorf("init database: %w", err)
	}

	wasmRuntime, close, err := newWasmRuntime(wasm)
	if err != nil {
		return fmt.Errorf("creating wasm runtime: %w", err)
	}
	defer close()

	canSee, err := wasmRuntime.HasPerm(userID, meetingID, perm)
	if err != nil {
		return fmt.Errorf("calling HasPerm: %w", err)
	}

	attr := "has not"
	if canSee {
		attr = "has"
	}

	fmt.Printf("user %d %s %s\n", userID, attr, perm)

	return nil
}

func initDB() error {
	file, err := os.Open("db.json")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&database); err != nil {
		return fmt.Errorf("decoding database: %w", err)
	}

	return nil
}

type wasmRuntime struct {
	mu sync.Mutex

	wasmRuntime wazero.Runtime
	memory      api.Memory

	hasPerm api.Function
	malloc  api.Function
	free    api.Function
}

func newWasmRuntime(wasm []byte) (*wasmRuntime, func(), error) {
	ctx := context.TODO()
	wazRuntime := wazero.NewRuntime(ctx)

	var r wasmRuntime

	_, err := wazRuntime.NewHostModuleBuilder("app").
		NewFunctionBuilder().WithFunc(r.getData).Export("getData").
		Instantiate(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("create host module: %w", err)
	}

	module, err := wazRuntime.Instantiate(ctx, wasm)
	if err != nil {
		return nil, nil, fmt.Errorf("instantiate: %w", err)
	}

	r.memory = module.Memory()

	r.hasPerm = module.ExportedFunction("hasPerm")
	if r.hasPerm == nil {
		return nil, nil, fmt.Errorf("can not find function hasPerm")
	}

	r.malloc = module.ExportedFunction("malloc")
	if r.malloc == nil {
		return nil, nil, fmt.Errorf("can not find function malloc")
	}

	r.free = module.ExportedFunction("free")
	if r.free == nil {
		return nil, nil, fmt.Errorf("can not find function free")
	}

	close := func() {
		wazRuntime.Close(ctx)
	}

	return &r, close, nil
}

func (r *wasmRuntime) HasPerm(userID int, meetingID int, perm string) (bool, error) {
	ctx := context.TODO()

	r.mu.Lock()
	defer r.mu.Unlock()

	permPtr, free, err := r.sendString(ctx, perm)
	if err != nil {
		return false, fmt.Errorf("sending string: %w", err)
	}
	defer free(ctx)

	results, err := r.hasPerm.Call(ctx, uint64(userID), uint64(meetingID), permPtr)
	if err != nil {
		return false, fmt.Errorf("calling wasm: %w", err)
	}

	switch results[0] {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, fmt.Errorf("wasm returned %d", results[0])
	}
}

func (r *wasmRuntime) getData(keyPointer uint32) uint32 {
	key, err := r.readString(context.TODO(), keyPointer)
	if err != nil {
		panic(err)
	}

	value, ok := database[key]
	if !ok {
		value = []byte("null")
	}

	resultPtr, _, err := r.sendString(context.TODO(), string(value))
	if err != nil {
		panic(err)
	}

	return uint32(resultPtr)
}

func (r *wasmRuntime) sendString(ctx context.Context, str string) (uint64, func(context.Context), error) {
	length := uint64(len(str) + 1)

	mallocResults, err := r.malloc.Call(ctx, length)
	if err != nil {
		return 0, nil, fmt.Errorf("calling malloc: %w", err)
	}

	ptr := uint32(mallocResults[0])
	free := func(ctx context.Context) {
		r.free.Call(ctx, uint64(ptr), length)
	}

	r.memory.Write(ptr, []byte(str))
	r.memory.WriteByte(uint32(length)-1, 0)

	return uint64(ptr), free, nil
}

func (r *wasmRuntime) readString(ctx context.Context, ptr uint32) (string, error) {
	p, ok := r.memory.ReadUint32Le(ptr)
	if !ok {
		return "", fmt.Errorf("can not read memory")
	}
	l, ok := r.memory.ReadUint32Le(ptr + 4)
	if !ok {
		return "", fmt.Errorf("can not read memory")
	}
	str, ok := r.memory.Read(p, l)
	if !ok {
		return "", fmt.Errorf("can not read memory")
	}

	return string(str), nil
}
