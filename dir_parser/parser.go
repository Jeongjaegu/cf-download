package dir_parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ibmjstart/cf-download/cmd_exec"
	"github.com/mgutz/ansi"
)

type Parser interface {
	ExecParseDir(readPath string) ([]string, []string)
	GetFailedDownloads() []string
}

type parser struct {
	cmdExec         cmd_exec.CmdExec
	appName         string
	instance        string
	onWindows       bool
	verbose         bool
	failedDownloads []string
}

func NewParser(cmdExec cmd_exec.CmdExec, appName, instance string, onWindows, verbose bool) *parser {
	return &parser{
		cmdExec:   cmdExec,
		appName:   appName,
		instance:  instance,
		onWindows: onWindows,
		verbose:   verbose,
	}
}

/*
*	execParseDir() uses os/exec to shell out commands to cf files with the given readPath. The returned
*	text contains file and directory structure which is then parsed into two slices, dirs and files. dirs
*	contains the names of directories in readPath, files contians the file names. dirs and files are returned
* 	to be downloaded by download() and downloadFile() respectively.
 */

func (p *parser) ExecParseDir(readPath string) ([]string, []string) {

	// make the cf files call using exec
	output, err := p.cmdExec.GetFile(p.appName, readPath, p.instance)
	dirSlice := strings.SplitAfterN(string(output), "\n", 3)

	// check for invalid or missing app
	if strings.Contains(dirSlice[1], "not found") {
		fmt.Println(createMessage("Error: "+p.appName+" app not found (check space and org)", "red+b", p.onWindows))
	}

	// p usually gets called when an app is not running and you attempt to download it.
	dir := dirSlice[2]
	if strings.Contains(dir, "error code: 190001") {
		fmt.Println(createMessage("App not found, or the app is in stopped state (This can also be caused by api failure)", "red+b", p.onWindows))
		check(err, "")
	}

	// handle an empty directory
	if strings.Contains(dir, "No files found") {
		return nil, nil
	} else {
		check(err, "Error E1: failed to read directory")
	}

	// directory inaccessible due to lack of permissions
	if strings.Contains(dirSlice[1], "FAILED") {
		messsage := createMessage(" Server Error: '"+readPath+"' not downloaded", "yellow", p.onWindows)

		p.failedDownloads = append(p.failedDownloads, messsage)

		if p.verbose {
			fmt.Println(messsage)
		}
		return nil, nil
	} else if strings.Contains(dirSlice[1], "status code: 502") {
		PrintSlice(dirSlice)
		//p.retryDirs = append(p.retryDirs, retryDir{ReadPath: readPath, WritePath: writePath})
	} else {
		// check for other errors
		check(err, "Error E1: failed to read directory")
	}

	// parse the returned output into files and dirs slices
	filesSlice := strings.Fields(dir)
	var files, dirs []string
	var name string
	for i := 0; i < len(filesSlice); i++ {
		if strings.HasSuffix(filesSlice[i], "/") {
			name += filesSlice[i]
			dirs = append(dirs, name)
			name = ""
		} else if isDelimiter(filesSlice[i]) {
			if len(name) > 0 {
				name = strings.TrimSuffix(name, " ")
				files = append(files, name)
			}
			name = ""
		} else {
			name += filesSlice[i] + " "
		}
	}
	return files, dirs
}

func (p *parser) GetFailedDownloads() []string {
	return p.failedDownloads
}

func isDelimiter(str string) bool {
	match, _ := regexp.MatchString("^[0-9]([0-9]|.)*(G|M|B|K)$", str)
	if match == true || str == "-" {
		return true
	}
	return false
}

func check(e error, errMsg string) {
	if e != nil {
		fmt.Println("\nError: ", e)
		if errMsg != "" {
			fmt.Println("Message: ", errMsg)
		}
		os.Exit(1)
	}
}

// prints slices in readable format
func PrintSlice(slice []string) error {
	for index, val := range slice {
		fmt.Println(index, ": ", val)
	}
	return nil
}

func createMessage(message, color string, onWindows bool) string {
	errmsg := ansi.Color(message, color)
	if onWindows == true {
		errmsg = message
	}

	return errmsg
}
