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

func mergeURL(lines []string) ([]string, []string) {
	fmt.Println("Merge with URL:")

	mergedList := make([]string, 0, len(lines))
	errorList := make([]string, 0, len(lines))
	urlMap := make(map[string]struct{})
	bar := pb.StartNew(len(lines))
	for i := range lines {
		bar.Increment()

		urlStr := getURL(lines[i])
		if urlStr != "" {
			_, exsist := urlMap[urlStr]
			if !exsist {
				urlMap[urlStr] = struct{}{}
				mergedList = append(mergedList, lines[i])
			} else {
				errorList = append(errorList, lines[i])
			}
		} else {
			errorList = append(errorList, lines[i])
		}
	}
	bar.Finish()

	fmt.Printf(" result: %+v\n", len(mergedList))
	fmt.Printf(" error : %+v\n", len(errorList))

	return mergedList, errorList
}

func mergeURL2(lines [][]string) ([][]string, [][]string) {
	fmt.Println("Merge with URL:")

	resultsList := make([][]string, 0, len(lines))
	results := make([]string, 0, getTotal(lines))
	errorsList := make([][]string, 0, len(lines))
	errors := make([]string, 0, getTotal(lines))
	setMap := make(map[string]struct{})

	bar := pb.StartNew(getTotal(lines))

	for i := range lines {
		line := lines[i]

		for j := range line {
			bar.Increment()

			urlStr := getURL(line[j])
			if urlStr != "" {
				_, exsist := setMap[urlStr]
				if !exsist {
					setMap[urlStr] = struct{}{}
					results = append(results, line[j])
				} else {
					errors = append(errors, line[j])
				}
			} else {
				errors = append(errors, line[j])
			}
		}
		if len(results) > 0 {
			resultsList = append(resultsList, results)
		}
		if len(errors) > 0 {
			errorsList = append(errorsList, errors)
		}
		results = nil
		errors = nil
	}
	bar.Finish()

	fmt.Printf(" result: %+v\n", getTotal(resultsList))
	fmt.Printf(" error : %+v\n", getTotal(errorsList))

	return resultsList, errorsList
}

func mergeName2(lines [][]string) ([][]string, [][]string) {
	fmt.Println("Merge with name:")

	resultsList := make([][]string, 0, len(lines))
	results := make([]string, 0, getTotal(lines))
	errorsList := make([][]string, 0, len(lines))
	errors := make([]string, 0, getTotal(lines))
	setMap := make(map[string]struct{})

	bar := pb.StartNew(getTotal(lines))

	for i := range lines {
		line := lines[i]

		for j := range line {
			bar.Increment()

			urlStr := getURL(line[j])
			if urlStr != "" {
				nameStr := strings.TrimSpace(strings.Replace(line[j], urlStr, "", -1))
				_, exsist := setMap[nameStr]
				if !exsist {
					setMap[nameStr] = struct{}{}
					results = append(results, line[j])
				} else {
					errors = append(errors, line[j])
				}
			} else {
				errors = append(errors, line[j])
			}
		}
		if len(results) > 0 {
			resultsList = append(resultsList, results)
		}
		if len(errors) > 0 {
			errorsList = append(errorsList, errors)
		}
		results = nil
		errors = nil
	}
	bar.Finish()

	fmt.Printf(" result: %+v\n", getTotal(resultsList))
	fmt.Printf(" error : %+v\n", getTotal(errorsList))

	return resultsList, errorsList
}

func mergeName(lines []string) ([]string, []string) {
	fmt.Println("Merge with name:")

	nameMap := make(map[string]int)
	bar := pb.StartNew(len(lines))
	for i := range lines {
		bar.Increment()

		urlStr := getURL(lines[i])
		if urlStr != "" {
			nameStr := strings.TrimSpace(strings.Replace(lines[i], urlStr, "", -1))
			value, exsist := nameMap[nameStr]
			if !exsist {
				nameMap[nameStr] = i
			} else {
				// adopt the shorter one
				if len(lines[value]) > len(lines[i]) {
					nameMap[nameStr] = i
				}
			}
		}
	}
	bar.Finish()

	mergedList := make([]string, 0, len(lines))
	indexMap := make(map[int]struct{})
	for _, value := range nameMap {
		mergedList = append(mergedList, lines[value])
		indexMap[value] = struct{}{}
	}

	errorList := make([]string, 0, len(lines))
	for i := range lines {
		if _, exsist := indexMap[i]; !exsist {
			errorList = append(errorList, lines[i])
		}
	}

	fmt.Printf(" result: %+v\n", len(mergedList))
	fmt.Printf(" error : %+v\n", len(errorList))

	return mergedList, errorList
}

func readFile2(path string) ([][]string, error) {
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
		if text == "" {
			if len(lines) > 0 {
				linesList = append(linesList, lines)
			}
			lines = nil
		} else {
			lines = append(lines, text)
		}
	}
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

func readFile(path string) ([]string, error) {
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
		return nil, errors.Wrapf(err, "can't file stat: %s", absPath)
	}
	fileSize := fileStat.Size()

	// create bar
	bar := pb.New(int(fileSize)).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10)
	bar.ShowSpeed = true
	bar.Start()

	// create proxy reader
	reader := bar.NewProxyReader(f)

	lines := make([]string, 0, 5000)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			lines = append(lines, scanner.Text())
		}
	}
	bar.Finish()

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to read line: %s", absPath)
	}
	if len(lines) <= 0 {
		return nil, errors.New("valid line did not exist")
	}
	fmt.Printf(" result: %+v\n", len(lines))

	return lines, nil
}

func writeFile(path string, name string, lines []string) error {
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
	bar := pb.StartNew(len(lines))
	for i := range lines {
		bar.Increment()

		_, err := writer.WriteString(lines[i] + "\n")
		if err != nil {
			return errors.Wrapf(err, "failed to write line: %s", lines[i])
		}
	}
	bar.Finish()
	writer.Flush()

	return nil
}

func checkParallel(lines []string) <-chan map[string]int {
	receiver := make(chan map[string]int)
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
	limit := make(chan int, cpus*6)

	for i := range lines {
		url := getURL(lines[i])
		go func(i int, url string) {
			limit <- 1
			defer func() { <-limit }()

			httpClient := &http.Client{}
			client, err := createClient(url, httpClient)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			httpRequest, _ := client.createRequest(ctx, "GET", "", nil)

			res, err := client.HTTPClient.Do(httpRequest)
			if err == nil {
				defer res.Body.Close()
				receiver <- map[string]int{lines[i]: res.StatusCode}
			} else {
				receiver <- map[string]int{lines[i]: -1}
			}
		}(i, url)
	}

	return receiver
}

func checkParallel2(lines [][]string) ([][]string, [][]string) {
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
	bar := pb.StartNew(getTotal(lines))

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
		bar.Finish()
	}()

	// Prepare the return value
	resultsList := make([][]string, 0, len(lines))
	errorsList := make([][]string, 0, len(lines))
	for range lines {
		resultsList = append(resultsList, make([]string, 0, getTotal(lines)))
		errorsList = append(errorsList, make([]string, 0, getTotal(lines)))
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

	fmt.Printf(" result: %+v\n", getTotal(resultsList))
	fmt.Printf(" error : %+v\n", getTotal(errorsList))

	return resultsList, errorsList
}

func checkSerial2(lines [][]string) ([][]string, [][]string) {
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

func checkSerial(lines []string) ([]string, []string) {
	fmt.Println("Check web in serial:")

	resultList := make([]string, 0, len(lines))
	errorList := make([]string, 0, len(lines))
	bar := pb.StartNew(len(lines))
	for i := range lines {
		bar.Increment()

		url := getURL(lines[i])
		httpClient := &http.Client{}
		client, err := createClient(url, httpClient)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		httpRequest, _ := client.createRequest(ctx, "GET", "", nil)

		res, err := client.HTTPClient.Do(httpRequest)
		if err == nil {
			defer res.Body.Close()

			if res.StatusCode == 200 {
				resultList = append(resultList, lines[i])
			} else {
				errorList = append(errorList, lines[i])
			}
		} else {
			errorList = append(errorList, lines[i])
		}
	}
	bar.Finish()

	fmt.Printf(" result: %+v\n", len(resultList))
	fmt.Printf(" error : %+v\n", len(errorList))

	return resultList, errorList
}
