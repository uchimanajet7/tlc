package cmd

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/cheggaaa/pb.v1"
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

	newTest := make([][]string, 100)
	linesList, cnt := readFile2(args[0])
	fmt.Printf(" newTest: %+v\n", getTotal(newTest))
	fmt.Printf(" linesList: %+v\n", getTotal(linesList))
	fmt.Printf(" cnt: %+v\n", cnt)
	urlList2, urlErrList2 := mergeURL2(linesList)
	fmt.Printf(" urlList2: %+v\n", getTotal(urlList2))
	fmt.Printf(" urlErrList2: %+v\n", getTotal(urlErrList2))
	mergeName2(urlList2)
	checkParallel2(urlList2)
	checkSerial2(urlList2)

	// create result list
	resultList := make([]string, 0, len(lines))
	errorList := make([]string, 0, len(lines))

	// merge with URL
	fmt.Println()
	urlList, urlErrList := mergeURL(lines)
	if len(urlList) <= 0 {
		return errors.New("valid URL did not exist")
	}
	if len(urlErrList) > 0 {
		sort.Strings(urlErrList)
		errorList = append(errorList, "# Merge with URL errors")
		errorList = append(errorList, urlErrList...)
		errorList = append(errorList, "")
	}

	// merge with name
	fmt.Println()
	nameList, nameErrList := mergeName(urlList)
	if len(nameList) <= 0 {
		return errors.New("valid URL did not exist")
	}
	if len(nameErrList) > 0 {
		sort.Strings(nameErrList)
		errorList = append(errorList, "# Merge with name errors")
		errorList = append(errorList, nameErrList...)
	}

	if web {
		// check web in parallel
		fmt.Println()
		fmt.Println("Check web in parallel:")

		parallelList := make([]string, 0, len(nameList))
		parallelErrList := make([]string, 0, len(nameList))
		check := checkParallel(nameList)
		bar := pb.StartNew(len(nameList))
		for i := 0; i < len(nameList); i++ {
			bar.Increment()

			for key, value := range <-check {
				if value == 200 {
					parallelList = append(parallelList, key)
				} else {
					parallelErrList = append(parallelErrList, key)
				}
			}
		}
		bar.Finish()
		fmt.Printf(" result: %+v\n", len(parallelList))
		fmt.Printf(" error : %+v\n", len(parallelErrList))

		if len(parallelList) <= 0 {
			return errors.New("valid URL did not exist")
		}
		if len(parallelErrList) > 0 {
			sort.Strings(parallelErrList)
			errorList = append(errorList, "# Check web in parallel errors")
			errorList = append(errorList, parallelErrList...)
		}

		resultList = append(resultList, parallelList...)

		if len(parallelErrList) > 0 {
			// check web in serial
			fmt.Println()

			serialList, serialErrList := checkSerial(parallelErrList)

			if len(serialList) > 0 {
				resultList = append(resultList, serialList...)
			}
			if len(serialErrList) > 0 {
				sort.Strings(serialErrList)
				errorList = append(errorList, "# Check web in serial errors")
				errorList = append(errorList, serialErrList...)
			}
		} else {
			resultList = append(resultList, nameList...)
		}

		// write result
		if len(resultList) > 0 {
			fmt.Println()
			sort.Strings(resultList)

			err := writeFile(filepath.Dir(args[0]), "merged_"+filepath.Base(args[0]), resultList)
			if err != nil {
				return errors.New("failed to write result")
			}
		}
		if len(errorList) > 0 {
			fmt.Println()

			err := writeFile(filepath.Dir(args[0]), "error_"+filepath.Base(args[0]), errorList)
			if err != nil {
				return errors.New("failed to write error result")
			}
		}

		fmt.Println()
		log.Println("--- End command ---")
	}
	return nil
}
