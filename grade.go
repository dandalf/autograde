package main

import (
	"fmt"
	"github.com/dandalf/autograde/lab/javalab"
	"log"
	"os"
	"os/exec"
)

func init() {
}

func check(e error) {
	if e != nil {
		log.Fatal("Fatal Error :", e)
		panic(e)
	}
}

type ExtProg struct {
	cmdName string
	args    []string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Proper usage: %v NameLabOne.zip\n", os.Args[0])
		return
	}

	secPolicyPath := fmt.Sprintf("secpolicy")
	log.Printf("Loading java sec policy %v", secPolicyPath)

	// Load the arguments
	zipFileName := os.Args[1]

	studentLab, err := javalab.New(zipFileName, secPolicyPath)
	check(err)

	// Begin processing the lab
	fmt.Printf("Processing %v\n", zipFileName)

	err = studentLab.Build()
	check(err)

	gradeReportFile, err := studentLab.RunAndReport()

	// Clean up files
	if err = studentLab.CleanUp(); err != nil {
		check(err)
	}

	// Open up textmate to grade output file
	textMateCmd := exec.Command("mate", gradeReportFile.Name())
	textMateCmd.Run()
}
