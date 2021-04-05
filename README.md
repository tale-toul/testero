# TESTERO
Testero is an application intended to test resource consumption in a kubernetes cluster.  It accepts requests at specific API endpoints to allocate memory, file storage and CPU usage.

The user running the application does not require any special privileges.

Resources can be released by requesting a zero ammount for memory and file storage, and by calling the _stop_ endpoint in the case of CPU usage.  Additionally when the applications is terminated all resources are released, in particular any files that may have been created are deleted.

As a safety meassure to prevent resource starvation in the system, each of the resource groups: memory, file storage and CPU usage, have a default limit for the ammount that the user can request by.  These limits are defined at the beginning of the program and not recalculated again.

It is not recommended to run this application in a production environment, due to its own nature as a resource consumer and despite the default limits in place, other applications running on the system can be affected by the reduction in available resources for their normal operation.  Another point to consider before using the application is that it does not require authentication, so anyone with network access to the endpoints can send requests and consume resources.

## RUNNING TESTERO
The _testero_ application can be used as a containerized service in a kubernetes cluster, or as an independent application.  In the spirit of cloud native development, the application does not support starting parameters, however some aspects of the execution can be adapted through the use of [environment variables](#configuration-with-environment-variables).

The web server listens on all local IPs and port 8080 (0.0.0.0:8080).

### RUNNING AS AN INDEPENDENT APPLICATION
_testero_ has been tested on linux systems, it is not guarrantied to work on Mac or Windows systems.

To run _testero_ on a linux host the [golang](https://golang.org) toolset is required to compile the source code into a binary application.  Once the bynari is generated, the golang toolset is not required anymore.

Get the code from the git [repository](https://github.com/tale-toul/testero) and use the _go_ tool to execute the testero.go main file.  

```
$ git clone https://github.com/tale-toul/testero
$ cd testero
$ go run testero.go
...
2021/04/04 12:14:04 Starting web server on: 0.0.0.0:8080
```
This can all be done with a single command:

```
$ go get -u -v github.com/tale-toul/testero
github.com/tale-toul/testero (download)
```
The resulting _testero_ binary file can be found at $GOPATH/bin/testero

```
$ ls $(go env GOPATH)/bin
```
Run the application with a command like:
```
$ $(go env GOPATH)/bin/testero
...
2021/04/04 12:17:00 Starting web server on: 0.0.0.0:8080
```
To stop the application simply press __CTRL-C__ or kill the process from another terminal.  It is recommended to terminate the application using a SIGTERM or SIGINT signal to make sure that any files created during program execution are properly cleaned up.


### RUNNING AS A CONTAINER
To run _testero_ as a container, create the binay file as explained in [the previous section](#running-as-an-independent-application), then create a container image that runs that binary, an example Dockerfile is included in the project code:

* Copy the Dockerfile and the _testero_ binary to a common directory
```
$ mkdir testero_app
$ cp $(go env GOPATH)/bin/testero testero_app
$ cp $(go env GOPATH)/src/github.com/tale-toul/testero/Dockerfile testero_app
```
* Build the container, __podman__ is used in the following examples but __docker__ is also a valid alternative using the same options and parameters:

```
$ cd testero_app/
$ sudo podman build -t testero .
```
* A container can be instantiated directly from the image:
```
$ sudo podman run --name testero -d -p 8080:8080 testero
```
* The application should be available at `http://localhost:8080`

```
$ curl http://localhost:8080/api/mem/getact
Last request ID: 0
Parts of size: 262144, Count: 0
Parts of size: 1048576, Count: 0
Parts of size: 4194304, Count: 0
Parts of size: 16777216, Count: 0
Parts of size: 67108864, Count: 0
Total size: 0 bytes.
```
### RUNNING IN AN OPENSHIFT CLUSTER
To run the container in a kubernetes cluster, the image must be pushed to a registry before it can be deployed into the cluster:

* Tag the image with the name of the registry and the user and project where it will be stored:

```
$ sudo podman tag testero quay.io/milponta/testero
``` 
* Push the image into the registry.  If the registry requires authentication, log in first:
```
$ sudo podman login -u milponta quay.io
$ sudo podman push quay.io/tale_toul/testero
Getting image source signatures
Copying blob da696ed0d687 done
...
Writing manifest to image destination
Storing signatures
```
* To deploy the application in an Openshift cluster the following example uses the __oc new-app__ command.  A [CodeReady Containers](https://developers.redhat.com/products/codeready-containers/overview) cluster is used, and the image repository is expected to be publicly available and don't require authentication to pull the image.

```
$ oc login -u developer -p developer https://api.crc.testing:6443
$ oc new-project testero
$ oc new-app --name testero --docker-image quay.io/milponta/testero
$ oc get pods
```
* Create a route to access the application:

```
$ oc expose svc testero
$ oc get routes
```
* The application should available at `http://testero-testero.apps-crc.testing`:

```
$ curl http://testero-testero.apps-crc.testing/api/mem/getact
Last request ID: 0
Parts of size: 262144, Count: 0
Parts of size: 1048576, Count: 0
Parts of size: 4194304, Count: 0
Parts of size: 16777216, Count: 0
Parts of size: 67108864, Count: 0
Total size: 0 bytes.
```

## CONFIGURATION WITH ENVIRONMENT VARIABLES

A well behaved cloud native application should [Store config in the environment](https://12factor.net/)

_testero_ checks for the following environment variables to modify its configuration.  None of these variables is strictly require, if not defined default values will be used:

* __HIGHMEMLIM__.- Used to set the limit of total memory the application can allocate.  Expects a number representing the ammount of memory in bytes, for example to set limit to 2GB use `HIGHMEMLIM=2147483648`.  Its default value is set to the ammount of free momemory in the system at application start up.

* __HIGHFILELIM__.- Used to set the limit of total file storage the application can create. Expects a number representing the ammount of storage in bytes, for example to set limit to 10GB use `HIGHFILELIM=10737418240`.  Its default value is set to the ammount of available disk space in the device associated with the directory defined by the __DATADIR__ environment variable, at application start up.

* __DATADIR__.- Used to specify the root directory where files will be created, this directory must already exist in the system, for example `DATADIR=/tmp`. Its default value is the application working directory.

* __NUMTOFACTOR__.- Used to specify the number to factorize, which is used by the CPU load generation part of the application, and defines the maximum ammount of time the application will load the CPU in the system.  Its default values is the number prime number __493440589722494743501__ which roughly requires between 15 to 25 minutes to factorize depending on the system.  To load the CPU for a longer or shorter time a different, possibly prime,  number can be used, for example `NUMTOFACTOR=49344058972249501099`.

The following example runs the application as a standalone program, defining some environment variables:
```
$ HIGHMEMLIM=2147483648 HIGHFILELIM=10737418240 DATADIR=/tmp NUMTOFACTOR=49344058972249501099 ./testero 
...
2021/04/04 19:05:04 Starting web server on: 0.0.0.0:8080
```
The following example uses the same environment variables for a standalone container execution:

```
$ sudo podman run -d --rm -p 8080:8080 --name testero -e HIGHMEMLIM=2147483648 -e HIGHFILELIM=10737418240 -e DATADIR=/tmp -e NUMTOFACTOR=49344058972249501099 testero
```
The following example uses the same environment variables to deploy the application in an Openshift cluster:
```
$ oc new-app --name testero -e HIGHMEMLIM=2147483648 -e HIGHFILELIM=10737418240 -e DATADIR=/tmp -e NUMTOFACTOR=49344058972249501099 --docker-image quay.io/tale_toul/testero
```

## API ENDPOINTS
The application publishes the following API endpoints:

The endpoints in every group (memory, disk, cpu) share a common locking mechanism, so when a request is accepted a message is returned inmediately, but the actual request will take some time to complete.  If a request arrives while another one is being served, it is discarded and a _server busy_ message returned to the client, [see the section about concurrency](#concurrency).  Unsuccessful request will have an intetionally added 1 second dealy in the response to avoid overloading the service.

Every group has its own independent locking mechanism so one request of each group can be served at the same time.

### MEMORY ENDPOINTS
* __/api/mem/set__ (parameter __size=number of bytes__). Sending an HTTP GET request to this endpoint results in the allocation of the specified number of bytes in memory.  If the size requested is more than the currently allocated ammount, or this is the first request, the application will create more data in memory until it reaches the ammount requested.  However if the size requested is less than the currently allocated ammount, the application will release the excess data in memory until it reaches the requested ammount.  To release all the memory use __size=0__

The actual ammount of memory allocated by the application will not be exactly the same ammount requested, this is because the memory is allocated in chunks of predefined sizes.
```
$ curl http://localhost:8080/api/mem/set?size=256000
Memory data request sent for 256000 bytes, with id#: 1616356861141864285, check /api/mem/getact
```
If the memory size requested goes over the limit, an error message is returned and nothing is done:

```
$ curl http://localhost:8080/api/mem/set?size=111000333555
Could not compute memory parts: Size requested is over the limit: requested 111000333555 bytes, limit: 447705088 bytes.
```
* __/api/mem/getdef__ (no parameters). Sending an HTTP GET request to this endpoint returns a description of the memory data structure that was computed for the last __set__ request.  If no successful __set__ request has been sent before, the values returned are set to zero.  This information represents the values computed not the actual memory reserved, although both should match.

```
$ curl http://localhost:8080/api/mem/getdef
Boxes of size: 256000, count: 1, total size: 256000
Boxes of size: 1048576, count: 0, total size: 0
Boxes of size: 4194304, count: 0, total size: 0
Boxes of size: 16777216, count: 0, total size: 0
Boxes of size: 67108864, count: 0, total size: 0
Total size reserved: 256000 bytes.
```
* __/api/mem/getact__ (no parameters). Sending an HTTP GET request to this endpoint returns the actual number of memory parts for each of the predefined sizes and the total size of memory allocated.
```
$ curl http://localhost:8080/api/mem/getact
Last request ID: 1616356861141864285
Parts of size: 256000, Count: 1
Parts of size: 1048576, Count: 0
Parts of size: 4194304, Count: 0
Parts of size: 16777216, Count: 0
Parts of size: 67108864, Count: 0
Total size: 256000 bytes
```
### DISK ENDPOINTS
Disk API endpoints work much like the memory endpoints:
* __/api/disk/set__ (parameter __size=number of bytes__). Sending an HTTP GET request to this endpoint results in the creation or deletion of files to reach the specified ammount of bytes, depending on wheter the requested size is more or less than the previous one.  To delete all files use __size=0__
```
$ curl http://localhost:8080/api/disk/set?size=2333111
File data request sent for 2333111 bytes, with id#: 1617641357639017521, check /api/disk/getact
```
If the file size requested goes over the limit, an error message is returned and nothing is done:
```
$ curl http://localhost:8080/api/disk/set?size=2333111445322376544
Could not compute file distribution: Size requested is over the limit: requested 2333111445322376544 bytes, limit: 50554786816 bytes.
```
* __/api/disk/getdef__ (no parameters). Sending an HTTP GET request to this endpoint returns a description of the files distribution data structure that was computed for the last __set__ request.  If no successful __set__ request has been sent before, the values returned are set to zero.  This information represents the values computed not the actual memory reserved, although both should match.
```
$ curl http://localhost:8080/api/disk/getdef
Files of size: 524288, count: 25, total size: 13107200
Files of size: 2097152, count: 25, total size: 52428800
Files of size: 8388608, count: 27, total size: 226492416
Files of size: 33554432, count: 25, total size: 838860800
Files of size: 134217728, count: 9, total size: 1207959552
Total size reserved: 2338848768 bytes.
```
* __/api/disk/getact__ (no parameters). Sending an HTTP GET request to this endpoint returns the actual number of files for each of the predefined sizes and the total size that they take.
```
$ curl http://localhost:8080/api/disk/getact
Last request ID: 1617641827431379890
Files of size: 524288, Count: 25
Files of size: 2097152, Count: 25
Files of size: 8388608, Count: 27
Files of size: 33554432, Count: 25
Files of size: 134217728, Count: 9
Total size: 2338848768 bytes.
```
### CPU ENDPOINTS
* __/api/cpu/load__ (parameter __time=number of seconds__).  Sending an HTTP GET request to this endpoint results in the execution of a process that will consume as much as it can of a single CPU in the system by looking for the factors of a big number.  The time parameters is used to set the ammount of time in senconds the process will run.  The maximum time that the CPU will be loaded depends on the number to factorize, by default it takes between 15 to 25 minutes, depending on the system.  So no matter how large the time parameter is, once the number is factorized the process will finish and the CPU load will cease.
```
$ curl http://localhost:8080/api/cpu/load?time=20
CPU load requested for 20 seconds with id: 1617644604926027157
```
* __/api/cpu/stop__ (parameter __id=current load request ID__).  Sending an HTTP GET request to this endpoint stops the process that is producing the CPU load immediately. 
```
$ curl http://localhost:8080/api/cpu/stop?id=1617644968125725512
CPU load stopped
```
If the ID value does not match the one returned when the load request was accepted, the stop request will be rejected:
```
$ curl http://localhost:8080/api/cpu/stop?id=1617644968125725512
Incorrect stop load request ID=1617644968125725512
```
If there is no load request running, nothing needs to be done:
```
$ curl http://localhost:8080/api/cpu/stop?id=1617644968125725512
No load request being processed, nothing to do
```
* __/api/cpu/getact__ (no parameters).  Sending an HTTP GET request to this endpoint returns information about the current load request being processed, if there is one.
```
$ curl http://localhost:8080/api/cpu/getact
Load request sent at: 2021-04-05 20:03:13 +0200 CEST
Load time requested: 200 seconds
Load request ends at: 2021-04-05 20:06:33 +0200 CEST
Number to factor: 493440589722494743501
```

## CONCURRENCY
Every API endpoint served by the Go web server runs as a _goroutine_, which is a lightweight thread managed by the Go runtime.  This means that the web server can process many request in parallel, accepting and processing new requests as they come, even if the previous requets have not been completed.  While this behavior makes the web server more responsive, and improves its performance, special care must be taken to make sure that the results delivered are consistent, specially when different _goroutines_ access and modify the same data structures.

In the case of the memory management in __testero__ a common data structure of type __partmem.PartCollection__, that contains the information about the memory allocation is used by all endpoints:
* The endpoints __/api/mem/getdef__ and __/api/mem/getact__ are used to query the data structure without modifying it.
* The endpoint __/api/mem/set__ is used to modify the data structure.

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

This two layer locking mechanism requiring the goroutine to get the lock and having a specific value may look inefficient, however this is designed this way because of the time that a large memory allocation may take.  When such a request is received, for example __/api/mem/set?size=1777333555__, the addMem() goroutine is called, gets the lock and eventually calls `partmem.CreateParts()` which is responsible for the actual adding or removing of data to reach the size specified in the request, around 1.6Gi in the example.  The time required to allocated such a large ammount of memory may be longer than the client or the Openshift routers are willing to wait, causing a timeout message: 

```
$ curl http://testero.apps.ocp4.example.com/api/mem/set?size=1777333555
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
With this design the timeout issue is avoided, but the function associated with the api endpoint __/api/mem/set__ releases the lock when it finishes making it available for another goroutine to claim it.  To guarrantee that the newly spawned `partmem.CreateParts()` function gets the lock before any other goroutine the following algorithm is used:

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
