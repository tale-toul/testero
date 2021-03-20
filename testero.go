package main

import (
	"fmt"
	"github.com/tale-toul/testero/partmem"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"
)

//Data structure with actual part definitions and data
var partScheme partmem.PartCollection
//Lock buffered, to make sure there is no concurrency problems
var lock chan struct{}

//Environment variable to set the limit for request to add data into memory.  In bytes
var HIGHMEMLIM uint64

func main() {
	lock = make(chan struct{},1)
	lock <- struct{}{}
	partScheme = partmem.NewpC()

	//Attempt to get the value for memory limit from HIGHMEMLIM environment variable
	var herr error
	hmeml := os.Getenv("HIGHMEMLIM")
	if hmeml != "" {
		HIGHMEMLIM, herr = strconv.ParseUint(hmeml, 10, 64)
		if herr != nil {
			log.Printf("Error: Cannot convert HIGHMEMLIM environment var. into number. HIGHMEMLIM=%s.  Default value will be used",hmeml)
		}
	}

	http.HandleFunc("/api/mem/add", addMem)
	http.HandleFunc("/api/mem/getdef", getDefMem)
	http.HandleFunc("/api/mem/getact", getActMem)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

//Free the concurrency lock. It's a function so it can be deferred
func freeLock() {
	lock <- struct{}{}
}

//Attemp to get hold of the lock
func getLock() (string, bool) {
	select {
		case <-lock:
			return "",true
		default:
			//Lock not available return
			return	fmt.Sprintf("Server busy, try again later\n"), false
	}

}
//Compute and create the parts for the ammount of memory requested
func addMem(writer http.ResponseWriter, request *http.Request) {
	lmsg,islav := getLock()
	if !islav { //Lock not available
		fmt.Fprintf(writer,"%s",lmsg)
		return
	}
	defer freeLock() //Make sure the lock is released even if errors occur
	bsm := request.URL.Query().Get("size")
	var sm uint64 //Requested size in bytes
	var err error
	if bsm != "" {
		sm, err = strconv.ParseUint(bsm, 10, 64)
		if err != nil {
			fmt.Fprintf(writer,"Could not get size: %s\n",err.Error())
			return
		}
	} else {  //No size specified
		fmt.Fprintf(writer, "No data size specified\n")
		return
	}

	//Set the high limit for the total size to request
	if HIGHMEMLIM == 0 { //Not defined
		HIGHMEMLIM = freeRam()
	}
	
	//Compute the number of parts of each size to accomodate the total size.
	//The result is stored in partScheme
	err = partmem.DefineParts(sm, HIGHMEMLIM, &partScheme)
	if err != nil {
		fmt.Fprintf(writer,"Could not compute mem parts: %s\n",err.Error())
		return
	}

	//Create the actual parts
	fmt.Fprintf(writer, partmem.CreateParts(&partScheme))

	fmt.Fprintf(writer, "Data added\n")
}

//Request the definition of parts
func getDefMem(writer http.ResponseWriter, request *http.Request) {
	lmsg,islav := getLock()
	if !islav { //Lock not available
		fmt.Fprintf(writer,"%s",lmsg)
		return
	}
	defer freeLock() //Make sure the lock is released even if error occur
	mensj := partmem.GetDefParts(&partScheme)
	fmt.Fprintf(writer, mensj)
}

//Request the actual definition of parts created
func getActMem(writer http.ResponseWriter, request *http.Request) {
	lmsg,islav := getLock()
	if !islav { //Lock not available
		fmt.Fprintf(writer,"%s",lmsg)
		return
	}
	defer freeLock() //Make sure the lock is released even if error occur
	mensj := partScheme.GetActParts()
	fmt.Fprintf(writer, mensj)
}

// Returns the number of bytes of free RAM memory available in the system
func freeRam() uint64 {
	var localInfo syscall.Sysinfo_t
	err := syscall.Sysinfo(&localInfo)
	if err != nil {
		log.Fatal(err)
	}
	return localInfo.Freeram
}
