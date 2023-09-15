import sys
from collections import defaultdict
import json

from wasmer import Store, Module, Instance, Function, Memory

database = json.load(open('db.json')) 

def init_wasm():
    global instance

    store = Store()
    module = Module(store, open('module.wasm', 'rb').read())

    import_object = defaultdict(dict)
    import_object["app"]["getData"] = Function(store, get_data)
    
    instance = Instance(module, import_object)   


def has_perm(userID, meetingID, perm):
    init_wasm()

    ptr = send_string(perm)
    got = instance.exports.hasPerm(userID, meetingID, ptr)

    return got == 1


def get_data(key_pointer: int) -> int:
    key = read_string(key_pointer)
    try:
        value = json.dumps(database[key])
    except KeyError:
        value = "null"
    return send_string(value) 


def send_string(str):
    length = len(str)+1

    ptr = instance.exports.malloc(length)
    # TODO: Free

    view = instance.exports.memory.uint8_view(offset=ptr)
    for i in range(len(str)):
        view[i] = ord(str[i])
    view[len(str)] = 0

    return ptr


def read_string(ptr):
    view = instance.exports.memory.uint32_view(offset=ptr//4)
    p = view[0]
    l = view[1]

    view = instance.exports.memory.uint8_view(offset=p)
    return ''.join(chr(c) for c in view[:l])



if __name__ == '__main__':
    if len(sys.argv) <4:
        print(f"Run: {sys.argv[0]} USER_ID MEETING_ID PERM")
        sys.exit(1)
    
    attr = "has" if has_perm(int(sys.argv[1]), int(sys.argv[2]), sys.argv[3]) else "has not"
    
    print(f"user {sys.argv[1]} {attr} {sys.argv[3]}")
