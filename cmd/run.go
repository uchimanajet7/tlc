package cmd

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var web bool

var runCmd = &cobra.Command{
	Use:   "run",
	Short: `Clean the list of "OneTab"`,
	Long:  `Clean the list of "OneTab"`,
	RunE:  execCmd,
}

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVarP(&web, "web", "w", false, "Check if you can access the website")
}

func execCmd(cmd *cobra.Command, args []string) error {
	log.Println("--- Start command ---")

	if len(args) <= 0 {
		fmt.Println()
		fmt.Println("Please specify file as argument")

		return cmd.Usage()
	}

	// read file
	fmt.Println()
	lines, err := readFile(args[0])
	if err != nil {
		return errors.Wrapf(err, "failed to read file: %s", args[0])
	}

	// create result list
	resultsList := make([][]string, 0, len(lines))
	errorsList := make([][]string, 0, len(lines))

	// merge with URL
	fmt.Println()
	urlList, urlErrList := mergeURL(lines)
	if len(urlList) <= 0 {
		return errors.New("valid URL did not exist")
	}
	if len(urlErrList) > 0 {
		errorsList = append(errorsList, []string{"# Merge with URL errors"})
		errorsList = append(errorsList, urlErrList...)
	}

	// merge with name
	fmt.Println()
	nameList, nameErrList := mergeName(urlList)
	if len(nameList) <= 0 {
		return errors.New("valid URL did not exist")
	}
	if len(nameErrList) > 0 {
		errorsList = append(errorsList, []string{"# Merge with name errors"})
		errorsList = append(errorsList, nameErrList...)
	}

	// check with web client
	if web {
		// check web in parallel
		fmt.Println()
		parallelList, parallelErrList := checkParallel(nameList)
		if len(parallelList) <= 0 {
			return errors.New("valid URL did not exist")
		}
		if len(parallelErrList) > 0 {
			errorsList = append(errorsList, []string{"# Check web in parallel errors"})
			errorsList = append(errorsList, parallelErrList...)
		}

		// if there are errors in parallel, check in series
		if len(parallelErrList) > 0 {
			// check web in serial
			fmt.Println()
			serialList, serialErrList := checkSerial(parallelErrList)

			if len(serialList) > 0 {
				// add last to simplify
				parallelList = append(parallelList, serialList...)
			}
			if len(serialErrList) > 0 {
				errorsList = append(errorsList, []string{"# Check web in serial errors"})
				errorsList = append(errorsList, serialErrList...)
			}
		}
		// add to results
		resultsList = append(resultsList, parallelList...)
	} else {
		// add to results
		resultsList = append(resultsList, nameList...)
	}

	// write result
	if len(resultsList) > 0 {
		fmt.Println()
		err := writeFile(filepath.Dir(args[0]), "merged_"+filepath.Base(args[0]), resultsList)
		if err != nil {
			return errors.New("failed to write result")
		}
	}
	if len(errorsList) > 0 {
		fmt.Println()
		err := writeFile(filepath.Dir(args[0]), "error_"+filepath.Base(args[0]), errorsList)
		if err != nil {
			return errors.New("failed to write error result")
		}
	}

	fmt.Println()
	log.Println("--- End command ---")

	return nil
}
