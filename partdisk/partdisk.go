package partdisk

import (
	"errors"
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

//Initializes a FileCollection struct
func (fc *FileCollection) NewfC(basedir string) {
	//                      512Kb   2Mb      8Mb      32Mb      128Mb
	fc.fileSizes = []uint64{524288, 2097152, 8388608, 33554432, 134217728}
	fc.fileAmmount = make([]uint64, len(fc.fileSizes))
	fc.frandi = basedir+"/"+randstring(rstl)
}

//Creates a random string 
func randstring(size int) string {
	rval := make([]byte,size)
	rand.Seed(time.Now().Unix())
	for i:=0; i<size; i++ {
		rval[i]=byte(rand.Intn(26) + 97)
	}
	return string(rval)
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
func createTree(fc *FileCollection) error {
	log.Printf("Creating base dir: %s",fc.frandi)
	err := createDir(fc.frandi)
	if err != nil {
		log.Printf("Error creating basedir: %s", fc.frandi)
		return err
	} else {//Base dir created, create subdirs
		for _,size := range fc.fileSizes {
			subdir := fmt.Sprintf("%s/d-%d", fc.frandi, size)
			log.Printf("createSubDirs(): creating %s\n",subdir)
			err = createDir(subdir)
			if err != nil {
				log.Printf("Error creating subdir: %s", subdir)
				return err
			}
		}
	}
	return nil
}

//Compute the number of files of each size require for the size requested
//tsize contains the number of bytes to allocate
//hlimit is the maximum size that can be requested
func DefineFiles(tsize uint64, hilimit uint64, flS *FileCollection) error {
	var nfiles, remain uint64
	if tsize > hilimit || tsize == 0 { //*THIS NEEDS TO BE CHANGED, BECAUSE THE REQUESTED SIZE MAY BE LOWER THAN THE ACTUAL USED SIZE IMPLYING A FILE DELETION, BUT GREATER THAN THE AVAILABLE SIZE WHICH WOULD BE REJECTED
		return fmt.Errorf("Invalid total size %d.  High limit is %d bytes.", tsize, hilimit)
	}
	for index, fsize := range flS.fileSizes {
		nfiles = tsize / fsize
		remain = tsize % fsize
		if nfiles > limitFiles { //Keep adding more parts
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
func GetDefFiles (fS *FileCollection) string {
	var semiTotal uint64
	var rst string
	for index, value := range fS.fileSizes {
		semiTotal += value * fS.fileAmmount[index]
		rst += fmt.Sprintf("Files of size: %d, count: %d, total size: %d\n", value, fS.fileAmmount[index], value*fS.fileAmmount[index])
	}
	rst += fmt.Sprintf("Total size reserved: %d bytes.\n", semiTotal)
	return rst
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
			log.Printf("partmem.CreateParts(): lock obtained, timestamps match: %d\n",ts)
		} else {
			log.Printf("partmem.CreateParts(): lock obtained, but timestamps missmatch: %d - %d\n", ts,chts)
			filelock <- chts
			return
		}
	}
	//Lock obtained proper, create/delete the files
	err = createTree(fS)
	if err != nil {
		log.Printf("CreateFiles(): Error creating directory tree: %s\n%s\n",fS.frandi,err.Error())
		return
	}
	//Directory tree created, now for the files
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
		//Get the total size in bytes consumed by the files
		var tfsize,rqsize,deltasize,fdelta uint64
		for _,v := range fileList {
			tfsize += uint64(v.Size())
			log.Printf("File: %s - Size: %d",v.Name(),v.Size())
		}
		log.Printf("Total file size in dir %s: %d",directory,tfsize)
		rqsize = fS.fileAmmount[index]*value
		log.Printf("Requested size: %d",rqsize)
		if tfsize > rqsize { //Need to remove files
			deltasize = tfsize - rqsize
			fdelta = deltasize / value
			log.Printf("Need to remove %d bytes, %d files of size %d",deltasize,fdelta,value)
		} else if tfsize < rqsize { //Need to create files
			deltasize = rqsize - tfsize
			fdelta = deltasize / value
			log.Printf("Need to add %d bytes, %d files of size %d",deltasize,fdelta,value)
		} else { //No need to add or remove anything 
			log.Printf("No need to add or remove any files")
		}

		//Get the number of the last file created, 0 if none has been created
		var lastfnum uint64
		if len(fileList) > 0 {
			lastfnum,_ = strconv.ParseUint(strings.TrimLeft(fileList[len(fileList)-1].Name(),"f-"),10,32)
		} else {
			lastfnum = 0 
		}
		log.Printf("Last file number: %d",lastfnum)

		for n:=1;n<=int(fdelta);n++ {
			filename := fmt.Sprintf("%s/d-%d/f-%d",fS.frandi,value,n)
			f,err := os.Create(filename)
			f.Close()
			if err != nil {
				log.Printf("adrefiles(): Error creating file: %s",filename)
				return err
			}
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