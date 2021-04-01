package cpuload

import (
	"log"
	"math/big"
	"time"
)

const bfn string = "49344058972249501099"
//const bfn string = "1234567890723456781" //Shorter and faster number

func LoadUp(ts int64, duration uint64, lock chan int64) {
	select {
	case <- time.After(5 * time.Second): //If 5 seconds pass without getting the proper lock, abort
		log.Printf("cpuload.LoadUp(): timeout waiting for lock\n")
		return
	case chts := <- lock:
		if chts == ts { //Got the lock and it matches the timestamp received, proceed
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

	var foundFactors []*big.Int
	var bNum *big.Int
	var bigSuccess bool

	bNum, bigSuccess = new(big.Int).SetString(bfn, 10)
	if bigSuccess {
		log.Printf("Number to factor: %d", bNum)
		foundFactors = factor(bNum)
		log.Printf("Factors found: %v", foundFactors)
	} else {
		log.Printf("cpuload.LoadUp(): Invalid input number: %s",bfn)
	}
}

// This function is intentionally ineficient
func factor(inNum *big.Int) []*big.Int {
	//Candidate factors
	c := big.NewInt(2)
	//List of found factors
	var outFactors []*big.Int
	//Higher possible factor candidate; temp divmod result; modulus for the division
	topc := new(big.Int)
	tempDm := new(big.Int)
	modulus := new(big.Int)
	//Zero, One, Two big Int
	zero := big.NewInt(0)
	one := big.NewInt(1)
	two := big.NewInt(2)

	// Zero is not a valid number to factorize
	if inNum.Cmp(zero) == 0 {
		log.Printf("cpuload.factor(): Invalid argument 0")
		return nil
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
		tempDm, modulus = tempDm.DivMod(inNum, c, modulus)
		if modulus.Cmp(zero) == 0 {
			outFactors = append(outFactors, new(big.Int).Set(c))
			inNum.Set(tempDm)
			topc.Sqrt(inNum) 
		} else {
			c.Add(c,two) //c += 2
		}
	}	
	outFactors = append(outFactors, inNum) //No need to make a copy because it is the last one
	return outFactors
}
