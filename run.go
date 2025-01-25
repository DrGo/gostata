package gostata

import (
	"os"
	"os/exec"
	"path"
	"strings"
)

/*
Running Stata from Bash
Stata(console) can be installed (from within Stata) by selecting Stata > Install Terminal Utility.... You can then start the console version by typing stata-se or stata-mp in a Terminal window. I alias it using alias st='stata-mp -q' to avoid annoying startup screen. Executable is usually at /usr/local/bin/stata-mp.
type stata-mp -h for usage
 usage:  stata-mp [-h -q -s -b] ["stata command"]
        where:
             -h      show this display
             -q      suppress logo, initialization messages
             -s      "batch" mode creating .smcl log
			 -b      "batch" mode creating .log file
			 -e 	 ? undocumented exit after running a do file

*/
const stataShellCommand = "stata-mp"

func RunStataDo(workDir, doFileName string) (output string, err error) {
	err = os.Chdir(  workDir) //Stata creates log file in this directory
	if err != nil {
		return "", err
	}
	cmdArgs := []string{"-q", "-e", doFileName}
	cmd := exec.Command(stataShellCommand, cmdArgs...)
	cmd.Dir= workDir 
	if err:=cmd.Run(); err != nil {
		return "", err
	}
	_, logFileName := path.Split(doFileName)
	ext := path.Ext(logFileName)
	logFileName = logFileName[0:len(logFileName)-len(ext)] + ".log"
	cmdOut, err := os.ReadFile(logFileName)
	if err != nil {
		return "", err
	}
	return string(cmdOut), nil
}

func GetKeyValuePairs(s string) map[string]string {
	lines := strings.Split(s, "\n")
	dict := make(map[string]string)
	if len(lines) == 0 {
		return dict
	}
	for _, line := range lines {
		if i := strings.Index(line, "="); i >= 0 {
			key, value := strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
			if key != "" && value != "" {
				dict[key] = value
			}
		}
	}
	return dict
}

func basename(s string) string {
	n := strings.LastIndexByte(s, '.')
	if n >= 0 {
		return s[:n]
	}
	return s
}
