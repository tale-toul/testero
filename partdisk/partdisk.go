package partdisk

import (
	"errors"
	"fmt"
	"log"
	"os"
)

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