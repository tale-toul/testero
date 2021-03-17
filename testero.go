package main

import (
	"github.com/tale-toul/testero/partmem"
	"log"
	"os"
	"strconv"
	"syscall"
)

func main() {
	sm, err := strconv.ParseUint(os.Args[1], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	
	partScheme := partmem.NewpC()
	
	//Set the high limit for the total size to request
	fmem := freeRam()
	//fmt.Println("Free RAM:", fmem)
	//Compute the number of parts of each size to accomodate the total size. 
	//The result is stored in partScheme
	err = partmem.DefineParts(sm, fmem, &partScheme)
	if err != nil {
		log.Fatal(err)
	} 

	//Create the actual parts
	partmem.CreateParts(&partScheme)
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