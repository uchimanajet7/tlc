package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/cheggaaa/pb.v1"
)

func getTotal(lines [][]string) int {
	total := 0
	for i := range lines {
		total = total + len(lines[i])
	}
	return total
}

func getURL(line string) string {
	items := strings.Split(line, "|")
	if len(items) <= 0 {
		return ""
	}

	urlStr := strings.TrimSpace(items[0])
	_, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return ""
	}

	return urlStr
}

func mergeURL(lines [][]string) ([][]string, [][]string) {
	fmt.Println("Merge with URL:")

	total := getTotal(lines)
	resultsList := make([][]string, 0, len(lines))
	errorsList := make([][]string, 0, len(lines))
	setMap := make(map[string]struct{})

	bar := pb.StartNew(total)
	for _, line := range lines {
		results := make([]string, 0, total)
		errors := make([]string, 0, total)
		for _, text := range line {
			bar.Increment()

			urlStr := getURL(text)
			if urlStr != "" {
				_, exsist := setMap[urlStr]
				if !exsist {
					setMap[urlStr] = struct{}{}
					results = append(results, text)
				} else {
					errors = append(errors, text)
				}
			} else {
				errors = append(errors, text)
			}
		}
		if len(results) > 0 {
			resultsList = append(resultsList, results)
		}
		if len(errors) > 0 {
			errorsList = append(errorsList, errors)
		}
	}
	bar.Finish()

	fmt.Printf(" result: %+v\n", getTotal(resultsList))
	fmt.Printf(" error : %+v\n", getTotal(errorsList))

	return resultsList, errorsList
}

func mergeName(lines [][]string) ([][]string, [][]string) {
	fmt.Println("Merge with name:")

	total := getTotal(lines)
	resultsList := make([][]string, 0, len(lines))
	errorsList := make([][]string, 0, len(lines))
	setMap := make(map[string]struct{})

	bar := pb.StartNew(total)

	for _, line := range lines {
		results := make([]string, 0, total)
		errors := make([]string, 0, total)
		for _, text := range line {
			bar.Increment()

			urlStr := getURL(text)
			if urlStr != "" {
				nameStr := strings.TrimSpace(strings.Replace(text, urlStr, "", -1))
				_, exsist := setMap[nameStr]
				if !exsist {
					setMap[nameStr] = struct{}{}
					results = append(results, text)
				} else {
					errors = append(errors, text)
				}
			} else {
				errors = append(errors, text)
			}
		}
		if len(results) > 0 {
			resultsList = append(resultsList, results)
		}
		if len(errors) > 0 {
			errorsList = append(errorsList, errors)
		}
	}
	bar.Finish()

	fmt.Printf(" result: %+v\n", getTotal(resultsList))
	fmt.Printf(" error : %+v\n", getTotal(errorsList))

	return resultsList, errorsList
}

func readFile(path string) ([][]string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to Abs file path: %s", path)
	}
	fmt.Printf("Read file: %+v\n", absPath)

	f, err := os.Open(absPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open file: %s", absPath)
	}
	defer f.Close()

	// get file size
	fileStat, err := f.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get file stat: %s", absPath)
	}
	fileSize := fileStat.Size()

	// create bar
	bar := pb.New(int(fileSize)).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10)
	bar.ShowSpeed = true
	bar.Start()

	// create proxy reader
	reader := bar.NewProxyReader(f)

	lines := make([]string, 0, 5000)
	linesList := make([][]string, 0, 5000)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		// break on a blank line
		if text == "" {
			if len(lines) > 0 {
				sort.Strings(lines)
				linesList = append(linesList, lines)
			}
			lines = nil
		} else {
			lines = append(lines, text)
		}
	}
	// add values for the last loop
	if len(lines) > 0 {
		linesList = append(linesList, lines)
	}
	bar.Finish()

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to read line: %s", absPath)
	}
	total := getTotal(linesList)
	if total <= 0 {
		return nil, errors.New("valid line did not exist")
	}
	fmt.Printf(" result: %+v\n", total)

	return linesList, nil
}

func writeFile(path string, name string, lines [][]string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return errors.Wrapf(err, "failed to Abs file path: %s", path)
	}

	filePath := filepath.Join(absPath, name)
	fmt.Printf("Write file: %+v\n", filePath)

	f, err := os.Create(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file: %s", filePath)
	}

	defer f.Close()

	writer := bufio.NewWriter(f)
	bar := pb.StartNew(getTotal(lines))
	for _, line := range lines {
		sort.Strings(line)
		for j, text := range line {
			bar.Increment()

			writeStr := text + "\n"
			if len(line) == j+1 {
				// a blank line represents one unit
				writeStr = text + "\n\n"
			}
			_, err := writer.WriteString(writeStr)
			if err != nil {
				return errors.Wrapf(err, "failed to write line: %s", text)
			}
		}
	}
	bar.Finish()
	writer.Flush()

	return nil
}

func checkParallel(lines [][]string) ([][]string, [][]string) {
	fmt.Println("Check web in parallel:")

	var wg sync.WaitGroup

	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
	limit := make(chan int, cpus*6)

	type webResult struct {
		index  int
		text   string
		status int
	}
	webCheck := make(chan webResult)
	total := getTotal(lines)
	bar := pb.StartNew(total)

	go func() {
		for i, line := range lines {
			for _, text := range line {
				wg.Add(1)

				go func(i int, text string) {
					limit <- 1
					defer func() {
						<-limit
						wg.Done()
					}()
					bar.Increment()

					urlStr := getURL(text)
					statusCode := -1
					if urlStr != "" {
						httpClient := &http.Client{}
						client, err := createClient(urlStr, httpClient)

						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()

						httpRequest, _ := client.createRequest(ctx, "GET", "", nil)

						res, err := client.HTTPClient.Do(httpRequest)
						if err == nil {
							defer res.Body.Close()
							statusCode = res.StatusCode
						}
					}
					webCheck <- webResult{i, text, statusCode}
				}(i, text)
			}
		}
		wg.Wait()
		close(webCheck)
	}()

	// Prepare the return value
	resultsList := make([][]string, 0, len(lines))
	errorsList := make([][]string, 0, len(lines))
	for range lines {
		resultsList = append(resultsList, make([]string, 0, total))
		errorsList = append(errorsList, make([]string, 0, total))
	}

	// get channel value
	for {
		result, ok := <-webCheck
		if !ok {
			break
		}

		// check status code
		if result.status == 200 {
			resultsList[result.index] = append(resultsList[result.index], result.text)
		} else {
			errorsList[result.index] = append(errorsList[result.index], result.text)
		}
	}
	bar.Finish()

	fmt.Printf(" result: %+v\n", getTotal(resultsList))
	fmt.Printf(" error : %+v\n", getTotal(errorsList))

	return resultsList, errorsList
}

func checkSerial(lines [][]string) ([][]string, [][]string) {
	fmt.Println("Check web in serial:")

	resultsList := make([][]string, 0, len(lines))
	errorsList := make([][]string, 0, len(lines))

	total := getTotal(lines)
	bar := pb.StartNew(getTotal(lines))
	for _, line := range lines {
		results := make([]string, 0, total)
		errors := make([]string, 0, total)
		for _, text := range line {
			bar.Increment()

			urlStr := getURL(text)
			if urlStr != "" {
				httpClient := &http.Client{}
				client, err := createClient(urlStr, httpClient)

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				httpRequest, _ := client.createRequest(ctx, "GET", "", nil)

				res, err := client.HTTPClient.Do(httpRequest)
				if err == nil {
					defer res.Body.Close()

					if res.StatusCode == 200 {
						results = append(results, text)
					} else {
						errors = append(errors, text)
					}
				} else {
					errors = append(errors, text)
				}
			}
		}
		if len(results) > 0 {
			resultsList = append(resultsList, results)
		}
		if len(errors) > 0 {
			errorsList = append(errorsList, errors)
		}
	}
	bar.Finish()

	fmt.Printf(" result: %+v\n", getTotal(resultsList))
	fmt.Printf(" error : %+v\n", getTotal(errorsList))

	return resultsList, errorsList
}
