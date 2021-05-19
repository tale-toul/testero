package partdisk

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//Max number of files for each size that can be created
const limitFiles uint64 = 25
//Length of random id string
const rstl int = 7

//Holds a representation of the file data
type FileCollection struct {
	//Sizes of files in bytes
	fileSizes []uint64
	//Amount of files per size
	fileAmmount []uint64
	//Last request ID
	flid int64
	//Base dir made of random id string
	frandi string
}

//Get FileCollection random string
func (fc FileCollection) GetRandStr() string {
	return fc.frandi
}

//Get FileCollection fileSizes
func (fc FileCollection) GetFileSizes() []uint64 {
	return fc.fileSizes
}

//Initializes a FileCollection struct
func (fc *FileCollection) NewfC(basedir string) {
	//                      512Kb   2Mb      8Mb      32Mb      128Mb
	fc.fileSizes = []uint64{524288, 2097152, 8388608, 33554432, 134217728}
	fc.fileAmmount = make([]uint64, len(fc.fileSizes))
	fc.frandi = basedir+"/"+randstring(rstl)
}

//Creates a random string made of lower case letters only
func randstring(size int) string {
	rval := make([]byte,size)
	rand.Seed(time.Now().UnixNano())
	for i:=0; i<size; i++ {
		rval[i] = byte(rand.Intn(26) + 97)
	}
	return string(rval)
}

//Creates a random list of bytes with printable ASCII characters
func randbytes(size uint64) []byte {
	const blength int = 1024
	rval := make([]byte,size)
	var base [blength]byte
	var counter, index uint64
	//Fill up the base array with random printable characters
	rand.Seed(time.Now().UnixNano())
	for x:=0; x<len(base); x++ {
		base[x]=byte(rand.Intn(95) + 32) //ASCII 32 to 126
	}
	//Fill the rval slice with pseudorandom characters picked from the base array
	for i:=uint64(0); i<size; i++ {
		//This psuedo random algorith is explained in the documentation
		counter += i + uint64(base[i%uint64(blength)])
		index = counter%uint64(len(base))
		rval[i]=base[index]
		if i%uint64(blength) == 0 {
			counter = uint64(rand.Intn(blength))
		}
	}
	return rval
}

//Get the total number of bytes used up by the files already created
func (fc FileCollection) totalFileSize() (uint64,error)	{
	var tfsize uint64
	for _,fsize := range fc.fileSizes {
		directory := fmt.Sprintf("%s/d-%d",fc.frandi,fsize)
		fileList,err := getFilesInDir(directory)
		if err != nil {
			log.Printf("totalSizeFiles(): Error listing directory: %s\n%s",directory,err.Error())
			return 0,err
		}
		for _,v := range fileList {
			tfsize += uint64(v.Size())
		}
	}
	return tfsize,nil
}

//Compute the number of files of each size required for the size requested
//tsize contains the number of bytes to allocate
//hlimit is the maximum size that can be requested
func DefineFiles(tsize uint64, hilimit uint64, flS *FileCollection) error {
	var nfiles, remain uint64
	tfs, err := flS.totalFileSize() 
	if err != nil {
		log.Printf("DefineFiles(): Error computing total file size: %s", err.Error())
		return err
	}
	if tsize > tfs && tsize > hilimit { //Trying to add files and the total size exceeds the limit
		return fmt.Errorf("Size requested is over the limit: requested %d bytes, limit: %d bytes.", tsize, hilimit)
	}
	for index, fsize := range flS.fileSizes {
		nfiles = tsize / fsize
		remain = tsize % fsize
		if nfiles > limitFiles { //Use all files of this size, keep adding more files of higher capacities
			tsize -= limitFiles * fsize
			flS.fileAmmount[index] = limitFiles
		} else if nfiles == 0 {
			flS.fileAmmount[index] = 0
		} else {
			tsize -= nfiles * fsize
			flS.fileAmmount[index] = nfiles
		}
	}
	if tsize > flS.fileSizes[len(flS.fileSizes)-1] { //The remaining size to allocate is bigger than the biggest file sezie, Add more parts of the maximum size
		nfiles = tsize / flS.fileSizes[len(flS.fileSizes)-1]
		remain = tsize % flS.fileSizes[len(flS.fileSizes)-1]
		flS.fileAmmount[len(flS.fileAmmount)-1] += nfiles
	}
	if remain > 0 { //The remain must be smaller than the bigger file size.
		for index, fsize := range flS.fileSizes {
			if remain <= 3*fsize {
				signRemain := int(remain)
				for signRemain > 0 {
					flS.fileAmmount[index]++
					signRemain -= int(fsize)
				}
				break
			}
		}
	}
	return nil
}

//Prints the number of _file_ elements defined
func GetDefFiles(fS *FileCollection) string {
	var semiTotal uint64
	var rst string
	for index, value := range fS.fileSizes {
		semiTotal += value * fS.fileAmmount[index]
		rst += fmt.Sprintf("Files of size: %d, count: %d, total size: %d\n", value, fS.fileAmmount[index], value*fS.fileAmmount[index])
	}
	rst += fmt.Sprintf("Total size reserved: %d bytes.\n", semiTotal)
	return rst
}

//Generate a message with information about the actual ammount and size of the existing files
func (fc FileCollection) GetActFiles() string {
	var mensj string
	var totalSize int64
	mensj += fmt.Sprintf("Last request ID: %d\n",fc.flid)
	for _,fsize := range fc.fileSizes {
		directory := fmt.Sprintf("%s/d-%d",fc.frandi,fsize)
		fileList,err := getFilesInDir(directory)
		if err != nil {
			log.Printf("GetActFiles(): Error listing directory: %s\n%s",directory,err.Error())
			return "Error getting files information\n"
		} 
		mensj += fmt.Sprintf("Files of size: %d, Count: %d\n", fsize,len(fileList))
		for _,fl := range fileList{
			totalSize += fl.Size()
		}
	}
	mensj += fmt.Sprintf("Total size: %d bytes.\n",totalSize)
	return mensj
}

//Create or remove files to reach the requested number of files of each size
func CreateFiles(fS *FileCollection, ts int64, filelock chan int64) {
	var lt time.Time
	var err error

	select {
	case <- time.After(5 * time.Second):
		//If 5 seconds pass without getting the proper lock, abort
		log.Printf("partdisk.CreateFiles(): timeout waiting for lock\n")
		return
	case chts := <- filelock:
		if chts == ts { //Got the lock and it matches the timestamp received
			//Proceed
			fS.flid = ts
			defer func(){
				filelock <- 0 //Release lock
			}()
			lt = time.Now() //Start counting how long does the parts creation take
			log.Printf("CreateFiles(): lock obtained, timestamps match: %d\n",ts)
		} else {
			log.Printf("CreateFiles(): lock obtained, but timestamps missmatch: %d - %d\n", ts,chts)
			filelock <- chts
			return
		}
	}
	//Lock obtained proper, create/delete the files
	err = adrefiles(fS)
	if err != nil {
		log.Printf("CreateFiles(): Error creating file: %s\n",err.Error())
		return
	}
	log.Printf("CreateFiles(): Request %d completed in %d seconds\n",ts,int64(time.Since(lt).Seconds()))
}

//Add or remove files match the files definition in the FileCollection struct
func adrefiles(fS *FileCollection) error {
	for index,value := range fS.fileSizes {
		directory := fmt.Sprintf("%s/d-%d",fS.frandi,value)
		//Create a list of files in directory
		fileList,err := getFilesInDir(directory)
		if err != nil {
			log.Printf("adrefiles(): Error listing directory: %s",directory)
			return err
		} 
		//Sort the list of files
		sort.Slice(fileList, func(i,j int) bool {
			s1 := strings.TrimLeft(fileList[i].Name(),"f-")
			s2 := strings.TrimLeft(fileList[j].Name(),"f-")
			n1,_ := strconv.ParseInt(s1,10,32)
			n2,_ := strconv.ParseInt(s2,10,32)
			return n1 < n2
		})
		//Get the number of the last file created, 0 if none has been
		var lastfnum uint64
		if len(fileList) > 0 {
			lastfnum,_ = strconv.ParseUint(strings.TrimLeft(fileList[len(fileList)-1].Name(),"f-"),10,32)
		} else {
			lastfnum = 0 
		}
		log.Printf("Last file number: %d",lastfnum)
		
		//Get the total size in bytes consumed by the files
		var tfsize,rqsize,deltasize,fdelta uint64
		for _,v := range fileList {
			tfsize += uint64(v.Size())
			//log.Printf("File: %s - Size: %d",v.Name(),v.Size())
		}
		log.Printf("Total file size in dir %s: %d",directory,tfsize)
		rqsize = fS.fileAmmount[index]*value
		log.Printf("Requested size: %d",rqsize)
		if tfsize > rqsize { //Need to remove files
			deltasize = tfsize - rqsize
			fdelta = deltasize / value
			log.Printf("- Need to remove %d bytes, %d files of size %d",deltasize,fdelta,value)
			for n:=0;n<int(fdelta);n++{
				filename := fmt.Sprintf("%s/d-%d/f-%d",fS.frandi,value,int(lastfnum)-n)
				err = os.Remove(filename)
				if err != nil {
					log.Printf("adrefiles(): error deleting file %s:",filename)
					return err
				}
			}
		} else if tfsize < rqsize { //Need to create files
			deltasize = rqsize - tfsize
			fdelta = deltasize / value
			log.Printf("+ Need to add %d bytes, %d files of size %d",deltasize,fdelta,value)
			for n:=1;n<=int(fdelta);n++ {
				filename := fmt.Sprintf("%s/d-%d/f-%d",fS.frandi,value,n+int(lastfnum))
				err = newFile(filename,value)
				if err != nil {
					log.Printf("adrefiles(): error creating file %s:",filename)
					return err
				}
			}
		} else { //No need to add or remove anything 
			log.Printf("= No need to add or remove any files")
		}
	}
	return nil
}

//Creates a single file of the indicated size
func newFile(filename string, size uint64) error {
	const blength int = 1024
	burval := make([]byte,blength)
	var base [blength]byte
	var counter, index uint64
	//Fill up the base array with random printable characters
	rand.Seed(time.Now().UnixNano())
	for x:=0; x<len(base); x++ {
		base[x]=byte(rand.Intn(95) + 32) //ASCII 32 to 126
	}
	f,err := os.Create(filename)
	defer f.Close()
	if err != nil {
		log.Printf("newFile(): Error creating file: %s",filename)
		return err
	}
	burval = base[:]
	for i:=uint64(0); i<size; i++ {
		counter += i + uint64(base[i%uint64(blength)])
		index = counter%uint64(len(base))
		burval[i%uint64(len(base))]=base[index]
		if i%uint64(blength) == 0 {
			_,err = f.Write(burval)
			if err != nil {
				log.Printf("newFile(): Error writing to file: %s",filename)
				return err
			}
			counter = uint64(rand.Intn(blength))
		}
	}
	return nil
}

//Returns a list of regular files with the correct name, in the directory specified, without 
//directories or other types of files
func getFilesInDir(directory string) ([]os.FileInfo,error) {
	entries, err := ioutil.ReadDir(directory)
	if err != nil {
		log.Printf("getFilesInDir(): Error reading directory: %s",directory)
		return nil,err
	}
	var files []os.FileInfo
	for _,entry := range entries {
		if entry.Mode().IsRegular()  {
			match,_ := regexp.Match("f-[0-9]+",[]byte(entry.Name()))
			if match {
				files = append(files,entry)
			}
		}
	}
	return files,nil
}