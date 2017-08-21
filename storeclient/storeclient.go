package main

import (
	"flag"
	"github.com/yasushi-saito/go-netdicom"
	"io/ioutil"
	"log"
)

var (
	fileFlag   = flag.String("file", "", "the DICOM file you want to parse")
	serverFlag = flag.String("server", "", "host:port of the DICOM service provider")
)

func main() {
	flag.Parse()

	if *serverFlag == "" || *fileFlag == "" {
		log.Fatal("Both --server and --file must be set")
	}

	params := netdicom.NewServiceUserParams(
		*serverFlag, "dontcare", "testclient", netdicom.StorageClasses)
	su := netdicom.NewServiceUser(params)
	data, err := ioutil.ReadFile(*fileFlag)
	if err != nil {
		log.Fatalf("%s: %v", *fileFlag, err)
	}
	su.CStore(data)
	err = su.Release()
	if err != nil {
		log.Fatalf("Release failed: %v", err)
	}
}
