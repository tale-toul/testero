package cpuload

import (
	"fmt"
	"log"
	"math/big"
	"time"
)

//Contains information about the CPU load task
type CpuCollection struct {
	clid int64  //Request ID corresponds to the Unix time when the request was sent
	lapse uint64 //Request load time in seconds
	bfn *big.Int //Number to factor
}

//Locking and communication channels
var foundFactors chan []*big.Int
var quit chan bool

//Get latest request load time
func (cc CpuCollection) GetDuration() uint64 {
	return cc.lapse
}

//Get time when the request was made
func (cc CpuCollection) GetReqTime() time.Time {
	tsecs := cc.clid / 1000000000 
	return time.Unix(tsecs,0)
}

//Initialize a CpuCollection object
func (cc *CpuCollection) NewCc(numtofactor string) {
	var bigSuccess bool

	cc.clid = 0 //To make it explicit
	cc.bfn, bigSuccess = new(big.Int).SetString(numtofactor, 10)
	if !bigSuccess  {
		panic("Invalid number to factor: NUMTOFACTOR="+ numtofactor)
	}
}

//Start a timer and launch the load generator, wait for the quickest to end
func LoadUp(cS *CpuCollection, ts int64, duration uint64, lock chan int64) {
	select {
	case <- time.After(5 * time.Second): //If 5 seconds pass without getting the proper lock, abort
		log.Printf("cpuload.LoadUp(): timeout waiting for lock")
		return
	case chts := <- lock:
		if chts == ts { //Got the lock and if it matches the timestamp received, proceed
			cS.clid = ts
			cS.lapse = duration
			defer func(){
				lock <- 0 //Release lock
			}()
			log.Printf("cpuload.LoadUp(): lock obtained, timestamps match: %d\n",ts)
		} else {
			log.Printf("cpuload.LoadUp(): lock obtained, but timestamps missmatch: %d - %d\n", ts,chts)
			lock <- chts
			return
		}
	}
	var returnedFactors []*big.Int
	foundFactors = make(chan []*big.Int,1)
	quit = make(chan bool,1)
	log.Printf("Load CPU for %d seconds factoring number: %d", duration,cS.bfn)
	go factor(cS.bfn)
	select {
	case <- time.After(time.Duration(duration) * time.Second):
		log.Printf("CPU high load for %d seconds elapsed",duration)
		quit <- true
	case returnedFactors = <-foundFactors:
		log.Printf("Factors found: %v", returnedFactors)
	}
}

// Finds the factors of inNum
func factor(inNum *big.Int) {
	//Candidate factors
	c := big.NewInt(2)
	//List of found factors
	var outFactors []*big.Int
	//Higher possible factor candidate; temp divmod result; modulus for the division
	topc := new(big.Int)
	tempDm := new(big.Int)
	modulus := new(big.Int)
	//Zero, One, Two as big Int
	zero := big.NewInt(0)
	one := big.NewInt(1)
	two := big.NewInt(2)

	// Zero is not a valid number to factorize
	if inNum.Cmp(zero) == 0 {
		log.Printf("cpuload.factor(): Invalid argument 0")
		foundFactors <- outFactors //Returns an empty slice
		return
	}
	topc.Sqrt(inNum)

	//Check 2 as candidate. 
	for c.Cmp(topc) != 1  {		// While c <= topc
		tempDm, modulus = tempDm.DivMod(inNum, c, modulus)
		if modulus.Cmp(zero) == 0 {
			outFactors = append(outFactors, new(big.Int).Set(c))  //save a copy of factor 
			inNum.Set(tempDm) //Remaining the number to factor
			topc.Sqrt(inNum)  //New top candidate
		} else {
			c.Add(c,one) //c++
			break
		}
	}
	for c.Cmp(topc) != 1  {   // While c <= topc
		select {
		case <-quit:
			log.Printf("cpuload.Factor(): Quiting early, external signal")
			foundFactors <- outFactors
			return
		default: //Keep factoring
			tempDm, modulus = tempDm.DivMod(inNum, c, modulus)
			if modulus.Cmp(zero) == 0 {
				outFactors = append(outFactors, new(big.Int).Set(c))
				inNum.Set(tempDm)
				topc.Sqrt(inNum)
			} else {
				c.Add(c,two) //c += 2
			}
		}
	}	
	outFactors = append(outFactors, inNum) //No need to make a copy because it is the last one
	foundFactors <- outFactors
}

//Stops the current factoring of a number if the ID requested match
func StopLoad(cS CpuCollection, id int64) string {
	if id != cS.clid { //IDs don't match, go away
		log.Printf("cpuload.StopLoad(): Stop request ID (%d) does not match last load request ID (%d)",id,cS.clid)
		time.Sleep(1 * time.Second)
		return fmt.Sprintf("Incorrect stop load request ID=%d\n",id)
	} else { //IDs match
		log.Printf("cpuload.StopLoad(): IDs match, stoping CPU load")
		quit <- true
		return fmt.Sprintf("CPU load stopped\n")
	}
}