package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"json"
	"os"
)

const (
	program = "gur"
	version = "0.0.1"
)

var (
	//rawurl = "https://localhost:80/"
	rawurl  = "https://aur.archlinux.org:443/"
	printf  = fmt.Printf
	println = fmt.Println
	sprintf = fmt.Sprintf
	// FIXME: change to final program name when decided. Use this so as not to give wrong userAgent
	//userAgent = sprintf("%v/%v", program, version)
	userAgent = "curl/7.21.4 (x86_64-unknown-linux-gnu) libcurl/7.21.4 OpenSSL/1.0.0d zlib/1.2.5"
	quiet     = flag.Bool("q", false, "only output package names")
	search    = flag.Bool("v", true, "search aur for packages")
	download  = flag.Bool("d", false, "download and extract tarball into working path")
	debug     = flag.Bool("dh", false, "debug http headers")
	dumpjson  = flag.Bool("dj", false, "dump json to stderr")
	aur       *Aur
)

func usage() {
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	var err os.Error
	aur, err = NewAur()
	handleError(err)
	defer aur.Close()
	if *download {
		*search = false
		doDownload()
		os.Exit(0)
	}
	if *search {
		doSearch()
		os.Exit(0)
	}
	usage()
}

//TODO: fix all the crazy err handling
func doDownload() {
	buf, err := getResults("info")
	handleError(err)
	err = checkInfoError(buf)
	handleError(err)
	info := new(Info)
	err = json.Unmarshal(buf, info)
	handleError(err)
	res, err := aur.GetTarBall(info.Results.URLPath)
	handleError(err)
	zbuf := new(bytes.Buffer)
	io.Copy(zbuf, res.Body)
	zip := NewZip()
	gzip, err := gzip.NewReader(zbuf)
	handleError(err)
	err = zip.Decompress("./", gzip)
	handleError(err)
}

func doSearch() {
	sr := new(SearchResults)
	buf, err := getResults("search")
	handleError(err)
	err = checkInfoError(buf)
	handleError(err)
	err = json.Unmarshal(buf, sr)
	handleError(err)
	for _, r := range sr.Results {
		println(r.Format())
	}
}

func checkInfoError(buf []byte) os.Error {
	info := new(Info)
	json.Unmarshal(buf, info)
	if info.Type == "error" {
		je := new(Error)
		err := json.Unmarshal(buf, je)
		if err != nil {
			return err
		}
		err = os.NewError(sprintf("gur: json %v", je.Results))
		return err
	}
	return nil
}

func getResults(method string) ([]byte, os.Error) {
	buf := new(bytes.Buffer)
	if len(flag.Args()) == 0 {
		err := os.NewError("no packages specified")
		handleError(err)
	}
	target := flag.Arg(0)
	res, err := aur.Method(method, target)
	if err != nil {
		return nil, err
	}
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		zr, err := gzip.NewReader(res.Body)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(buf, zr)
		handleError(err)
	default:
		_, err := io.Copy(buf, res.Body)
		handleError(err)
	}
	return buf.Bytes(), nil
}

func handleError(err os.Error) {
	if err != nil {
		printf("%v\n", err.String())
		os.Exit(1)
	}
}
