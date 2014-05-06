/**
This file contains the structure and methods for javalab.
Use its constructor, newJavaLab, by passing it a name of a zip
file for an exported NetBeans project.
Then use the other methods to build and execute the project.
*/
package javalab

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type JavaLab struct {
	zipFileName        string
	labFolderName      string
	extractPath        string
	labName            string
	templateFolderName string
	secPolicyPath      string
	*log.Logger
}

const (
	questionSrcFolder = "%v/src/edu/carrollcc/cis132/"
	CommandTimeout    = 20 * time.Second
	InputDelay        = 200
)

var (
	rxAddTab *regexp.Regexp
)

func init() {
	rxAddTab = regexp.MustCompile(".*")
}

func New(zipFileName string, secPolicyPath string) (j *JavaLab, err error) {

	//Check if zip file exists
	_, err = ioutil.ReadFile(zipFileName)
	if err != nil {
		fmt.Printf("File %v does not exist\n", zipFileName)
		return nil, err
	}

	//Chop off .zip if necessary
	r := regexp.MustCompile(".zip$")
	labFolderName := r.ReplaceAllString(zipFileName, "")

	//Error if Lab Name does not mach
	r = regexp.MustCompile("(Lab.*?)$|(Midterm.*?)$")
	if !r.MatchString(labFolderName) {
		panic("ZipFileName must be of format \"%%sLab%%s\" (or %%sMidterm) where the string " +
			"before \"Lab\" is the student name, and the string after " +
			" and including \"Lab\" is the lab name. (ex. RuskLabOne.zip) ")
	}
	labName := r.FindString(labFolderName)

	extractPath := fmt.Sprintf("%v", labFolderName)
	labFolderName = fmt.Sprintf("%v/%v", labFolderName, labName)
	templateFolderName := fmt.Sprintf("%v/%v", "LabTemplate", labName)

	return &JavaLab{zipFileName: zipFileName, labFolderName: labFolderName,
		extractPath: extractPath, labName: labName, secPolicyPath: secPolicyPath,
		templateFolderName: templateFolderName,
		Logger:             log.New(os.Stderr, "JavaLab: ", log.Ldate)}, nil
}

/*
	Unzip and build the project zip file
*/
func (j *JavaLab) Build() (err error) {
	// Unzip
	j.Printf("Unzipping %v to %v\n", j.zipFileName, j.extractPath)
	unzipCmd := exec.Command("unzip", "-u", j.zipFileName, "-d", j.extractPath)
	err = unzipCmd.Run()

	// if err != nil {
	// 	return
	// }

	//Copy build file from a template project so build files are the same
	j.Printf("Copying lab files to template project")
	srcFolder := fmt.Sprintf(questionSrcFolder, j.labFolderName)
	destFolder := fmt.Sprintf(questionSrcFolder, j.templateFolderName)
	cpCmd := exec.Command("cp", "-rf", srcFolder, destFolder)
	out, err := cpCmd.CombinedOutput()
	if err != nil {
		j.Printf("%v", string(out))
		return
	}

	//Execute Ant Build
	j.Print("Building project")
	antCmd := exec.Command("ant", "-f", fmt.Sprintf("%v/build.xml", j.templateFolderName))
	out, err = antCmd.CombinedOutput()

	if err != nil {
		j.Printf("%v", string(out))
	}

	return
}

/*
	Delete the unzipped project
*/
func (j *JavaLab) CleanUp() (err error) {
	j.Print("Cleaning up created directories")
	deleteCmd := exec.Command("rm", "-rf", fmt.Sprintf("%v/", j.extractPath))
	err = deleteCmd.Run()
	if err != nil {
		return
	}

	j.Print("Cleaning up Lab Template")
	deleteCmd = exec.Command("rm", "-rf", j.getTemplateSrcPath())
	err = deleteCmd.Run()

	return
}

func (j *JavaLab) RunAndReport() (gradeReportFile *os.File, err error) {
	// Create output file for student
	os.Mkdir("output", 0777)
	gradeReportName := fmt.Sprintf("output/%v.md", j.extractPath)
	gradeReportFile, err = os.Create(gradeReportName)

	defer gradeReportFile.Close()
	gradeReportWriter := bufio.NewWriter(gradeReportFile)

	// Execute each question
	for questionNum := 1; ; questionNum++ {
		// Get the src, if it exists
		questionSrc, err := ioutil.ReadFile(j.getQuestionFile(questionNum))
		if err != nil {
			//The question does not exist, stop running questions
			break
		}

		j.Printf("Running Question %d\n", questionNum)

		gradeReportWriter.WriteString(fmt.Sprintf("# Question %d\n", questionNum))

		j.writeSource(string(questionSrc), fmt.Sprintf("Question%d.java", questionNum), gradeReportWriter)

		classPackageDir := j.getQuestionPackagePath(questionNum)
		classPackageFileInfos, err := ioutil.ReadDir(classPackageDir)
		if err != nil {
			break
		}

		//Find all .java files for the question and print them out
		for _, f := range classPackageFileInfos {
			if !f.IsDir() {
				j.Printf("Found file %v\n", f.Name())
				if strings.HasSuffix(f.Name(), ".java") {
					src, err := ioutil.ReadFile(fmt.Sprintf("%v/%v", classPackageDir, f.Name()))
					if err != nil {
						continue
					}

					j.writeSource(string(src), f.Name(), gradeReportWriter)
				}
			}
		}

		j.runQuestion(questionNum, gradeReportWriter)

	}

	gradeReportWriter.WriteString("\n\n# Summary\n")
	gradeReportWriter.WriteString("\n Total Points: TOGRADE\n")
	gradeReportWriter.Flush()

	return
}

// Gets the .java file for the question number
func (j *JavaLab) getQuestionFile(questionNum int) string {
	return fmt.Sprintf("%v/src/edu/carrollcc/cis132/Question%d.java", j.labFolderName, questionNum)
}

// Gets the package dir for the question number
func (j *JavaLab) getQuestionPackagePath(questionNum int) string {
	return fmt.Sprintf("%v/src/edu/carrollcc/cis132/q%d", j.labFolderName, questionNum)
}

// Gets the Lab Template question source dir for the lab
func (j *JavaLab) getTemplateSrcPath() string {
	return fmt.Sprintf(questionSrcFolder, j.templateFolderName)
}

func (j *JavaLab) writeSource(questionSrc string, fileName string, gradeReportWriter *bufio.Writer) {
	//Output source code
	gradeReportWriter.WriteString(fmt.Sprintf("## %v Source Code\n", fileName))

	gradeReportWriter.WriteString("\n")
	gradeReportWriter.WriteString(indent(questionSrc))
	gradeReportWriter.WriteString("\n")
}

func (j *JavaLab) runQuestion(questionNum int, gradeReportWriter *bufio.Writer) {

	//Loop through each input
	for inputNum := 0; ; inputNum++ {
		inputCmd, err := inputScript(questionNum, inputNum)
		inScriptExists := err == nil

		//Run the input script
		if inScriptExists {
			cmdOut, err := inputCmd.CombinedOutput()
			if err != nil {
				j.Print(err)
			}
			if len(strings.Trim(string(cmdOut), " ")) > 0 {
				gradeReportWriter.WriteString(fmt.Sprintf("\n## Input File #%d\n", inputNum))
				gradeReportWriter.WriteString(indent(string(cmdOut)))
			}
		}

		inputText, err := inputFile(questionNum, inputNum)
		inputExists := err == nil

		if !inputExists && inputNum > 0 {
			//If the input doesn't exist, ext
			break
		}

		// Run the question

		questionCmd := exec.Command("java", "-Djava.security.manager",
			fmt.Sprintf("-Djava.security.policy=%v", j.secPolicyPath),
			"-cp",
			fmt.Sprintf("%v/dist/%v.jar", j.templateFolderName, j.labName),
			fmt.Sprintf("edu.carrollcc.cis132.Question%d", questionNum))

		var questionOutput bytes.Buffer
		questionCmd.Stdout, questionCmd.Stderr = &questionOutput, &questionOutput

		stdin, err := questionCmd.StdinPipe()
		if err != nil {
			panic(err)
		}

		// Start the process
		if err := questionCmd.Start(); err != nil {
			j.Printf("failed to start command: %s", err)
		}

		if inputExists {
			// Display question input
			gradeReportWriter.WriteString(fmt.Sprintf("\n## Stdin Buffer #%d\n", inputNum))
			gradeReportWriter.WriteString(indent(string(inputText)))

			// Pass the file as a bufio reader
			bufferedWriter := bufio.NewWriter(stdin)
			go func() {
				// Use a delay per line to handle multiple Scanners on input.
				for _, value := range strings.Split(string(inputText), "\n") {
					bufferedWriter.WriteString(value + "\n")
					bufferedWriter.Flush()
					time.Sleep(InputDelay * time.Millisecond)
				}
			}()
		}

		// Kill the process if it doesn't exit in time
		defer time.AfterFunc(CommandTimeout, func() {
			j.Printf("command timed out")
			questionCmd.Process.Kill()
		}).Stop()

		// Wait for the process to finish
		if err := questionCmd.Wait(); err != nil {
			j.Printf("command failed: %s", err)
			questionOutput.WriteString("\nExecution failed.")
		} else {
			j.Print("Exited gracefully")
		}

		//Display Expected Execution Output if available
		outputText, err := outputFile(questionNum, inputNum)
		if err == nil {
			//Output file exists
			gradeReportWriter.WriteString(fmt.Sprintf("\n## Expected Execution Output #%d\n", inputNum))
			gradeReportWriter.WriteString(indent(string(outputText)))
		}

		gradeReportWriter.WriteString(fmt.Sprintf("\n## Actual Execution Output #%d\n", inputNum))
		//Truncate the output so it only has the first 5000 characters
		var truncatedOutput []byte
		if questionOutput.Len() > 5000 {
			truncatedOutput = questionOutput.Bytes()[:5000]
		} else {
			truncatedOutput = questionOutput.Bytes()
		}
		gradeReportWriter.WriteString(indent(string(truncatedOutput)))
		gradeReportWriter.WriteString("\n\n\n")

		outputCmd, err := outputScript(questionNum, inputNum)
		outScriptExists := err == nil

		//Run the output script
		if outScriptExists {
			cmdOut, err := outputCmd.CombinedOutput()
			if err != nil {
				j.Print(err)
			}
			if len(strings.Trim(string(cmdOut), " ")) > 0 {
				gradeReportWriter.WriteString(fmt.Sprintf("\n## Output File #%d\n", inputNum))
				gradeReportWriter.WriteString(indent(string(cmdOut)))
				gradeReportWriter.WriteString("\n\n\n")
			}
		}

		gradeReportWriter.Flush()
	}

	//Display grading area
	rubricText, err := rubricFile(questionNum)
	if err == nil {
		//Rubric file exists
		gradeReportWriter.WriteString(string(rubricText))
		gradeReportWriter.WriteString("\n")
	} else {
		//Default grading
		gradeReportWriter.WriteString("\nPoints: TOGRADE\n")
		gradeReportWriter.WriteString("\nFeedback: TOGRADE\n\n")
	}

	gradeReportWriter.Flush()
}

func inputScript(questionNum int, inputNum int) (*exec.Cmd, error) {
	fileName := fmt.Sprintf("q%din%d.sh", questionNum, inputNum)
	_, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	command := exec.Command("./" + fileName)
	return command, nil
}

func outputScript(questionNum int, inputNum int) (*exec.Cmd, error) {
	fileName := fmt.Sprintf("q%dout%d.sh", questionNum, inputNum)
	_, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	command := exec.Command("./" + fileName)

	return command, nil
}

func inputFile(questionNum int, inputNum int) ([]byte, error) {
	return ioutil.ReadFile(fmt.Sprintf("q%din%d.txt", questionNum, inputNum))
}

func outputFile(questionNum int, inputNum int) ([]byte, error) {
	return ioutil.ReadFile(fmt.Sprintf("q%dout%d.txt", questionNum, inputNum))
}

func rubricFile(questionNum int) ([]byte, error) {
	return ioutil.ReadFile(fmt.Sprintf("q%drubric.txt", questionNum))
}

func indent(text string) string {
	return rxAddTab.ReplaceAllString(text, "\t$0")
}
