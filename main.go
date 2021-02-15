package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type Result struct {
	toStdout string
	toStderr string
}

type Response struct {
	Completion int
	Results    []Result
}

type Test struct {
	Input  string
	Output string
}

type Exercise struct {
	Tests    []Test
	ExecTime uint
}

// ReadJSONFile return the struct append with data
func ReadJSONFile(file string, target interface{}) error {
	data, err := ioutil.ReadFile(file + ".json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, target)
	if err != nil {
		return err
	}
	return nil
}

func printResult(result Response) {
	fmt.Printf("{")
	fmt.Printf("\"completion:\" \"%d\",", result.Completion)
	fmt.Printf("\"results\":[")
	for i := range result.Results {
		fmt.Printf("{")
		fmt.Printf("\"toStdout\":%s,", result.Results[i].toStdout)
		fmt.Printf("\"toStderr\" :%s", result.Results[i].toStderr)
		if (i + 1) == len(result.Results) {
			fmt.Printf("}")
		} else {
			fmt.Printf("},")
		}
	}
	fmt.Printf("]}")
}

func (result *Result) makeResult(completion int, out string, test string, index int, exLen int) int {
	if completion >= 0 {
		if completion == 0 {
			result.toStdout = strconv.Itoa(index+1) + "/" + strconv.Itoa(exLen) + " : FAILED\n\tExpected " + test + "\n\tGot " + out + "\n"
			result.toStderr = "[" + out + "]\n"
		} else if completion == 1 {
			result.toStdout = strconv.Itoa(index+1) + "/" + strconv.Itoa(exLen) + " : PASSED\n"
			result.toStderr = "[" + out + "]\n"
			return (1)
		} else {
			result.toStdout = strconv.Itoa(index+1) + "/" + strconv.Itoa(exLen) + " : FAILED\n\t TIME OUT\n"
			result.toStderr = "[]\n"
		}
	} else {
		result.toStdout = strconv.Itoa(index+1) + "/" + strconv.Itoa(exLen) + " : Error Skiped\n"
		result.toStderr = "[]\n"
	}
	return (0)
}

func (test *Test) execProcess(execCmd string, execFile string, execTime uint, response *Response) (int, string) {
	// var sec = 40
	t := execTime * uint(time.Second)
	cnx, cancel := context.WithTimeout(context.Background(), time.Duration(t))
	defer cancel()
	cmd := exec.CommandContext(cnx, execCmd, execFile)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
		return -1, ""
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, test.Input)
	}()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, "TIME OUT"
	}
	if bytes.Compare(out, []byte(test.Output)) == 0 {
		return 1, string(out)
	} else {
		return 0, string(out)
	}
}

func runProcess(language string, pathProg string, pathTest string) {
	var exercises Exercise
	var resp Response
	resp.Completion = 0
	err := ReadJSONFile(pathTest, &exercises)
	exLen := len(exercises.Tests)
	if err != nil {
		fmt.Println(err)
		return
	}
	if language == "compiler" {
		language = pathProg
		pathProg = ""
	}
	for i := range exercises.Tests {
		completion, out := exercises.Tests[i].execProcess(language, pathProg, exercises.ExecTime, &resp)
		var result Result
		point := result.makeResult(completion, out, exercises.Tests[i].Output, i, exLen)
		resp.Completion += point
		resp.Results = append(resp.Results, result)
	}
	resp.Completion = (resp.Completion / exLen) * 100
	printResult(resp)
}

// ./testers [langage] => {php, node, python, compiler, etc..} [path to program] [path to testfile] [wich compiler]
func main() {
	args := os.Args[1:]
	if len(args) < 3 {
		fmt.Print("too feew argument for the program")
	} else {
		runProcess(args[0], args[1], args[2])
	}
}
