package main

// Generates a report of the distribution of file sizes
// that would live on the SAFE network
// assuming files > 1 MB are split into 1 MB chunks
// and files between 3 KB - 1 MB are split into 3 chunks
// and files < 3 KB are a single chunk
//
// This tool reports how many chunks there would be
// and what their distribution is.

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/user"
	"path"
	"sort"
	"strconv"
)

const OneKb = 1024
const OneMb = 1024 * 1024
const OneGb = 1024 * 1024 * 1024

func main() {
	fmt.Println("chunk_distribution v0.1.0")
	u, err := user.Current()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Gathering current user HomeDir stats")
	files := walkDir(u.HomeDir)
	reportSizes(files)
}

// returns all files from a director, including files in subdirectories
func walkDir(dirname string) []os.FileInfo {
	allFiles := []os.FileInfo{}
	files, _ := ioutil.ReadDir(dirname)
	for _, file := range files {
		if file.IsDir() {
			subdirFiles := walkDir(path.Join(dirname, file.Name()))
			allFiles = append(allFiles, subdirFiles...)
		} else {
			allFiles = append(allFiles, file)
		}
	}
	return allFiles
}

// prints out the details of the files
func reportSizes(files []os.FileInfo) {
	var gt int64
	var lt int64
	var totalChunks int64      // how many chunks of any size on this disk
	var largeChunks int64      // how many 1 MB chunks on this disk
	var smallChunks int64      // how many chunks smaller than 1 MB on this disk
	var largeGigabytes float64 // total gigabytes consumed by large files
	var smallGigabytes float64 // total gigabytes consumed by small files
	histogram := map[int64]int64{
		0:    0,
		100:  0,
		200:  0,
		300:  0,
		400:  0,
		500:  0,
		600:  0,
		700:  0,
		800:  0,
		900:  0,
		1000: 0,
	}
	for _, file := range files {
		size := file.Size()
		if size > OneMb {
			gt = gt + 1
			largeGigabytes = largeGigabytes + float64(size)/float64(OneGb)
			fileChunks := int64(math.Ceil(float64(size) / float64(OneMb)))
			totalChunks = totalChunks + fileChunks + 1                   // + 1 for datamap
			largeChunks = largeChunks + fileChunks - 1                   // - 1 for last chunk which is smaller
			smallChunks = smallChunks + 2                                // + 2 for last chunk plus datamap
			histogram = addToHistogram(histogram, 1024, fileChunks-1)    // large chunks
			histogram = addToHistogram(histogram, (size%OneMb)/OneKb, 1) // last chunk
			histogram = addToHistogram(histogram, 1, 1)                  // datamap
		} else {
			lt = lt + 1
			smallGigabytes = smallGigabytes + float64(size)/float64(OneGb)
			// files less than 3KB are chunked to a minimum of 3 chunks, each
			// chunk being 1/3 of the original file size.
			if size < 3*OneKb {
				totalChunks = totalChunks + 1 // + 1 for datamap with no chunks
				smallChunks = smallChunks + 1 // + 1 for datamap with no chunks
				histogram = addToHistogram(histogram, size/OneKb, 1)
			} else {
				totalChunks = totalChunks + 4                          // + 3 + 1 for 3 chunks plus datamap
				smallChunks = smallChunks + 4                          // + 3 + 1 for 3 chunks plus datamap
				histogram = addToHistogram(histogram, size/OneKb/3, 3) // chunks
				histogram = addToHistogram(histogram, 1, 1)            // datamap which is typically about 500 B
			}
		}
	}
	// stats
	fmt.Println("Total files:", len(files))
	fmt.Printf("Files larger than 1 MB: %v (%f GB)\n", gt, largeGigabytes)
	fmt.Printf("Files smaller than 1 MB: %v (%f GB)\n", lt, smallGigabytes)
	fmt.Println("Total chunks:", totalChunks)
	fmt.Println("Large chunks:", largeChunks)
	fmt.Println("Small chunks:", smallChunks)
	// histogram
	fmt.Println("\nChunk Size  Count")
	reportHistogram(histogram)
}

func addToHistogram(histogram map[int64]int64, size, count int64) map[int64]int64 {
	key := (size / 100) * 100
	_, exists := histogram[key]
	if !exists {
		fmt.Println("Missing key in histogram", key)
		histogram[key] = 0
	}
	histogram[key] = histogram[key] + count
	return histogram
}

func reportHistogram(h map[int64]int64) {
	sortedKeys := []int{}
	for key := range h {
		sortedKeys = append(sortedKeys, int(key))
	}
	sort.Ints(sortedKeys)
	for _, sortedKey := range sortedKeys {
		spacing := ""
		upperRange := "-" + strconv.Itoa(sortedKey+100)
		if sortedKey < 1 {
			spacing = "   "
			upperRange = "-" + strconv.Itoa(sortedKey+100) + " KB"
		} else if sortedKey < 900 {
			spacing = " "
			upperRange = "-" + strconv.Itoa(sortedKey+100) + "   "
		} else if sortedKey < 999 {
			spacing = " "
			upperRange = "-" + strconv.Itoa(sortedKey+100) + "  "
		} else {
			upperRange = "+      "
		}
		fmt.Printf(spacing+"%v%v %v\n", sortedKey, upperRange, h[int64(sortedKey)])
	}
}
