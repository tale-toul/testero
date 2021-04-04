# TESTERO
Testero is an application intended to test resource consumption in a kubernetes cluster.  It accepts requests at specific API endpoints to allocate memory, file storage and CPU usage.

The user running the application does not require any special privileges.

Resources can be released by requesting a zero ammount for memory and file storage, and by calling the _stop_ endpoint in the case of CPU usage.  Additionally when the applications is terminated all resources are released, in particular any files that may have been created are deleted.

As a safety meassure to prevent resource starvation in the system, each of the resource groups: memory, file storage and CPU usage, have a default limit in the ammount that can be requested by the user.  These limits are defined at the beginning of the program and not reevaluated again.

It is not recommended to run this application in a production environment, due to its own nature as a resource consumer and despite the default limits that it imposes on in resource, other applications running on the system can be affected by the reduction in available resources for their normal operation.

## RUNNING TESTERO
The _testero_ application can be used as a containerized service in a kubernetes cluster, or as an independent application.

### RUNNING AS AN INDEPENDENT APPLICATION
_testero_ has been tested on linux systems, it is not guarrantied to work on Mac or Windows systems.

To run _testero_ on a linux host the [golang](https://golang.org) toolset is required to compile the source code into a binary application.  Once the bynari is generated, the golang toolset is not required anymore.

Get the code from the git [repositoryr](https://github.com/tale-toul/testero) and use the _go_ tool to execute the testero.go main file.  

```shell
$ git clone https://github.com/tale-toul/testero
$ cd testero
$ go run testero.go
...
2021/04/04 12:14:04 Starting web server on: 0.0.0.0:8080
```
This can all be done with a single command:

```shell
$ go get -u -v github.com/tale-toul/testero
github.com/tale-toul/testero (download)
```
The _testero_ binary file can be found at $GOPATH/bin/testero

```shell
$ ls $(go env GOPATH)/bin
```
Run the command with a command like:
```shell
$ $(go env GOPATH)/bin/testero
...
2021/04/04 12:17:00 Starting web server on: 0.0.0.0:8080
```
To stop the program simply press __CTRL-C__ or kill the process from another terminal.  It is recommended to terminate the application with a SIGTERM or SIGINT signal to make sure that any files created during program execution are deleted.

The application does not support any starting parameters at the moment.

The web server listens on all local IPs and port 8080 (0.0.0.0:8080) by default.

### RUNNING AS A CONTAINER
To run _testero_ as a container, create the binay file as explained in [the previous section](#running_as_an_independent_application), then create a container image that runs that binary, an example Dockerfile is included in the project code:

* Copy the Dockerfile and the _testero_ binary to a common directory
```shell
$ mkdir testero_app
$ cp $(go env GOPATH)/bin/testero testero_app
$ cp $(go env GOPATH)/src/github.com/tale-toul/testero/Dockerfile testero_app
```
* Build the container, __podman__ is used in the following examples but __docker__ is also a valid alternative using the same options and parameters:

```shell
$ cd testero_app/
$ sudo podman build -t testero .
```
* A container can be instantiated directly from the image:
```shell
$ sudo podman run --name testero -d -p 8080:8080 testero
```
* The application should be available at `http://localhost:8080`

```shell
$ curl http://localhost:8080/api/mem/getact
Last request ID: 0
Parts of size: 262144, Count: 0
Parts of size: 1048576, Count: 0
Parts of size: 4194304, Count: 0
Parts of size: 16777216, Count: 0
Parts of size: 67108864, Count: 0
Total size: 0 bytes.
```
To run the container in a kubernetes cluster, the image must be pushed to a registry before it can be deployed into the cluster:

* Tag the image with the name of the registry and the user and project where it will be stored:

```shell
$ sudo podman tag testero quay.io/milponta/testero
``` 
* Push the image into the registry.  If the registry requires authentication, log in first:
```shell
$ sudo podman login -u milponta quay.io
$ sudo podman push quay.io/tale_toul/testero
Getting image source signatures
Copying blob da696ed0d687 done
...
Writing manifest to image destination
Storing signatures
```
* To deploy the application in an Openshift cluster the following example uses the __oc new-app__ command.  A [CodeReady Containers](https://developers.redhat.com/products/codeready-containers/overview) cluster is used, and the image repository is expected to be publicly available and don't require authentication to pull the image.

```shell
$ oc login -u developer -p developer https://api.crc.testing:6443
$ oc new-project testero
$ oc new-app --name testero --docker-image quay.io/milponta/testero
$ oc get pods
```
* Create a route to access the application:

```shell
$ oc expose svc testero
$ oc get routes
```
* The application should available at `http://testero-testero.apps-crc.testing`:

```shell
$ curl http://testero-testero.apps-crc.testing/api/mem/getact
Last request ID: 0
Parts of size: 262144, Count: 0
Parts of size: 1048576, Count: 0
Parts of size: 4194304, Count: 0
Parts of size: 16777216, Count: 0
Parts of size: 67108864, Count: 0
Total size: 0 bytes.
```

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

This two layer locking mechanism requiring the goroutine to get the lock and having a specific value may look inefficient, however this is designed this way because of the time that a large memory allocation may take.  When such a request is received, for example __/api/mem/add?size=1777333555__, the addMem() goroutine is called, gets the lock and eventually calls `partmem.CreateParts()` which is responsible for the actual adding or removing of data to reach the size specified in the request, around 1.6Gi in the example.  The time required to allocated such a large ammount of memory may be longer than the client or the Openshift routers are willing to wait, causing a timeout message: 

```shell
$ curl http://testero.apps.ocp4.example.com/api/mem/add?size=1777333555
<html><body><h1>504 Gateway Time-out</h1>
The server didn't respond in time.
</body></html>
```
This however does not affect the goroutine, which will carry on with its task until completion, as a followup query will show:
```
$ curl -s http://testero.apps.ocp4.example.com:8080/api/mem/getact
Last request ID: 1616421155253155251
Parts of size: 256000, Count: 20
Parts of size: 1048576, Count: 20
Parts of size: 4194304, Count: 20
Parts of size: 16777216, Count: 20
Parts of size: 67108864, Count: 20
Total size: 1787699200 bytes.
```
To avoid this situation the call to `partmem.CreateParts()` is executed as another goroutine, as soon as the call is made, a message is sent to the client and the connection is closed, while the memory allocation continues on the server side until completion.
```
go partmem.CreateParts(&partScheme, tstamp, lock)
fmt.Fprintf(writer, "Memory data request sent for %d bytes, with id#: %d, check /api/mem/getact\n",sm,tstamp)
```
With this design the timeout issue is avoided, but the function associated with the api endpoint __/api/mem/add__ releases the lock when it finishes making it available for another goroutine to claim it.  To guarrantee that the newly spawned `partmem.CreateParts()` function gets the lock before any other goroutine the following algorithm is used:

1. If the `addMem()` goroutine launches `partmem.CreateParts()`, the lock is released by assigning a timestamp value in nanoseconds to it, that same timestamp value is sent as a parameter to `partmem.CreateParts()`
1. Any goroutine that gets the lock, will read its value and seeing its a non zero value, will realese the lock again putting the same value back.
1. When the `partmem.CreateParts()` function starts it waits for the lock to be available. If the lock can be read and it contains the same value that was passed as a parameter, the function can proceed; if the values don't match the lock is returned and a log message is sent because this probably should not happen.  If 5 seconds pass and the lock could not be obtained a log message is recorded and the function returns.
1. If the function got the correct lock, the lock will be released with a value of 0 so another function can take it.
```
select {
  case <- time.After(5 * time.Second):
    log.Printf("partmem.CreateParts(): timeout waiting for lock\n")
    return
  case chts := <- lock:
    if chts == ts { //Got the lock and it matches the timestamp received
      //Proceed
      ptS.lid = ts
      defer func(){
        lock <- 0 //Release lock
      }()
      lt = time.Now() //Start counting how long does the parts creation take
      log.Printf("partmem.CreateParts(): lock obtained, timestamps match: %d\n",ts)
    } else {
      log.Printf("partmem.CreateParts(): lock obtained, but timestamps missmatch: %d - %d\n", ts,chts)
      lock <- chts
      return
    }
  }
```
An important point to make sure that the lock is released even if a goroutine ends in failure is that a function is used to release the lock, and that function is deferred as soon as the lock is obtained.
