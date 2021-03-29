package main

import (
	"fmt"
	"github.com/tale-toul/testero/partmem"
	"github.com/tale-toul/testero/partdisk"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"
)

//Data structure with memory part definitions and data
var partScheme partmem.PartCollection

//Data structure with files definitions
var fileScheme partdisk.FileCollection

//Lock buffered, to make sure there is no concurrency problems with memory operations
var lock chan int64
//Lock buffered, to facilitate disk operations
var filelock chan int64

//Environment variable to set the limit for request to add data into memory.  In bytes
var HIGHMEMLIM uint64
//Environment var to set the limit of storage space, in bytes.
var HIGHFILELIM uint64
//Env var specifying the directory to store files
var DATADIR string

//Attempt to get the value for memory limit from HIGHMEMLIM environment variable
func setEnvNum(evv string) uint64{
	var errnv error
	var envalue uint64
	evveml := os.Getenv(evv)
	if evveml != "" { //Env var exists
		envalue, errnv = strconv.ParseUint(evveml, 10, 64)
		if errnv != nil { //There was an error during convertion to number
			log.Printf("Error: Cannot convert %s environment var. into number. %s=%s.  Default value will be used", evv,evv,evveml)
			return 0
		} else {
			return envalue
		}
	} else {//Env var does not exist
		return 0
	}
}

func main() {
	//Initialize memory lock
	lock = make(chan int64, 1)
	lock <- 0
	//Initialize file lock
	filelock = make(chan int64, 1)
	filelock <- 0

	//Create objects for memory and files
	partScheme = partmem.NewpC()
	fileScheme.NewfC()

	//Get values from environment variables, if they exist
	HIGHMEMLIM = setEnvNum("HIGHMEMLIM")
	HIGHFILELIM = setEnvNum("HIGHFILELIM")
	DATADIR = os.Getenv("DATADIR")

	//Memory handlers
	http.HandleFunc("/api/mem/add", addMem)
	http.HandleFunc("/api/mem/getdef", getDefMem)
	http.HandleFunc("/api/mem/getact", getActMem)
	//Disk handlers
	http.HandleFunc("/api/disk/add", addFiles)
	http.HandleFunc("/api/disk/getdef", getDefFiles)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

//Free the concurrency memory lock. It's a function so it can be deferred
//value is a pointer because the function is deferred, and value can change during the execution of the calling function
func freeLock(l chan int64, value *int64) {
	l <- *value
}

//Attemps to get hold of the memory lock
func getLock(l chan int64) (int64, bool) {
	select {
	case r := <- l:
		return r, true
	default:
		//Lock not available return
		return 0, false
	}
}

//Compute and create the parts for the ammount of memory requested
func addMem(writer http.ResponseWriter, request *http.Request) {
	tstamp := time.Now().UnixNano() //Request timestamp
	lval, islav := getLock(lock)
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(lock,&lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock is available and no pending requests (0)
		defer freeLock(lock,&tstamp) //Make sure the lock is released even if errors occur
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
	lval, islav := getLock(lock)
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(lock,&lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock obtained and no pending request
		var unlock int64 = 0
		defer freeLock(lock,&unlock) //Make sure the lock is released even if error occur
		mensj := partmem.GetDefParts(&partScheme)
		fmt.Fprintf(writer, mensj)
	}
}

//Request the actual definition of parts created
func getActMem(writer http.ResponseWriter, request *http.Request) {
	lval, islav := getLock(lock)
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(lock,&lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock obtained and no pending request
		var unlock int64 = 0
		defer freeLock(lock,&unlock) //Make sure the lock is released even if error occur
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

//Get the ammount of free space in the device associated with the directory
func getfreeDisk(dir string) (uint64, error) {
	var fstats syscall.Statfs_t
	err := syscall.Statfs(dir, &fstats)
	if err != nil {
		return 0, err
	}
	return fstats.Bavail * uint64(fstats.Bsize), nil
}

//Request the definition of files
func addFiles(writer http.ResponseWriter, request *http.Request) {
	tstamp := time.Now().UnixNano() //Request timestamp
	lval, islav := getLock(filelock)
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(filelock,&lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock is available and no pending requests (0)
		defer freeLock(filelock,&tstamp) //Make sure the lock is released even if errors occur
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
		if HIGHFILELIM == 0 { //Not defined
			if DATADIR == "" {
				DATADIR = "."
			}
			HIGHFILELIM, err = getfreeDisk(DATADIR)
			if err != nil {
				fmt.Fprintf(writer, "Error computing available disk space\n")
				return
			}
		}

		//Compute the number of parts of each size to accomodate the total size.
		//The result is stored in partScheme
		err = partdisk.DefineFiles(sm, HIGHFILELIM, &fileScheme)
		if err != nil {
			fmt.Fprintf(writer, "Could not compute file distribution: %s\n", err.Error())
			tstamp = 0
			return
		}
		//Create the actual parts under here
		go partdisk.CreateFiles(&fileScheme, tstamp, filelock)
		fmt.Fprintf(writer, "File data request sent for %d bytes, with id#: %d, check /api/disk/getact\n",sm,tstamp)
	}
}

//Shows the definition of files and sizes
func getDefFiles(writer http.ResponseWriter, request *http.Request) {
	lval, islav := getLock(filelock)
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for file allocation
		defer freeLock(filelock,&lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock obtained and no pending request
		var unlock int64 = 0
		defer freeLock(filelock,&unlock) //Make sure the lock is released even if error occur
		mensj := partdisk.GetDefFiles(&fileScheme)
		fmt.Fprintf(writer, mensj)
	}
}