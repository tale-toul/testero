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
Memory data request sent for 256000 bytes, with id#: 1616356861141864285, check /api/mem/getact
```
The server returns the message inmediately, while the actual allocation of memory is done in the background, [see the section about concurrency](#concurrency). 

* __/api/mem/getdef__ (no parameters). Sending a GET request to this endpoint returns a description of how the data structure that was computed for the last __add__ request.

```shell
$ curl http://localhost:8080/api/mem/getdef
Boxes of size: 256000, count: 1, total size: 256000
Boxes of size: 1048576, count: 0, total size: 0
Boxes of size: 4194304, count: 0, total size: 0
Boxes of size: 16777216, count: 0, total size: 0
Boxes of size: 67108864, count: 0, total size: 0
Total size reserved: 256000 bytes.
```
* __/api/mem/getact__ (no parameters). Sending a GET request to this endpoints returns the actual number of memory chunks allocated for each of the predefined sizes.
```shell
$ curl http://localhost:8080/api/mem/getact
Last request ID: 1616356861141864285
Parts of size: 256000, Count: 1
Parts of size: 1048576, Count: 0
Parts of size: 4194304, Count: 0
Parts of size: 16777216, Count: 0
Parts of size: 67108864, Count: 0
Total size: 256000 bytes
```
## CONCURRENCY
Every API endpoint served by the Go web server runs as a _goroutine_, which is a lightweight thread managed by the Go runtime.  This means that the web server can process many request in parallel, accepting and processing new requests as they come, even if the previous requets have not been completed.  While this behavior makes the web server more responsive, and improves its performance, special care must be taken to make sure that the results delivered are consistent, specially when different _goroutines_ access and modify the same data structures.

In the case of the memory management in __testero__ a common data structure of type __partmem.PartCollection__, that contains the information about the memory allocation is used by all endpoints:
* The endpoints __/api/mem/getdef__ and __/api/mem/getact__ are used to query the data structure without modifying it.
* The endpoint __/api/mem/add__ is used to modify the data structure.

These endpoints can be accessed by any number of clients at the same time and the web server will accept and try to process those requests in parallel.  This situation leads to the common data structure to be accessed by different goroutines at the same time, and if more than one modifies the data at the same time, the results will be unpredictable and inconsistent.  

Care must be taken to ensure that only one modification operation can be executed on the common data structure, and only when that operation is completed, can the next one proceed.

Executing concurrent reading operations on the common data structure is usually safe in terms of data integrety, however the results obtained can also be inconsistent if the reading operation is executed while a writing operation is happening.

For the above reasons the access to the common data structure is limited to a single goroutine, associated with the web server endpoints, at a time, effectively serializing the requests and reducing performance in favor of consistency.  To achive this goal the data structure is protected by a locking mechanism based of a single channel called __lock__. 

The channel is defined as a global variable of type __int64__ and constructed as a _buffered_ channel with a single element, initially loaded with the value 0:

```
var lock chan int64
...
func main() {
lock = make(chan int64, 1)
lock <- 0
```
Being a global variable gives every function in the package access to it.
The numeric value sent to the channel carries a message for the function reading it.  If the value is 0, the lock is available to be used by the function holding it, if the value is any other number, an update operation is being processed and other requests should not run.
The lock can be initialized to 0 because it is defined as buffered.

### Locking logic
Every goroutine serving an API endpoints tries to get the lock at the beginning by calling the `getLock()` function.  This function uses a `select` statement to avoid blocking the execution of the gorutine while waiting for a value in the lock, and returns two values: a number and a boolean.  

* If the boolean is false, the lock could not be acquired which means that another gorutine is running, so a _server busy_ message is returned to the client. In this case the value is not important.
* If the boolean is true, the lock could be acquired and the value indicates the message described before. If the value is 0 the goroutine can procedd, otherwise a memory update is being processed and this request cannot run, the lock is returned with the same value, the executio sleeps for 1 second and a server busy message is returned to the client.

This two layer locking mechanism requiring the goroutine to get the lock and having a specific value may look inefficient, however this is because of the time that a large memory allocation may take.  When such a request is received, for example __/api/mem/add?size=1777333555__, the addMem() goroutine is called, gets the lock and eventually calls `partmem.CreateParts()` which is responsible for adding or removing data to memory to reach the size specified in the request, in the example around 1.6Gi.  The time required to allocated such a large ammount of memory may take more time than the client or, in the case of Openshift, the routers timeout, causing the cliente to get an error message:

```shell

```
