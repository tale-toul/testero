package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"syscall"
	"time"
)

func main() {
	sm, err := strconv.ParseUint(os.Args[1], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	
	//250Kb, 1Mb, 4Mb, 16Mb, 64Mb
	var partScheme partCollection
	partScheme.partSizes = []uint64{256000, 1048576, 4194304, 16777216, 67108864}
	partScheme.partAmmount = make([]uint64, len(partScheme.partSizes))
	partScheme.partLists = make([]*apart, len(partScheme.partSizes))

	//Set the high limit for the total size to request
	fmem := freeRam()
	fmt.Println("Free RAM:", fmem)
	//Compute the number of parts of each size to accomodate the total size. 
	//The result is stored in partScheme
	err = defineParts(sm, fmem, &partScheme)
	if err != nil {
		log.Fatal(err)
	} else {
		var semiTotal uint64
		for index, value := range partScheme.partSizes {
			semiTotal += value * partScheme.partAmmount[index]
			fmt.Printf("Boxes of size: %d, count: %d, total size: %d\n", value, partScheme.partAmmount[index], value*partScheme.partAmmount[index])
		}
		fmt.Printf("Total size reserved: %d\n", semiTotal)
	}
	//Create the actual parts
	createParts(&partScheme)
	partScheme.crawl()
	//fmt.Scanln()

	/*
		for _, value := range ar256 {
			fmt.Printf("%c",value)
		}
		fmt.Println()
	*/
}

//Number of parts of every size to aim for
const limitParts uint64 = 20

//Type to hold boxes sizes and the amount of boxes of each size
type partCollection struct {
	//Part sizes in bytes
	partSizes []uint64
	//Amount of each of the parts
	partAmmount []uint64
	//Lists of actual parts with data
	partLists []*apart
}

//Counts the number of elements for a particular size
func (pc partCollection) countPS (size uint64) (int) {
	var count int = 0
	for index,value := range pc.partSizes {
		if size == value {
			var pntP *apart = pc.partLists[index]
			for pntP != nil {
				count++
				pntP = pntP.next
			}
		}
	}
	return count
}

//
func (pc partCollection) crawl() {
	fmt.Printf("Crawling the data\n")
	fmt.Printf("pc.partSizes: %v\n", pc.partSizes)
	fmt.Printf("pc.partLists: %v\n", pc.partLists)
	for index,_ := range pc.partSizes {
		p := pc.partLists[index]
		x := 1
		for p != nil {
			fmt.Printf("#%d Element size: %d\n",x,len(p.data))
			p=p.next
			x++
		}
	}
}
//Defines the component which holds the data assigned to it, and a pointer to the next one
type apart struct {
	data []byte
	next *apart
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

//Compute the number and sizes of parts to accomodate the total size
//tsize is the number of bytes to partition
//hilimit is the maximum number of bytes allowed to partition
func defineParts(tsize uint64, hilimit uint64, ptS *partCollection) error {
	var nparts, remain uint64

	if tsize > hilimit || tsize == 0 {
		return fmt.Errorf("Invalid total size %d.  High limit is %d", tsize, hilimit)
	}
	for index, psize := range ptS.partSizes {
		nparts = tsize / psize
		remain = tsize % psize
		fmt.Printf("This number of parts: %d of size: %d\n", nparts, psize)
		if nparts > limitParts { //Keep adding more parts
			tsize -= limitParts * psize
			ptS.partAmmount[index] = limitParts
		} else {
			tsize -= nparts * psize
			ptS.partAmmount[index] = nparts
			break //No more parts to add, excep a possible remainder
		}
	}
	if tsize > ptS.partSizes[len(ptS.partSizes)-1] { //Add more parts of the maximum size
		nparts = tsize / ptS.partSizes[len(ptS.partSizes)-1]
		remain = tsize % ptS.partSizes[len(ptS.partSizes)-1]
		ptS.partAmmount[len(ptS.partAmmount)-1] += nparts
	}
	if remain > 0 { //Take care of the reaminder
		for index, psize := range ptS.partSizes {
			if remain <= 3*psize {
				signRemain := int(remain)
				for signRemain > 0 {
					ptS.partAmmount[index]++
					signRemain -= int(psize)
				}
				break
			}
		}
	}
	return nil
}

//Fill the part array with random bytes from the writable section of the ASCI chart.
func fillPart(part []byte) {
	rand.Seed(time.Now().Unix())
	for x := 0; x < len(part); x++ {
		part[x] = byte(rand.Intn(95) + 32)
	}
}

//Create the expected parts, or delete if there are more parts than should from privious runs
func createParts(ptS *partCollection) {
	var pap *apart 
	for index,value := range ptS.partSizes {
		desirednumParts := ptS.partAmmount[index]
		pap = ptS.partLists[index]
		if pap == nil && desirednumParts > 0 { //First element
			var newpart apart
			newpart.data = make([]byte,value) 
			fillPart(newpart.data)
			ptS.partLists[index] = &newpart
			pap = &newpart
		}
		for i:=uint64(1); i<desirednumParts; i++ {
			if pap.next == nil {  
				var newpart apart
				newpart.data = make([]byte,value) 
				fillPart(newpart.data)
				pap.next = &newpart
			}			
			if pap != nil {
				pap = pap.next
			}
		}		
	}
	if pap !=nil {
		pap.next = nil
	}
}

