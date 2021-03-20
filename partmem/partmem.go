package partmem

import (
	"fmt"
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
}

//Computes the actual number of parts and its sizes
func (pc PartCollection) GetActParts() string {
	var mensj string
	var totalSize uint64
	for index, value := range pc.partLists {
		count := 0 
		mensj += fmt.Sprintf("Parts of size: %d, ", pc.partSizes[index])
		for value != nil {
			count++
			totalSize += uint64(len(value.data))
			value = value.next
		}
		mensj += fmt.Sprintf("Count: %d\n", count)
	}
	mensj += fmt.Sprintf("Total size: %d\n",totalSize)
	return mensj
}

//Prints the actual list of _apart_ elements of every size
func (pc PartCollection) Crawl() {
	fmt.Printf("Crawling the data\n")
	fmt.Printf("pc.partSizes: %v\n", pc.partSizes)
	fmt.Printf("pc.partLists: %v\n", pc.partLists)
	for index, _ := range pc.partSizes {
		p := pc.partLists[index]
		x := 1
		for p != nil {
			fmt.Printf("#%d Element size: %d\n", x, len(p.data))
			p = p.next
			x++
		}
	}
}

//Creates a new instance of partCollection 
func NewpC() PartCollection {
	var pC PartCollection
	//Works best when each size is 4x the previous one
	//                      250Kb,  1Mb,     4Mb,     16Mb,     64Mb
	pC.partSizes = []uint64{256000, 1048576, 4194304, 16777216, 67108864}
	pC.partAmmount = make([]uint64, len(pC.partSizes))
	pC.partLists = make([]*apart, len(pC.partSizes))
	return pC
}

//Compute the number and sizes of parts to accomodate the total size
//tsize is the number of bytes to partition
//hilimit is the maximum number of bytes allowed to partition
func DefineParts(tsize uint64, hilimit uint64, ptS *PartCollection) error {
	var nparts, remain uint64

	if tsize > hilimit || tsize == 0 {
		return fmt.Errorf("Invalid total size %d.  High limit is %d bytes.", tsize, hilimit)
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
	rand.Seed(time.Now().Unix())
	for x := 0; x < len(part); x++ {
		part[x] = byte(rand.Intn(95) + 32)
	}
}

//Create or remove parts to reach the expected number of parts as defined in the partCollection parameter
func CreateParts(ptS *PartCollection) string {
	var pap *apart
	var mensj string
	for index, value := range ptS.partSizes {
		desirednumParts := ptS.partAmmount[index]
		//mensj += fmt.Sprintf("Desired number of parts: %d:\n", desirednumParts)
		pap = ptS.partLists[index]
		if desirednumParts == 0 {
			ptS.partLists[index] = nil
		} else if pap == nil && desirednumParts > 0 { //Create the first element
			//mensj += fmt.Sprintf("Create 1st elment")
			var newpart apart
			newpart.data = make([]byte, value)
			fillPart(newpart.data)
			ptS.partLists[index] = &newpart
			pap = &newpart
		}
		for i := uint64(1); i < desirednumParts; i++ {
			if pap.next == nil {
				//mensj += fmt.Sprintf("\nCreate elm #%d; ",i)
				var newpart apart
				newpart.data = make([]byte, value)
				fillPart(newpart.data)
				pap.next = &newpart
			}
			if pap != nil {
				pap = pap.next
				//mensj += fmt.Sprintf("elmi #%d; ",i)
			}
		}
		if pap != nil && desirednumParts > 0 {
			//mensj += fmt.Sprintf("\nLast element pointer to nil.\n")
			pap.next = nil
		} 
	}
	return mensj
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