package main

import (
	"fmt"
	"errors"
	"github.com/tale-toul/testero/partdisk"
	"github.com/tale-toul/testero/partmem"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

//Listening IP for the web server
const ip string = "0.0.0.0"
//Listening port for web server
const port string = "8080"

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
func setEnvNum(evv string) uint64 {
	var errnv error
	var envalue uint64
	evveml := os.Getenv(evv)
	if evveml != "" { //Env var exists
		envalue, errnv = strconv.ParseUint(evveml, 10, 64)
		if errnv != nil { //There was an error during convertion to number
			log.Printf("Error: Cannot convert %s environment var. into number. %s=%s.  Default value will be used", evv, evv, evveml)
			return 0
		} else {
			return envalue
		}
	} else { //Env var does not exist
		return 0
	}
}

func main() {
	var err error

	//Channel and gorutine to handle TERM and INT Operating System signals gracefully
	sigs := make(chan os.Signal,1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go gracefulShutdown(sigs)

	//Initialize memory lock
	lock = make(chan int64, 1)
	lock <- 0
	//Initialize file lock
	filelock = make(chan int64, 1)
	filelock <- 0

	//Get values from environment variables, if they exist
	HIGHMEMLIM = setEnvNum("HIGHMEMLIM")
	HIGHFILELIM = setEnvNum("HIGHFILELIM")
	DATADIR = os.Getenv("DATADIR")
	if DATADIR == "" {
		DATADIR = "."
	}
	log.Printf("DATADIR set to: %s",DATADIR)
	//Set the high limit for the total size to request
	if HIGHMEMLIM == 0 { //Not defined
		HIGHMEMLIM = freeRam()
	}
	log.Printf("HIGHMEMLIM set to: %d bytes.",HIGHMEMLIM)
	//Set the high limit for the total file size requests
	if HIGHFILELIM == 0 {
		HIGHFILELIM, err = getfreeDisk(DATADIR)
		if err != nil {
			fmt.Printf("Error computing available disk space for directory: %s\n%s\n", DATADIR, err.Error())
			return
		}
	}
	log.Printf("HIGHFILELIM set to: %d bytes.",HIGHFILELIM)

	//Create objects for memory and files
	partScheme = partmem.NewpC()
	fileScheme.NewfC(DATADIR)

	err = createTree(fileScheme)
	if err != nil {
		log.Printf("CreateFiles(): Error creating directory tree: %s\n%s\n",fileScheme.GetRandStr(),err.Error())
		return
	}

	//Memory handlers
	http.HandleFunc("/api/mem/add", addMem)
	http.HandleFunc("/api/mem/getdef", getDefMem)
	http.HandleFunc("/api/mem/getact", getActMem)
	//Disk handlers
	http.HandleFunc("/api/disk/add", addFiles)
	http.HandleFunc("/api/disk/getdef", getDefFiles)
	http.HandleFunc("/api/disk/getact",getActFiles)

	//Start web server
	lisock := fmt.Sprintf("%s:%s",ip,port)
	log.Printf("Starting web server on: %s",lisock)
	log.Fatal(http.ListenAndServe(lisock, nil))
	//Delete all files before exiting
	log.Printf("Deleting all files at %s",fileScheme.GetRandStr())
	deleteTree((&fileScheme))
}

//Free the concurrency memory lock. It's a function so it can be deferred
//value is a pointer because the function is deferred, and value can change during the execution of the calling function
func freeLock(l chan int64, value *int64) {
	l <- *value
}

//Attemps to get hold of the memory lock
func getLock(l chan int64) (int64, bool) {
	select {
	case r := <-l:
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
		defer freeLock(lock, &lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock is available and no pending requests (0)
		defer freeLock(lock, &tstamp) //Make sure the lock is released even if errors occur
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
			fmt.Fprintf(writer, "File size (in bytes) not specified: add?size=<number of bytes>\n")
			tstamp = 0
			return
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
		fmt.Fprintf(writer, "Memory data request sent for %d bytes, with id#: %d, check /api/mem/getact\n", sm, tstamp)
	}
}

//Request the definition of parts
func getDefMem(writer http.ResponseWriter, request *http.Request) {
	lval, islav := getLock(lock)
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(lock, &lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock obtained and no pending request
		var unlock int64 = 0
		defer freeLock(lock, &unlock) //Make sure the lock is released even if error occur
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
		defer freeLock(lock, &lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock obtained and no pending request
		var unlock int64 = 0
		defer freeLock(lock, &unlock) //Make sure the lock is released even if error occur
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

//Create a single directory
func createDir(dirname string) error {
	err := os.Mkdir(dirname,0755)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			fin, errin := os.Stat(dirname)
			if errin == nil && fin.IsDir() { //Directory already exists, that's fine
				return nil
			} else {//Error creating directory
				log.Printf("createDir(): Out error: %s; In error: %s", err,errin)
				return err
			}
		} else { //Error creating directory
			return err
		}
	}
	return nil
}

//Creates the directory tree to store files
func createTree(fc partdisk.FileCollection) error {
	log.Printf("Creating base dir: %s",fc.GetRandStr())
	err := createDir(fc.GetRandStr())
	if err != nil {
		log.Printf("Error creating basedir: %s", fc.GetRandStr())
		return err
	} else {//Base dir created, create subdirs
		for _,size := range fc.GetFileSizes() {
			subdir := fmt.Sprintf("%s/d-%d", fc.GetRandStr(), size)
			log.Printf("\tcreateSubDirs(): creating %s\n",subdir)
			err = createDir(subdir)
			if err != nil {
				log.Printf("Error creating subdir: %s", subdir)
				return err
			}
		}
	}
	return nil
}

//Removes the directory tree and all its contents.  This function is to be called as part of program graceful shutdown.
func deleteTree(fc *partdisk.FileCollection) error {
	log.Printf("Deleting directory tree: %s", fc.GetRandStr())
	err := os.RemoveAll(fc.GetRandStr())
	if err != nil {
		log.Printf("deleteTree(): error deleting directory tree: %s.  Filecollection will be inconsistent",fc.GetRandStr())
		return err
	}
	fc.NewfC(DATADIR)
	return nil
}

//Request the definition of files
func addFiles(writer http.ResponseWriter, request *http.Request) {
	tstamp := time.Now().UnixNano() //Request timestamp
	lval, islav := getLock(filelock)
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(filelock, &lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock is available and no pending requests (0)
		defer freeLock(filelock, &tstamp) //Make sure the lock is released even if errors occur
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
		fmt.Fprintf(writer, "File data request sent for %d bytes, with id#: %d, check /api/disk/getact\n", sm, tstamp)
	}
}

//Shows the definition of files and sizes
func getDefFiles(writer http.ResponseWriter, request *http.Request) {
	lval, islav := getLock(filelock)
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for file allocation
		defer freeLock(filelock, &lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock obtained and no pending request
		var unlock int64 = 0
		defer freeLock(filelock, &unlock) //Make sure the lock is released even if error occur
		mensj := partdisk.GetDefFiles(&fileScheme)
		fmt.Fprintf(writer, mensj)
	}
}

//Request the actual file sizes and distribution
func getActFiles(writer http.ResponseWriter, request *http.Request) {
	lval, islav := getLock(filelock)
	if !islav { //Lock not available
		fmt.Fprintf(writer, "Server busy, try again later\n")
		return
	} else if lval != 0 { //There is a pending request for mem allocation
		defer freeLock(filelock, &lval)
		fmt.Fprintf(writer, "Server contains pending request, try again later\n")
		time.Sleep(1 * time.Second)
		return
	} else { //Lock obtained and no pending request
		var unlock int64 = 0
		defer freeLock(filelock, &unlock) //Make sure the lock is released even if error occur
		mensj := fileScheme.GetActFiles()
		fmt.Fprintf(writer, mensj)
	}
}

//Takes care of cleaning up when the application is terminated by a TERM or INT signal
func gracefulShutdown(sigchan chan os.Signal) {
	wait := <- sigchan
	log.Printf("Signal received: %v",wait)
	err := deleteTree(&fileScheme)
	if err != nil {
		log.Printf("Error shuting down: %s",err.Error())
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
