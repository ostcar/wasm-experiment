# Module

This builds the wasm module with zig.

Build command:

zig build-lib main.zig -target wasm32-freestanding -dynamic -femit-bin=../module.wasm -rdynamic -OReleaseSmall
