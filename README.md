# Wasm experiment

This is a proof of concept to call a wasm-module from go, python and
javascript.

The wasm-module implements the function `hasPerm` that expects a user_id, a
meeting_id and a permission string. It returns, if the user has the permission
in that meeting.

The data is defined in db.json. This file is read from the host (go, python or
javascript) and provided to the wasm-module. In reality, the db.json file would
not exist, but the host would use the postgres-database or in case of the
client, call the server.


## Go

```
go run main.go 5 1 agenda.can_see
```


## Python

You have to use a python version lower then 3.11 (for example 3.10). See: https://github.com/wasmerio/wasmer-python/issues/696


```
python -m venv .venv
. .venv/bin/activate
pip install wasmer wasmer_compiler_cranelift
python has_perm.py 5 1 agenda.can_see
```
