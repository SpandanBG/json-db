# Introduction
`json-db` is a web server that provides a way to query the json files stored in the `data` folder using the `jq` cmd.

This will work as a pseudo database with any json file as long as the query passed is correct.

## Routes:
1. GET - `/:filename/get` :: Returns the entire JSON data in the file `filename`.
2. POST - `/:filename/query` :: Performs the `jq` query in the following cmd
```shell
jq $query ./data/$filename.json
```
The request post body:
```json
{ "query": "jq-query"}
```

## Prerequisits
- Install `jq` from [here](https://jqlang.github.io/jq/)
