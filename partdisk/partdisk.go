package partdisk

import (
	"errors"
	"fmt"
	"log"
	"os"
)

//Max number of files for each size that can be created
const limitFiles uint64 = 25

//Holds a representation of the file data
type FileCollection struct {
	//Sizes of files in bytes
	fileSizes []uint64
	//Amount of files per size
	fileAmmount []uint64
}

//Initializes a FileCollection struct
func (fc *FileCollection) NewfC() {
	//                      512Kb   2Mb      8Mb      32Mb      128Mb
	fc.fileSizes = []uint64{524288, 2097152, 8388608, 33554432, 134217728}
	fc.fileAmmount = make([]uint64, len(fc.fileSizes))
}

//Check for directory errors
func checkCreaDirErr(err error) {
	if errors.Is(err, os.ErrPermission) { //Not enough permissions to create the dir
		log.Printf("Error creating directory: %s", err)
	} else if errors.Is(err, os.ErrExist) { //Created in the meanwhile, odd
		log.Printf("The directory already exists: %s", err)
	}
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
//DON'T update fc value, it's a copy not the original
func CreateTree(basedir string, fc FileCollection) error {
	log.Printf("Creating base dir: %s",basedir)
	err := createDir(basedir)
	if err != nil {
		log.Printf("Error creating basedir: %s", basedir)
		return err
	} else {//Base dir created, create subdirs
		for _,size := range fc.fileSizes {
			subdir := fmt.Sprintf("%s/f-%d", basedir, size)
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
	if tsize > hilimit || tsize == 0 {
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