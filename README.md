# 🏋️kworx

Multi-threaded kubectl (kubectl with workers)

## Features

* Configurable parallelism
* Color-coded output for easy delineation of output per value

## Warning

Due to the parallelism of this tool, it is not recommended for mutating calls.

## Using

```sh
$ kworx run -w 25 --values-file ns.txt -- bash -c 'kubectl -n $KWORX_VALUE get po 2>&1' > res.txt
```
