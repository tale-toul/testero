package main

import (
	"fmt"
	"github.com/tale-toul/testero/partmem"
	"log"
	"net/http"
	"strconv"
	"syscall"
)

var partScheme partmem.PartCollection

func main() {
	partScheme = partmem.NewpC()
	http.HandleFunc("/api/mem/add", addMem)
	http.HandleFunc("/api/mem/getdef", getDefMem)
	http.HandleFunc("/api/mem/getact", getActMem)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func addMem(writer http.ResponseWriter, request *http.Request) {
	bsm := request.URL.Query().Get("size")
	var sm uint64
	var err error
	if bsm != "" {
		sm, err = strconv.ParseUint(bsm, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Fprintf(writer, "No data size specified\n")
		return
	}

	//Set the high limit for the total size to request
	fmem := freeRam()

	//Compute the number of parts of each size to accomodate the total size.
	//The result is stored in partScheme
	err = partmem.DefineParts(sm, fmem, &partScheme)
	if err != nil {
		log.Fatal(err)
	}

	//Create the actual parts
	fmt.Fprintf(writer, partmem.CreateParts(&partScheme))

	fmt.Fprintf(writer, "Data added\n")
}

//Request the definition of parts
func getDefMem(writer http.ResponseWriter, request *http.Request) {
	mensj := partmem.GetDefParts(&partScheme)
	fmt.Fprintf(writer, mensj)
}

//Request the actual definition of parts created
func getActMem(writer http.ResponseWriter, request *http.Request) {
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
