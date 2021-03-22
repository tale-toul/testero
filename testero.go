package main

import (
	"fmt"
	"github.com/tale-toul/testero/partmem"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"
)

//Data structure with actual part definitions and data
var partScheme partmem.PartCollection

//Lock buffered, to make sure there is no concurrency problems
var lock chan int64

//Environment variable to set the limit for request to add data into memory.  In bytes
var HIGHMEMLIM uint64

func main() {
	lock = make(chan int64, 1)
	lock <- 0
	partScheme = partmem.NewpC()

	//Attempt to get the value for memory limit from HIGHMEMLIM environment variable
	var herr error
	hmeml := os.Getenv("HIGHMEMLIM")
	if hmeml != "" {
		HIGHMEMLIM, herr = strconv.ParseUint(hmeml, 10, 64)
		if herr != nil {
			log.Printf("Error: Cannot convert HIGHMEMLIM environment var. into number. HIGHMEMLIM=%s.  Default value will be used", hmeml)
		}
	}

	http.HandleFunc("/api/mem/add", addMem)
	http.HandleFunc("/api/mem/getdef", getDefMem)
	http.HandleFunc("/api/mem/getact", getActMem)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

//Free the concurrency lock. It's a function so it can be deferred
func freeLock(value *int64) {
	lock <- *value
}

//Attemp to get hold of the lock
func getLock() (int64, bool) {
	select {
	case r := <-lock:
		return r, true
	default:
		//Lock not available return
		return 0, false
	}

}

//Compute and create the parts for the ammount of memory requested
func addMem(writer http.ResponseWriter, request *http.Request) {
	tstamp := time.Now().UnixNano() //Request timestamp
	lval, islav := getLock()
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(&lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock is available and no pending requests (0)
		defer freeLock(&tstamp) //Make sure the lock is released even if errors occur
		bsm := request.URL.Query().Get("size")
		var sm uint64 //Requested size in bytes
		var err error
		if bsm != "" {
			sm, err = strconv.ParseUint(bsm, 10, 64)
			if err != nil {
				fmt.Fprintf(writer, "Could not get size: %s\n", err.Error())
				tstamp = 0
				return
			}
		} else { //No size specified
			fmt.Fprintf(writer, "No data size specified\n")
			tstamp = 0
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
			fmt.Fprintf(writer, "Could not compute mem parts: %s\n", err.Error())
			tstamp = 0
			return
		}
		//Create the actual parts
		go partmem.CreateParts(&partScheme, tstamp, lock)
		fmt.Fprintf(writer, "Memory data request sent for %d bytes, with id#: %d, check /api/mem/getact\n",sm,tstamp)
	}
}

//Request the definition of parts
func getDefMem(writer http.ResponseWriter, request *http.Request) {
	lval, islav := getLock()
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(&lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock obtained and no pending request
		var unlock int64 = 0
		defer freeLock(&unlock) //Make sure the lock is released even if error occur
		mensj := partmem.GetDefParts(&partScheme)
		fmt.Fprintf(writer, mensj)
	}
}

//Request the actual definition of parts created
func getActMem(writer http.ResponseWriter, request *http.Request) {
	lval, islav := getLock()
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(&lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock obtained and no pending request
		var unlock int64 = 0
		defer freeLock(&unlock) //Make sure the lock is released even if error occur
		mensj := partScheme.GetActParts()
		fmt.Fprintf(writer, mensj)
	}
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
