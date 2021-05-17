package partmem

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

//Number of parts of every size to aim for
const limitParts uint64 = 20

//Defines the individual component holding the data, and a pointer to the next one
type apart struct {
	data []byte
	next *apart
}

//Type to hold boxes sizes and the amount of boxes of each size
type PartCollection struct {
	//Part sizes in bytes
	partSizes []uint64
	//Amount of each of the parts
	partAmmount []uint64
	//Lists of actual parts with data
	partLists []*apart
	//Last request ID
	lid int64
}

//Computes the actual number of parts and its sizes
func (pc PartCollection) GetActParts(dump string) string {
	var mensj string
	var totalSize uint64
	mensj += fmt.Sprintf("Last request ID: %d\n",pc.lid)
	for index, value := range pc.partLists {
		count := 0 
		mensj += fmt.Sprintf("Parts of size: %d, ", pc.partSizes[index])
		for value != nil {
			count++
			totalSize += uint64(len(value.data))
			if dump == "true" {
				log.Printf("\n\nPointer value:%v\nFile contents:\n%s",&value.next,value.data)
			}
			value = value.next
		}
		mensj += fmt.Sprintf("Count: %d\n", count)
	}
	mensj += fmt.Sprintf("Total size: %d bytes.\n",totalSize)
	return mensj
}

//Computes the total size in bytes used up by the momory parts
func (pc PartCollection) sizeOfParts() uint64 {
	var tmsize uint64
	for _, value := range pc.partLists { //For every list of parts of each size
		for value != nil { //Sum up the actual parts size
			tmsize += uint64(len(value.data))
			value = value.next
		}
	}
	return tmsize
}

//Creates a new instance of partCollection.  No need to initialize lid
func NewpC() PartCollection {
	var pC PartCollection
	//Works best when each size is 4x the previous one
	//                      250Kb,  1Mb,     4Mb,     16Mb,     64Mb
	pC.partSizes = []uint64{262144, 1048576, 4194304, 16777216, 67108864}
	pC.partAmmount = make([]uint64, len(pC.partSizes))
	pC.partLists = make([]*apart, len(pC.partSizes))
	return pC
}

//Compute the number and sizes of parts to accomodate the total size
//tsize is the number of bytes to partition
//hilimit is the maximum number of bytes allowed to partition
func DefineParts(tsize uint64, hilimit uint64, ptS *PartCollection) error {
	var nparts, remain, usedSize uint64

	usedSize = ptS.sizeOfParts() //The memory being used up at the moment

	if tsize > usedSize && tsize > hilimit { //If the requested size bigger than the currently used memory, and the increment is bigger than the limit
		return fmt.Errorf("Size requested is over the limit: requested %d bytes, limit: %d bytes.", tsize, hilimit)
	}
	for index, psize := range ptS.partSizes {
		nparts = tsize / psize
		remain = tsize % psize
		if nparts > limitParts { //Keep adding more parts
			tsize -= limitParts * psize
			ptS.partAmmount[index] = limitParts
		} else if nparts == 0 {
			ptS.partAmmount[index] = 0
		} else {
			tsize -= nparts * psize
			ptS.partAmmount[index] = nparts
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
	rand.Seed(time.Now().UnixNano())
	for x := 0; x < len(part); x++ {
		part[x] = byte(rand.Intn(95) + 32)
	}
}

//Create or remove parts to reach the expected number of parts as defined in the partCollection parameter
func CreateParts(ptS *PartCollection, ts int64, lock chan int64) {
	var pap *apart
	var lt time.Time

	select {
	case <- time.After(5 * time.Second):
		//If 5 seconds pass without getting the proper lock, abort
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

	for index, value := range ptS.partSizes {
		desirednumParts := ptS.partAmmount[index]
		pap = ptS.partLists[index]
		if desirednumParts == 0 {
			ptS.partLists[index] = nil
		} else if pap == nil && desirednumParts > 0 { //Create the first element
			var newpart apart
			newpart.data = make([]byte, value)
			fillPart(newpart.data)
			ptS.partLists[index] = &newpart
			pap = &newpart
		}
		for i := uint64(1); i < desirednumParts; i++ {
			if pap.next == nil {
				var newpart apart
				newpart.data = make([]byte, value)
				fillPart(newpart.data)
				pap.next = &newpart
			}
			if pap != nil {
				pap = pap.next
			}
		}
		if pap != nil && desirednumParts > 0 {
			pap.next = nil
		} 
	}
	log.Printf("CreateParts(): Request %d completed in %d seconds\n",ts,int64(time.Since(lt).Seconds()))
}

//Prints the number of _apart_ elements defined
func GetDefParts (pS *PartCollection) string {
	var semiTotal uint64
	var rst string
	for index, value := range pS.partSizes {
		semiTotal += value * pS.partAmmount[index]
		rst += fmt.Sprintf("Boxes of size: %d, count: %d, total size: %d\n", value, pS.partAmmount[index], value*pS.partAmmount[index])
	}
	rst += fmt.Sprintf("Total size reserved: %d bytes.\n", semiTotal)
	return rst
}