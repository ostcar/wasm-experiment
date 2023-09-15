
async function load_wasm(wasm_file, database_file) {
    const database = await fetch(database_file, {
        method: 'GET',
        headers: {
            'Accept': 'application/json',
        },
        cache: 'no-cache',
    })
    .then(response => response.json());

    const importObj = {
        app: {
            getData: (key_pointer) => {
                const key = read_string(key_pointer);
                const value = JSON.stringify(database[key]);
                const ptr = send_string(value);
                return ptr;
            },
        },
    };

    const wasm = await WebAssembly.instantiateStreaming(fetch(wasm_file), importObj);

    const memory = wasm.instance.exports.memory;
    const malloc = wasm.instance.exports.malloc;
    const free = wasm.instance.exports.free;
    const wasm_has_perm = wasm.instance.exports.hasPerm;

    const has_perm = (user_id, meeting_id, perm) => {
        try {
            const perm_pointer = send_string(perm);
            // TODO: Free memory
            
            const result = wasm_has_perm(user_id, meeting_id, perm_pointer);

            let attr = undefined;
            if (result == 1) {
                attr = "has"
            } else {
                attr = "has not"
            }

            return `user ${user_id} ${attr} ${perm}`
        } catch (e) {
            console.error(e);
        }
    };

    const send_string = (str) => {
        const message = new TextEncoder().encode(str+ "\x00")
        const pointer = malloc(message.length);
        const slice = new Uint8Array(memory.buffer, pointer, message.length);
        slice.set(message);
        return pointer;
    };

    const read_string = (pointer) => {
        const sliceView = new Uint32Array(memory.buffer, pointer, 2);
        const slicePointer = sliceView[0];
        const sliceLength = sliceView[1];

        const strView = new Uint8Array(memory.buffer, slicePointer, sliceLength);
        const decoded = new TextDecoder().decode(strView);
        return decoded;
    };

    return has_perm;
}


if (typeof module !== "undefined") {
    module.exports = {
        load_wasm,
    };
}
