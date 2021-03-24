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

//Create sub directories
func createSubDirs(basedir string, fc FileCollection) error {
	for size := range fc.fileSizes {
		subdir := fmt.Sprintf("%s/f-%d", basedir, size)
		log.Printf("createSubDirs(): creating %s\n",subdir)
		err := os.Mkdir(subdir, 0775)
		if err != nil {
			checkCreaDirErr(err)
			return err
		}
	}
	return nil
}

//Creates the directory tree to store files
//DON'T CHANGE fc, it's a copy not the original
func CreateTree(basedir string, fc FileCollection) error {
	finfo, err := os.Stat(basedir)
	if err != nil {
		if errors.Is(err, os.ErrPermission) { //Not enough permissions to get the dir
			log.Printf("Error reading directory: %s", err)
			return err
		} else if errors.Is(err, os.ErrNotExist) { //The directory does not exist, create it
			err = os.Mkdir(basedir, 0775)
			if err != nil {
				checkCreaDirErr(err)
				return err
			} else { //Create subdirectories
				err = createSubDirs(basedir, fc)
				if err != nil {
					return err
				}
			}
		} else {
			log.Printf("Unexpected error: %s", err)
		}
	} else if !finfo.IsDir() { //Not a directory
		log.Printf("Not a directory: %s", finfo.Name())
		return err
	} else { //Directory exists
		err = createSubDirs(basedir, fc)
		if err != nil {
			return err
		}
	}
	return nil
}
