# TESTERO
Testero is a simple webserver application written in go to test resource consumption in a running system.  

The application is intended to run as a containerized service in a kubernetes cluster, but it can also be run directly from the terminal in a host.

The user running the application does not require any special privileges.

## RUNNING TESTERO
Get the code and use the _go_ compiler to execute the testero.go main file:

```shell
$ go run testero.go
```
While the program is running, the terminal from which it was started is blocked.  To stop the program simply press __CTRL-C__ or kill the process from another terminal.

The application does not support any starting parameters at the moment.

The web service listens on localhost, port 8080 (127.0.0.1:8080)

## API ENDPOINTS
The web service publishes the following API endpoints

* __/api/mem/add__ (parameter __size=number of bytes__).  Sending a GET request to this endpoint will result in the allocation of the specified number of bytes in memory.
```shell
$ curl http://localhost:8080/api/mem/add?size=256000
Data added
```
* __/api/mem/getdef__ (no parameters). Sending a GET request to this endpoint returns a description of how the data structure that was computed for the last __add__ request.

```shell
$ curl http://localhost:8080/api/mem/getdef
Boxes of size: 256000, count: 1, total size: 256000
Boxes of size: 1048576, count: 0, total size: 0
Boxes of size: 4194304, count: 0, total size: 0
Boxes of size: 16777216, count: 0, total size: 0
Boxes of size: 67108864, count: 0, total size: 0
Total size reserved: 256000
```
* __/api/mem/getact__ (no parameters). Sending a GET request to this endpoints returns the actual number of memory chunks allocated for each of the predefined sizes.
```shell
$ curl http://localhost:8080/api/mem/getact
Parts of size: 256000, Count: 1
Parts of size: 1048576, Count: 0
Parts of size: 4194304, Count: 0
Parts of size: 16777216, Count: 0
Parts of size: 67108864, Count: 0
```
